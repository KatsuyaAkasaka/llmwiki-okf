---
name: lint
description: >
  Health-check the llmwiki OKF bundle: run the deterministic `llmwiki lint` CLI (OKF
  conformance, broken links, staleness) and layer on the semantic checks only an LLM can
  do — contradictions between pages, duplicate topics, index drift, knowledge gaps. Use this
  whenever the user asks to lint, check, validate, clean up, or audit the wiki — "wikiを
  lintして", "is the wiki healthy?", "check the knowledge base for problems", "wikiの矛盾を
  探して" — and after large ingests, before deploys, or periodically as wiki maintenance.
---

# llmwiki lint — keep the wiki healthy

Wikis rot in two ways: **mechanically** (broken links, missing frontmatter, stale dates) and
**semantically** (contradictions, duplicates, drift between index and reality). The CLI
catches the first kind deterministically; you catch the second kind. Run both — reporting
only CLI output when the user asked for a lint is half the job.

## Step 1 — Deterministic pass (CLI)

Locate the bundle via `llmwiki.yaml` (search upward; `bundle_dir`, default `wiki/`). Run:

```bash
llmwiki lint --format json
```

Errors are OKF conformance violations (§11) — these make the bundle non-conformant and
block CI, so they are never "minor". Warnings (broken links, stale pages, orphans,
unindexed pages, missing title/description) are quality debt. If the CLI is missing, tell
the user (`go install github.com/KatsuyaAkasaka/llmwiki-okf/cmd/llmwiki@latest`) and do
the conformance checks manually against `../../docs/okf-cheatsheet.md` — slower but valid.

## Step 2 — Semantic pass (you)

Read `references/semantic-checks.md` for the procedure. In short, working from `index.md`
and the pages themselves:

1. **Contradictions** — pages making incompatible claims about the same thing.
2. **Duplicates** — two pages that are really one topic (the top failure mode of ingestion).
3. **Index drift** — index descriptions that no longer match page content.
4. **Knowledge gaps** — pages that are linked to but don't exist yet, thin pages,
   heavily-cited sources with no source-summary page.
5. **Staleness triage** — for each `stale_after` violation: still true, needs re-verification,
   or should be `status: deprecated`?

Scale the depth to the wiki: read everything if it's small; for large bundles prioritize
pages the CLI flagged plus the most-linked pages, and say what you didn't inspect —
a lint report that silently skipped half the wiki is worse than one that says so.

## Step 3 — Report, then fix

Present one integrated report, worst first:

1. **Errors** (conformance) — file, rule, one-line fix each.
2. **Contradictions & duplicates** — the claims, the pages, your recommended resolution.
3. **Warnings & drift** — grouped, with counts.
4. **Gaps & staleness** — as suggestions (ingest X, verify Y, deprecate Z).

Offer to fix what has an unambiguous fix: frontmatter errors, broken links, unindexed
pages, index drift. Apply after the user agrees. Contradictions and duplicate merges need
a human decision on which claim/page wins — propose, don't decide, unless told to.
Re-run `llmwiki lint` after fixing to confirm clean, and record the session in `log.md`
(`* **Update**: lint fixes — ...`).
