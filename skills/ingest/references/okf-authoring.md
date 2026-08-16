# Page templates

Slugs: lowercase kebab-case, descriptive (`transformer-architecture.md`, not `page1.md`).
Timestamps: ISO-8601 UTC. Dates: `YYYY-MM-DD`. Replace `<actor>` with the configured actor.

## Concept page — `concepts/<slug>.md`

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

One-paragraph definition a reader (or model) can use without opening anything else.

## Key points

- Claim with attribution.[^attention-paper]
- Relationship to [self-attention](/concepts/self-attention.md) — say *how* they relate.

## Open questions

- Anything the sources left unresolved.

[^attention-paper]: Vaswani et al., Attention Is All You Need.
```

## Entity page — `entities/<slug>.md`

Same shape, `type: Entity`. First line: who/what it is. Sections that earn their place:
`## Role`, `## Related` (linked pages with the relationship stated in prose).

## Guide page — `guides/<slug>.md`

`type: Guide`. Numbered steps, prerequisites first, one fenced code block per command.
Guides go stale fastest — set `stale_after` (6–12 months out) so lint resurfaces them.

## Source summary — `sources/<slug>.md`

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

What this source is and why it was ingested.

## Key claims

- The distilled claims, briefly.

## Pages touched

- [LLM Wiki pattern](/concepts/llm-wiki-pattern.md) — created
- [Andrej Karpathy](/entities/andrej-karpathy.md) — updated
```

## Synthesis page — `synthesis/<slug>.md`

`type: Synthesis`. Only when insight spans **multiple sources** — a synthesis with one
source is a concept page in disguise. Cite the source-summary pages, not just raw URLs,
in `sources` (bundle-relative resource paths like `/sources/karpathy-llm-wiki.md`).

## index.md entry

```markdown
## Concepts

* [Transformer Architecture](/concepts/transformer-architecture.md) - Attention-based sequence architecture.
```

## log.md entry (newest date heading at the top of the file)

```markdown
## 2026-08-16

* **Creation**: Added [Transformer Architecture](/concepts/transformer-architecture.md) from Attention paper.
* **Update**: Extended [Self-Attention](/concepts/self-attention.md) with scaling discussion.
```
