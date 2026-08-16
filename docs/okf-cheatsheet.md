# OKF v0.2 Core Cheatsheet

The subset of the Open Knowledge Format spec that llmwiki implements. Full spec:
https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md

## Bundle layout

- A bundle is a directory tree of markdown files. Every non-reserved `.md` file is a
  **concept document** and must have YAML frontmatter with a non-empty `type`.
- Reserved filenames (never concept documents):
  - `index.md` — catalog for its directory: section headings + `* [Title](link) - one-line description`.
    Only the bundle-root `index.md` may have frontmatter, and only `okf_version: "0.2"`.
  - `log.md` — update history, newest first: `## YYYY-MM-DD` headings with bullet entries
    prefixed `**Update**` / `**Creation**` / `**Deprecation**`.
- Non-`.md` files (html, images, archived sources) are outside OKF conformance — fine to include.

## Concept document frontmatter

```yaml
type: Concept                # REQUIRED, non-empty. llmwiki defaults per category:
                             #   Concept / Entity / Guide / Source Summary / Synthesis
title: Display Name          # recommended
description: One sentence.   # recommended
tags: [topic-a, topic-b]     # recommended
sources:                     # provenance; each entry needs `resource`
  - id: karpathy-gist        # optional stable key, used as footnote label [^karpathy-gist]
    resource: https://...    # URL | /bundle/path.md | relative path | scope descriptor
    title: LLM Wiki gist
    author: Andrej Karpathy
    last_modified: 2026-08-16
generated:                   # who/when produced this page; `by` required if present
  by: llmwiki/claude-fable-5
  at: 2026-08-16T12:00:00Z
verified:                    # absent=unverified; machine actors=machine-confirmed;
  - by: human:akasaka        # any human:<id> entry = human-reviewed
    at: 2026-08-16T12:00:00Z
status: stable               # draft | stable | deprecated (absent = stable)
stale_after: 2027-08-16      # date after which content should be re-checked
```

Actor convention: agents `producer/version` (e.g. `llmwiki/claude-fable-5`),
people `human:<id>`, automation `process:<id>`.

## Linking

- Prefer bundle-relative links: `[customers](/concepts/customers.md)` — stable across moves.
- Relative links (`./other.md`, `../entities/x.md`) are allowed.
- Links are undirected; express the relationship type in the surrounding prose.
- Attribute specific claims with footnotes tied to `sources[].id`:

```markdown
The wiki compounds knowledge instead of re-deriving it.[^karpathy-gist]

[^karpathy-gist]: Karpathy, LLM Wiki gist.
```

## Body conventions

Prefer structural markdown (headings, lists, tables, fenced code) over long prose.
Conventional optional headings: `# Schema`, `# Examples`.

## Conformance (§11) — what `llmwiki lint` enforces as errors

1. Every non-reserved `.md` has parseable YAML frontmatter.
2. Every frontmatter has a non-empty `type`.
3. Reserved files follow their required structure when present.

Consumers must tolerate unknown `type` values and unknown frontmatter keys — never
remove custom keys you didn't write.
