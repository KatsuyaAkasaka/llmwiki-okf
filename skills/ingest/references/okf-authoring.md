# ページ雛形

スラグ: 小文字ケバブケースで内容がわかるもの(`transformer-architecture.md`、`page1.md` は不可)。
タイムスタンプ: ISO-8601 UTC。日付: `YYYY-MM-DD`。`<actor>` は設定された actor に置換する。

## コンセプトページ — `concepts/<slug>.md`

```markdown
---
type: Concept
title: Transformer Architecture
description: Attention-based sequence architecture underlying modern LLMs.
tags: [ml, architecture]
sources:
  - id: attention-paper
    resource: https://arxiv.org/abs/1706.03762
    title: Attention Is All You Need
    author: Vaswani et al.
generated: { by: <actor>, at: 2026-08-16T12:00:00Z }
---

読み手(人間もモデルも)が他のページを開かずに使える、1 段落の定義。

## Key points

- 出典付きの主張。[^attention-paper]
- [self-attention](/concepts/self-attention.md) との関係 — *どう*関係するかを書く。

## Open questions

- ソースが解決しなかった論点。

[^attention-paper]: Vaswani et al., Attention Is All You Need.
```

## エンティティページ — `entities/<slug>.md`

同じ構造で `type: Entity`。冒頭 1 行: それが誰/何か。置く価値のあるセクション:
`## Role`、`## Related`(リンク先ページとの関係を本文で述べる)。

## ガイドページ — `guides/<slug>.md`

`type: Guide`。番号付き手順、前提条件を最初に、コマンドは 1 つずつコードフェンスで。
ガイドは最も早く陳腐化するので `stale_after`(6〜12 ヶ月先)を必ず設定し、lint が
再浮上させられるようにする。

## ソース要約 — `sources/<slug>.md`

```markdown
---
type: Source Summary
title: "Karpathy: LLM Wiki gist"
description: Proposal for LLM-maintained compounding knowledge wikis.
tags: [knowledge-management]
sources:
  - id: origin
    resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
    author: Andrej Karpathy
generated: { by: <actor>, at: 2026-08-16T12:00:00Z }
---

このソースが何で、なぜ取り込んだか。

## Key claims

- 蒸留した主張を簡潔に。

## Pages touched

- [LLM Wiki pattern](/concepts/llm-wiki-pattern.md) — created
- [Andrej Karpathy](/entities/andrej-karpathy.md) — updated
```

## 統合ページ — `synthesis/<slug>.md`

`type: Synthesis`。洞察が**複数ソース**にまたがるときだけ — ソースが 1 つの synthesis は
コンセプトページの変装にすぎない。`sources` には生 URL ではなくソース要約ページを
bundle 相対パスで引用する(例 `/sources/karpathy-llm-wiki.md`)。

## index.md エントリ

```markdown
## Concepts

* [Transformer Architecture](/concepts/transformer-architecture.md) - Attention-based sequence architecture.
```

## log.md エントリ(最新の日付見出しをファイル先頭に)

```markdown
## 2026-08-16

* **Creation**: Added [Transformer Architecture](/concepts/transformer-architecture.md) from Attention paper.
* **Update**: Extended [Self-Attention](/concepts/self-attention.md) with scaling discussion.
```
