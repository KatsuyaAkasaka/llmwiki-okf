---
name: query
description: >
  Answer questions from the llmwiki OKF knowledge bundle — local checkout if present,
  otherwise the hosted wiki over HTTP — with provenance, trust tier, and freshness attached
  to every answer. Use this whenever the user asks what the wiki/knowledge base knows,
  or asks a factual/project question in a repo that has a llmwiki.yaml — phrases like
  "what do we know about X", "wikiに聞いて", "check the knowledge base", "Xについてwikiで調べて" — even when they don't say "query". Prefer this over ad-hoc grepping: wiki pages
  carry provenance that raw grep results lack.
---

# llmwiki query — answer from the wiki, with receipts

The wiki is compiled knowledge: distilled once, kept current. Your job is to answer from it
**and say how much the answer can be trusted** — an answer without provenance is just
chat. If the wiki cannot answer, say so plainly; do not silently substitute your own
general knowledge for wiki content (offering it *labeled as your own* is fine).

## Step 1 — Find the wiki

Try in order; use the first that works:

1. **Local bundle**: search upward from the working directory for `llmwiki.yaml` →
   `bundle_dir` (default `wiki/`). Local wins — it is fresher than any deployment.
2. **Remote**: `remote_url` from `llmwiki.yaml`, else the `LLMWIKI_REMOTE_URL` environment
   variable, else a URL the user gave you. Fetch `<remote_url>/index.md`.
3. Neither → tell the user you need a bundle path or a wiki URL; don't guess.

## Step 2 — Navigate, don't scan

Start from `index.md` — it is the map: every page with a one-line description. Pick the
2–5 most relevant pages and read them (Read locally / fetch `<remote_url>/<path>.md`
remotely). Follow bundle-relative links (`/concepts/foo.md`) one or two hops when a page
points somewhere more specific. Locally, Grep is a good fallback when the index doesn't
surface an obvious page — remotely you only have the index and links, which is why
ingest keeps them complete.

## Step 3 — Answer with receipts

Structure the answer:

- **The answer**, synthesized across the pages you read.
- **Sources**: which wiki pages (as links — local paths or `<remote_url>/...`), and the
  original sources their frontmatter cites where relevant.
- **Trust & freshness**, derived from frontmatter (compact, e.g. one line per cited page):
  - no `verified` → *unverified* · `verified` by machine actors only → *machine-confirmed* ·
    any `human:` entry → *human-reviewed*
  - `status: draft`/`deprecated` — say so; prefer stable pages when they conflict.
  - `stale_after` in the past — warn that the page may be outdated.
- **Conflicts**: if pages disagree, present both positions with their sources. Never pick
  a side silently.
- **Gaps**: if the wiki only partially covers the question, name what's missing and suggest
  ingesting a source for it (`/llmwiki:ingest`).

## Step 4 — File good answers back (local only)

If the answer required synthesizing across 3+ pages and the synthesis feels durable
(would be asked again), offer to save it as a `synthesis/` page so next time it's one read
instead of N. Only with the user's yes: follow the ingest skill's authoring rules —
frontmatter, `sources` pointing at the pages you synthesized (bundle-relative), index.md
and log.md updates. Never write to a remote wiki; suggest doing it from the wiki's repo.
