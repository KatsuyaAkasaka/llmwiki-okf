# llmwiki-okf

LLM が維持する知識 wiki。

- **パターン**: [Karpathy の LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) —
  wiki はクエリのたびに知識を再導出する RAG インデックスではなく、LLM が構築・維持する
  **蓄積・複利する成果物(compounding artifact)**。
- **フォーマット**: [Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) —
  プレーンな Markdown + YAML フロントマターで、来歴(`sources`)、信頼(`generated`/`verified`)、
  鮮度(`status`/`stale_after`)を表現する。ベンダー非依存で、人間にも LLM にも読める。
- **インターフェース**: [Claude Code](https://claude.com/claude-code) スキル — `ingest`、`query`、`lint` —
  に加えて、スキャフォールドと適合性チェックを担う決定的な Go CLI。
- **閲覧**: 一次成果物は生の Markdown — LLM やエージェントはページを直接読む。
  人間には、同じ Markdown をクライアントサイドでレンダリングする自己完結の
  単一ファイルビューア(`index.html`)。ビルド工程も SSG もなし:
  bundle は任意の静的ファイルサーバーでそのまま配信できる。

## クイックスタート

```bash
# 1. CLI をインストール
go install github.com/KatsuyaAkasaka/llmwiki-okf/cmd/llmwiki@latest

# 2. wiki プロジェクトをスキャフォールド
llmwiki init my-wiki && cd my-wiki
# → llmwiki.yaml(設定)+ wiki/(OKF bundle: index.md、log.md、シードページ、ビューア)

# 3. Claude Code プラグインをインストール(claude 内で):
#    /plugin marketplace add KatsuyaAkasaka/llmwiki-okf
#    /plugin install llmwiki@llmwiki-okf
```

その後、wiki プロジェクト内の Claude Code で:

| 話しかける内容 | 何が起きるか |
|---|---|
| 「https://example.com/paper をwikiに取り込んで」 | `ingest` がソースを相互リンク付きページに蒸留し、来歴を記録して `index.md` + `log.md` を更新、`llmwiki lint` で自己検証する |
| 「rawを取り込んで」 | `ingest` が raw(デフォルト `~/wiki_raw`)に置かれたファイルのうち**未取り込み・変更分だけ**を一括で蒸留する(manifest の content hash で差分検出) |
| 「Xについて何を知ってる?」 | `query` が `index.md` → ページと辿り、**出典付きで**回答する: 根拠ページ、信頼層、鮮度 |
| 「wikiをlintして」 | `lint` が決定的な CLI チェックを実行し、その上に意味的チェック(矛盾・重複・ギャップ)を重ねて修正を提案する |

取り込みの入力は 4 経路: URL、個別ファイル、貼り付けテキスト/会話、そして
**raw ディレクトリ**。raw はファイルを置いておくだけの経路で、場所は
`llmwiki.yaml` の `raw_dir` → 環境変数 `LLMWIKI_RAW_DIR` → **`~/wiki_raw`**
の順で解決される。デフォルトが wiki プロジェクトの**外**なのは意図的 — wiki リポジトリは
git 管理されることがあり、生ソース(PDF・画像・チャットログ等)をその中に入れることは
推奨しない(`llmwiki init` も raw を作らない)。

`llmwiki raw` が各ファイルを `.llmwiki-manifest.json` の content hash と突き合わせて
`new` / `changed` / `ingested` / `missing` に分類するため、再実行してもライブラリ全体を
処理し直すことはない — 差分だけが処理される。取り込み後もファイルは移動・削除されない
(raw は不変の生ソース層)。manifest は wiki 側に置かれるので、同じ raw を複数の
wiki が独立に取り込める。

**手動メモは `manual/` サブディレクトリへ**。raw 直下は書籍・記事・エクスポート等の
「出典の明確なソース」置き場で、そこから蒸留したページには ingest が `verified`
(machine-confirmed)を付ける。一方 `manual/` 配下は自分のメモ・未確認の主張の置き場で、
蒸留されたページは **unverified** のまま残る — 内容を確認したら自分で
`verified: {by: human:<id>}` を足して human-reviewed に昇格させる。`llmwiki raw` は
各ファイルの `origin`(`manual` / `source`)を表示するので機械的に区別できる。

```
~/wiki_raw/
├── some-book-chapter.pdf   # source → 蒸留ページは verified(machine-confirmed)
└── manual/
    └── my-hypothesis.md    # manual → 蒸留ページは unverified
```

```bash
mkdir -p ~/wiki_raw
cp ~/Downloads/paper.pdf ~/wiki_raw/      # 置く
cd my-wiki && llmwiki raw               # 何が待っているか確認
# → new  paper.pdf
# あとは Claude Code で「rawを取り込んで」
```

**wiki の場所の解決**: スキルと CLI は作業ディレクトリから上方向に `llmwiki.yaml` を
探す。wiki プロジェクトの外で使うときは対象を明示する — CLI はパス引数
(`llmwiki update <dir>`、`llmwiki lint <bundle-dir>`)、スキルは会話で wiki のパスを
伝える(不明ならスキルが尋ねる)。

`query` スキルは**ローカルにcloneがなくても**使える: `llmwiki.yaml` の `remote_url` か
環境変数 `LLMWIKI_REMOTE_URL` でホスティング済み wiki を指定すれば、`index.md` と
ページを HTTP で取得する。プラグインを入れて URL を設定し、質問するだけ。

## bundle の構成

```
~/wiki_raw/                # 取り込み待ちファイルの置き場(プロジェクト外・デプロイされない)
my-wiki/
├── llmwiki.yaml           # bundle_dir、raw_dir、remote_url、actor、categories
├── .llmwiki-manifest.json # raw 取り込みの追跡(content hash、触れたページ)
└── wiki/                  # OKF bundle = アップロードする単位そのもの
    ├── index.html         # 自己完結ビューア(ハッシュルーティング SPA、CDN 依存なし)
    ├── index.md           # カタログ — 人間と LLM の両方が辿る地図
    ├── log.md             # 追記専用の履歴(新しい順)
    ├── concepts/ entities/ guides/ sources/ synthesis/
    └── references/        # 消えうるソースのオプトインアーカイブ
```

すべてのページは OKF フロントマター付きの Markdown ファイル。信頼は宣言ではなく導出される:
`verified` なし → *unverified*、機械 actor のみ → *machine-confirmed*、`human:` エントリあり →
*human-reviewed*。本プロジェクトが実装するフォーマットのサブセットは
[docs/okf-cheatsheet.md](docs/okf-cheatsheet.md)、設計の意思決定は [DESIGN.md](DESIGN.md) を参照。

## 仕組みの更新を既存 wiki に反映する

llmwiki-okf 側の仕組みが更新されたら、既存の wiki プロジェクトには `update` で反映する:

```bash
go install github.com/KatsuyaAkasaka/llmwiki-okf/cmd/llmwiki@latest   # CLI 本体の更新
llmwiki update <dir>      # 対象の wiki プロジェクトを指定して反映
llmwiki update            # wiki プロジェクトの中で実行する場合は引数不要
```

`update` が触るのは**ツール所有ファイルだけ**(現状はビューア `index.html`)。
`llmwiki.yaml` と wiki ページはユーザーの持ち物なので書き換えず、テンプレートに増えた
設定キーがあれば「note:」として報告する(未設定でもデフォルトで動く)。何度実行しても
安全(冪等)。スキルの更新は `/plugin marketplace update llmwiki-okf` → `/plugin update
llmwiki@llmwiki-okf` で別途行う。

## lint

```bash
llmwiki lint                  # 人間向け表示; 適合性エラーがあれば exit 1
llmwiki lint --format json    # CI と lint スキル用
llmwiki lint --strict         # 警告も失敗扱いにする
```

エラーは OKF §11 の適合性違反(フロントマターのパース可否、非空 `type`、
予約ファイルの構造、フィールド形状の不正)。警告は品質負債: リンク切れ、
`stale_after` 超過、orphan ページ、どの `index.md` にも載っていないページ、
title/description の欠落。矛盾・重複・ギャップの検出には判断が必要なので
lint **スキル**側の仕事 — CLI の JSON 出力の上に重ねて実行される。

## ビューア

`wiki/index.html` は `llmwiki init` が配置するもので、以後の生成工程は不要:
インライン CSS/JS のみで CDN 依存のない自己完結の 1 ファイルで、bundle の Markdown を
クライアントサイドでレンダリングする。ページを変更しても再ビルドするものはない —
ページの公開とは `.md` ファイルを書くことそのもの。

動作の仕組み:

- **ハッシュルーティング** — `index.html#/concepts/foo.md` が相対リクエストで
  `concepts/foo.md` を fetch してその場でレンダリングする。サイドバーは `index.md` から
  組み立てられる。任意のページへのディープリンクは共有可能な URL になる。
- **フロントマターパネル** — 各ページの OKF メタデータをバッジ表示する: `type`、`status`、
  導出された信頼層(*unverified / machine-confirmed / human-reviewed*)、タグ、そして
  `stale_after` 超過時の赤い *stale* バッジ。生のフロントマターは折りたたみの中に置かれる。
- **リンク書き換え** — 内部の `.md` リンクはビューア内リンク(`#/...`)になり、
  外部リンクは通常どおり開く。Markdown レンダリングは見出し・リスト・テーブル・
  コードフェンス・引用・OKF のソース脚注をカバーする。ダーク/ライトはシステムテーマに追従。

閲覧するには bundle ディレクトリを HTTP で配信する — ビューアはページを fetch するため、
`file://` で直接開いても動かない:

```bash
cd wiki && python3 -m http.server 8000
# → http://localhost:8000/index.html
```

ビューアは相対パスの fetch しか使わないので、bundle をどこから配信しても
同じ 2 つのビューがそのまま成立する:

- 人間: `<base-url>/index.html`(または `<base-url>/index.html#/path/page.md`)
- LLM・エージェント: `<base-url>/index.md`、および全ページの生 Markdown

デプロイ自体は意図的にスコープ外 — `wiki/` ディレクトリを好きな方法で静的ホスティングに
同期すればよい。`llmwiki lint --strict` をゲートにするのが望ましい。

## ライセンス

Apache-2.0
