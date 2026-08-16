---
type: Concept
title: LLM Wiki Pattern
description: Treating a wiki as a compounding artifact an LLM maintains, instead of re-deriving knowledge per query.
tags: [knowledge-management, llm]
sources:
  - id: karpathy-gist
    resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
    title: llm-wiki gist
    author: Andrej Karpathy
generated: { by: llmwiki/init, at: 2026-08-16T00:00:00Z }
---

A knowledge base the LLM builds and maintains incrementally: sources are distilled once
into cross-linked pages, so answers accumulate instead of being reconstructed from raw
documents on every query.[^karpathy-gist]

## Key points

- Three layers: immutable raw sources → LLM-maintained wiki → schema (the rules, i.e. the
  llmwiki skills and this bundle's conventions).
- Three operations: **ingest** (distill a source into pages), **query** (answer from pages,
  with provenance), **lint** (find contradictions, rot, and gaps).
- Humans curate sources and ask questions; the LLM does the bookkeeping it never tires of —
  cross-references, index updates, revision logging.
- Origin and rationale: [Karpathy: LLM Wiki gist](/sources/karpathy-llm-wiki.md).

[^karpathy-gist]: Karpathy, llm-wiki gist.
