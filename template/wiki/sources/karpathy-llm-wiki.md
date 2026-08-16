---
type: Source Summary
title: "Karpathy: LLM Wiki gist"
description: Proposal for LLM-maintained compounding knowledge wikis; origin of this wiki's design.
tags: [knowledge-management]
sources:
  - id: origin
    resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
    title: llm-wiki gist
    author: Andrej Karpathy
generated: { by: llmwiki/init, at: 2026-08-16T00:00:00Z }
---

The gist that proposed the pattern this wiki implements: instead of RAG re-deriving
synthesis on every query, the LLM maintains the wiki as a persistent artifact.

## Key claims

- Human wikis fail because maintenance cost exceeds perceived value; LLMs remove that cost.
- Ingesting one source legitimately touches many pages (10–15 is normal).
- `index.md` (catalog) and `log.md` (history) make the wiki navigable and auditable.

## Pages touched

- [LLM Wiki Pattern](/concepts/llm-wiki-pattern.md) — created
