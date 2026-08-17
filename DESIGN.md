# llmwiki-okf 設計書

Karpathy の [LLM Wiki パターン](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f)を
[Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)
準拠で実装する OSS。Claude Code をインターフェースとし、成果物は GCS 等の静的ホスティングにそのまま置ける。

本書は grilling(実装前インタビュー)で確定した決定事項の記録である。

## 1. コンセプト

- wiki は「毎回再導出する RAG」ではなく **蓄積・複利する成果物(compiled artifact)**。
- LLM が ingest(取り込み)/ query(照会)/ lint(健全性検査)を通じて wiki を維持する。
- 人間はソースの選定と問いかけを担い、LLM が相互参照・更新・矛盾検出の簿記を担う。

## 2. 決定事項サマリ

| # | 論点 | 決定 | 主な理由 |
|---|---|---|---|
| 1 | 配布形態 | モノレポ(Claude Code プラグイン + Go CLI + wiki テンプレート) | LLM 判断が要る操作はスキル、決定的な操作は CLI に分離。1 リポジトリで導入体験を完結 |
| 2 | OKF 準拠度 | コア準拠のみ。Attested Computation(§10)は非対応 | 知識 wiki 用途に不要。適合性ルール(§11)はコアのみで完全に満たせる |
| 3 | 配信形態 | 生 Markdown + 自己完結ビューア SPA(index.html 1 枚) | LLM 消費者には生 md が一次ソース。ビルドレスで deploy = アップロードのみ |
| 4 | query 対象 | ローカル bundle 優先、無ければ remote_url へフォールバック | clone 不要の読み取り専用ユーザーにも query スキルだけで配布可能 |
| 5 | lint 設計 | 2 層: 決定的チェック = Go CLI、意味的チェック = LLM スキル | CI ゲートは決定的に、矛盾・ギャップ検出は LLM に |
| 6 | スキル構成 | ingest / query / lint の 3 スキルのみ。init は CLI | LLM 判断が要る操作だけをスキル化 |
| 7 | CLI 言語 | Go(単一バイナリ、依存は yaml.v3 のみ) | ランタイム依存ゼロで配布。go install / release バイナリ |
| 8 | デプロイ | **スコープ外**。利用者が各自 GitHub Actions 等で実施 | 成果物が「任意の静的ホスティングで動く」ことのみ保証(ビューアは相対パス fetch) |
| 9 | ingest 入力 | URL / ローカルファイル / 貼り付け / 会話コンテキストの任意入力 | LLM wiki の主要ユースケースを全てカバー |
| 10 | 生ソース保存 | デフォルト保存なし。指示時のみ `wiki/references/` にアーカイブ | バンドル肥大と公開バケットでの著作権リスクを回避 |
| 11 | 命名 | リポジトリ `llmwiki-okf`、CLI バイナリ / プラグイン名 `llmwiki` | スキル呼び出しは `/llmwiki:ingest` 等 |
| 12 | ライセンス | Apache-2.0 | OKF 本家と同一。仕様引用・企業利用に強い |
| 13 | raw 取り込み | raw ディレクトリのファイルを ingest の追加経路として一括取り込み。解決順: `llmwiki.yaml` の `raw_dir` → `$LLMWIKI_RAW_DIR` → **`~/wiki_raw`**。`.llmwiki-manifest.json`(wiki 側)の content hash で差分検出 | [Ar9av/obsidian-wiki](https://github.com/Ar9av/obsidian-wiki) の manifest 方式を参考。デフォルトを wiki プロジェクトの**外**にするのは、git 管理されうるリポジトリに生ソースを入れないため。init も raw を作らない。2 回目以降は差分のみ処理、ファイルは移動・削除しない |

## 3. リポジトリ構成

```
llmwiki-okf/
├── .claude-plugin/plugin.json   # プラグイン定義(name: llmwiki)
├── skills/
│   ├── ingest/SKILL.md          # + references/okf-authoring.md(ページ雛形・執筆規約)
│   ├── query/SKILL.md
│   └── lint/SKILL.md            # + references/semantic-checks.md(意味的チェック手順)
├── cmd/llmwiki/main.go          # CLI エントリポイント(init / lint / version)
├── internal/
│   ├── lint/                    # 決定的 lint 実装 + テスト
│   └── scaffold/                # init(埋め込みテンプレート展開)
├── embed.go                     # //go:embed all:template
├── template/                    # `llmwiki init` が展開する雛形
│   ├── llmwiki.yaml
│   └── wiki/                    # OKF bundle ルート(= アップロード単位)
│       ├── index.html           # ビューア SPA
│       ├── index.md             # OKF カタログ(okf_version: "0.2")
│       ├── log.md               # 更新履歴(新しい順)
│       ├── concepts/ entities/ guides/ sources/ synthesis/
│       └── references/          # オプトインのアーカイブ置き場
├── docs/okf-cheatsheet.md       # OKF コア準拠の要点(スキルから参照)
├── DESIGN.md / README.md / LICENSE
└── .github/workflows/ci.yml     # go test + テンプレート bundle の自己 lint
```

## 4. wiki プロジェクトの構成(init 後)

```
~/wiki_raw/                      # 取り込み待ちファイルの置き場(生ソース層、プロジェクト外)
my-wiki/
├── llmwiki.yaml                 # 設定(下記)
├── .llmwiki-manifest.json       # raw 取り込みの追跡(content hash / 触れたページ)
└── wiki/                        # OKF bundle = 静的ホスティングに置く単位
```

raw を wiki プロジェクトの外に置くのは意図的: my-wiki は git 管理されうるので、
生ソース(PDF・画像・チャットログ等)をその中に入れることは推奨しない。manifest は
wiki 側にあるため、同じ raw を複数の wiki が独立に取り込める。

`llmwiki.yaml`:

```yaml
bundle_dir: wiki                 # bundle ルートの相対パス
raw_dir: ""                    # 未設定なら $LLMWIKI_RAW_DIR → ~/wiki_raw の順で解決
remote_url: ""                   # 公開 URL(例 https://storage.googleapis.com/my-wiki)
actor: ""                        # generated.by の既定値(例 llmwiki/claude-fable-5)
categories:
  concepts: 理論・概念・メンタルモデル
  entities: 人物・組織・ツール・プロジェクト
  guides: 手順・ハウツー
  sources: 取り込んだソース単位の要約ページ
  synthesis: 複数ソース横断の統合分析
```

- カテゴリはデフォルト提案であり自由に増減可(OKF はディレクトリ構造を規定しない)。
- `sources/` は「ソース 1 件 = 1 ページの要約」(Karpathy の references 層)。
  OKF §6 の `references/` 規約(外部資料・コード置き場)と名前が衝突するため、
  アーカイブ用途の `wiki/references/` とは明確に区別する。
- アーカイブ保存時、元が `.md` のファイルは `.md.txt` として保存する
  (§11「全ての非予約 .md はフロントマター必須」との衝突を避けるため)。

## 5. OKF 準拠仕様(コア)

各コンセプト文書のフロントマター:

```yaml
type: Concept                    # 必須・非空。値は自由(Concept / Entity / Guide / Source Summary / Synthesis を既定提案)
title: 表示名                     # 推奨
description: 一文サマリ           # 推奨
tags: [a, b]                     # 推奨
sources:                         # 来歴。URL / bundle 相対パス / スコープ記述
  - id: src-1                    # 脚注 [^src-1] で本文中の主張に対応付け
    resource: https://...
    title: ...
    author: ...
    last_modified: 2026-08-16
generated: { by: <actor>, at: <ISO-8601> }   # by 必須(generated を書く場合)
verified:  [{ by: human:alice, at: ... }]    # 無し=unverified / 非human=machine-confirmed / human=human-reviewed
status: draft | stable | deprecated           # 省略時 stable
stale_after: 2027-01-01                       # 鮮度期限
```

- リンクは bundle 相対(`/concepts/foo.md`)を推奨(移動に強い)。
- `index.md` はカタログ(セクション見出し + リンク + 一行説明)。ルートのみ `okf_version: "0.2"`。
- `log.md` は新しい順の更新履歴(`## YYYY-MM-DD` + `**Update**` 等の太字プレフィックス)。
- actor 規約: エージェント `producer/version`、人間 `human:<id>`、プロセス `process:<id>`。

### wiki プロジェクトの発見ルール(スキル・CLI 共通)

1. 作業ディレクトリから上方向に `llmwiki.yaml` を探す(プロジェクト文脈が最優先)
2. 見つからなければ環境変数 `LLMWIKI_WIKI_DIR`(デフォルト wiki のプロジェクトルート)
3. どちらもなければエラー(init か環境変数設定を案内)

スキルは wiki と無関係なリポジトリで発火することが多いため、2 のフォールバックがないと
「取り込み先の wiki が見つからない」状態になる。これが env を用意する理由である。

## 6. CLI 仕様(`llmwiki`)

### `llmwiki init [dir]`
埋め込みテンプレートを展開。既存ファイルは上書きしない。

### `llmwiki raw [--format json]` / `llmwiki raw mark <file> [--pages a.md,b.md]`
raw の各ファイルを manifest の content hash と比較して分類する:
`new`(未取り込み)/ `changed`(取り込み後に変更)/ `ingested`(処理済み)/
`missing`(manifest にあるがファイル消失)。`mark` は取り込み完了の宣言で、
現在のハッシュ・時刻・触れたページを `.llmwiki-manifest.json` に記録する
(キーは raw 相対なので raw の場所を変えても追跡が生きる)。
raw の場所は `raw_dir`(yaml)→ `$LLMWIKI_RAW_DIR` → `~/wiki_raw` の順で解決。
発見と記録だけを CLI が決定的に担い、蒸留は ingest スキルの仕事(役割分担は lint と同型)。

### `llmwiki lint [--format json] [--strict] [dir]`
bundle ルート(llmwiki.yaml の bundle_dir、または指定ディレクトリ)に対し決定的チェックを実行。

**error(適合性違反 → exit 1)**
| ルール | 内容 |
|---|---|
| frontmatter-parse | 非予約 .md の YAML フロントマターがパース不能・欠落 |
| missing-type | `type` が欠落・空 |
| invalid-status | `status` が draft/stable/deprecated 以外 |
| invalid-generated | `generated` に `by` が無い |
| invalid-sources | `sources[]` エントリに `resource` が無い |
| invalid-date | `stale_after` / `usage_window` 等が YYYY-MM-DD でない |
| reserved-structure | ルート index.md に okf_version 以外のフロントマター、log.md の日付見出し不正 |

**warning(品質問題 → 通常 exit 0、`--strict` で exit 1)**
| ルール | 内容 |
|---|---|
| broken-link | bundle 内リンク(`/x.md`・相対)の切れ |
| stale | `stale_after` を経過 |
| missing-title / missing-description | 推奨フィールド欠落 |
| orphan | どのページ・index からもリンクされていない |
| unindexed | どの index.md にも載っていないコンセプト文書 |

JSON 出力(スキル・CI 用):

```json
{
  "bundle": "wiki",
  "errors":   [{"file": "concepts/foo.md", "rule": "missing-type", "message": "..."}],
  "warnings": [{"file": "concepts/bar.md", "rule": "broken-link", "message": "..."}],
  "stats": {"pages": 12, "links": 34}
}
```

## 7. スキル仕様

3 スキルとも冒頭で `llmwiki.yaml` を探索(cwd から上方向)して bundle 位置と設定を得る。

### `/llmwiki:ingest <source>`
1. ソース読解(URL は WebFetch、ファイルは Read、貼り付け・会話はそのまま)。
   **raw モード**(ソース未指定 or「rawを取り込んで」): `llmwiki raw --format json`
   で差分を得て、`new`/`changed` の各ファイルに以下を適用 → 処理ごとに
   `llmwiki raw mark` で記録。ファイルは移動・削除しない
2. `sources/<slug>.md`(type: Source Summary)を作成
3. 影響するページを洗い出し、既存ページ更新 or 新規作成(1 ソースで複数ページに波及してよい)
4. 全ページの frontmatter に sources / generated を記録、相互リンクを張る
5. index.md 更新、log.md に追記(新しい順)
6. `llmwiki lint` を実行して自己検証、結果を報告
7. 明示指示があった場合のみ原本を `wiki/references/` にアーカイブ

### `/llmwiki:query <question>`
1. ローカル bundle があれば index.md → Grep/Read で該当ページを辿って回答
2. 無ければ `remote_url`(llmwiki.yaml → 環境変数 LLMWIKI_REMOTE_URL → ユーザー指定)から
   `index.md` を fetch → 該当ページを fetch して回答
3. 回答には出典ページと信頼層(unverified / machine-confirmed / human-reviewed)・鮮度を添える
4. 良い統合回答が生まれ、かつローカル書き込みが可能なら `synthesis/` への保存を提案

### `/llmwiki:lint`
1. `llmwiki lint --format json` を実行(決定的チェック)
2. LLM による意味的チェック: ページ間の矛盾、重複トピック、index との乖離、知識ギャップ、
   stale ページの更新要否
3. 統合レポートを提示し、自動修正可能なもの(リンク切れ・index 追記等)は承認を得て修正
4. 修正した場合は log.md に記録

## 8. ビューア SPA

- `wiki/index.html` 1 ファイル・自己完結(外部 CDN 依存なし)。
- ハッシュルーティング: `index.html#/concepts/foo.md` → 相対 fetch → クライアントサイドで md レンダリング。
- 相対パスのみ使用するため、GCS 直 URL・独自ドメイン・任意の静的ホスティングで動く。
- フロントマターはメタデータパネル(type / tags / status / 信頼層 / sources)として表示。
- `.md` へのリンクはビューア内リンク(`#/...`)に変換。ダークモード対応(prefers-color-scheme)。

## 9. スコープ外(将来課題)

- GCS へのデプロイ自動化(利用者が各自 GitHub Actions で実施。bundle_dir をそのまま同期すればよい)
- Attested Computation(OKF §10)
- 非公開 bundle の認証付き query(IAP / signed URL)
- 全文検索インデックス(現状は index.md + Grep / fetch で十分な規模を想定)
