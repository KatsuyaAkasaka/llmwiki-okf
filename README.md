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
| 「inboxを取り込んで」 | `ingest` が `inbox/` に置かれたファイルのうち**未取り込み・変更分だけ**を一括で蒸留する(manifest の content hash で差分検出) |
| 「Xについて何を知ってる?」 | `query` が `index.md` → ページと辿り、**出典付きで**回答する: 根拠ページ、信頼層、鮮度 |
| 「wikiをlintして」 | `lint` が決定的な CLI チェックを実行し、その上に意味的チェック(矛盾・重複・ギャップ)を重ねて修正を提案する |

取り込みの入力は 4 経路: URL、個別ファイル、貼り付けテキスト/会話、そして
**inbox ディレクトリ**。inbox はファイルを `inbox/` に置いておくだけの経路で、
`llmwiki inbox` が `.llmwiki-manifest.json` の content hash と突き合わせて
`new` / `changed` / `ingested` / `missing` に分類するため、再実行してもライブラリ全体を
処理し直すことはない — 差分だけが処理される。取り込み後もファイルは移動・削除されない
(inbox は不変の生ソース層)。

```bash
cp ~/Downloads/paper.pdf my-wiki/inbox/   # 置く
llmwiki inbox                             # 何が待っているか確認
# → new  inbox/paper.pdf
# あとは Claude Code で「inboxを取り込んで」
```

`query` スキルは**ローカルにcloneがなくても**使える: `llmwiki.yaml` の `remote_url` か
環境変数 `LLMWIKI_REMOTE_URL` でホスティング済み wiki を指定すれば、`index.md` と
ページを HTTP で取得する。プラグインを入れて URL を設定し、質問するだけ。

## bundle の構成

```
my-wiki/
├── llmwiki.yaml           # bundle_dir、inbox_dir、remote_url、actor、categories
├── .llmwiki-manifest.json # inbox 取り込みの追跡(content hash、触れたページ)
├── inbox/                 # 取り込み待ちファイルの置き場(デプロイされない)
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
