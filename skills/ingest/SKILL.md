---
name: ingest
description: >
  Ingest a source (URL, local file, pasted text, or the current conversation) into the
  OKF-format llmwiki knowledge bundle: distill it into wiki pages, cross-link them, record
  provenance, and update index.md/log.md. Use this whenever the user wants to add, capture,
  save, or "remember" knowledge in their wiki — phrases like "ingest this article", "add this
  to the wiki", "この記事をwikiに取り込んで", "この会話の知見を保存して" — even if they
  don't say "ingest". If a llmwiki.yaml exists in the project, any request to store knowledge
  should go through this skill rather than ad-hoc file writes.
---

# llmwiki ingest — distill a source into the wiki

You are maintaining a **compounding knowledge artifact**, not writing notes. A source is
read once, distilled into interconnected pages, and never needs re-reading. Sloppy ingestion
(missing provenance, no cross-links, stale index) silently degrades every future query, so
follow the whole workflow — the bookkeeping is the product.

## Step 0 — Locate the bundle

Search upward from the working directory for `llmwiki.yaml`. It gives you:

- `bundle_dir` — the OKF bundle root (default `wiki/`)
- `actor` — the value for `generated.by` (fall back to `llmwiki/<your-model-id>`)
- `categories` — the category directories and their meanings

If there is no `llmwiki.yaml`, stop and tell the user to run `llmwiki init` first.

Read the bundle's `index.md` before writing anything — you need to know what already
exists to update rather than duplicate. If a page on the topic already exists, **update it**;
creating a near-duplicate page is the main failure mode of wiki rot.

For frontmatter and linking rules, read `../../docs/okf-cheatsheet.md` (relative to this
skill's base directory). For page templates, read `references/okf-authoring.md`.

## Step 1 — Read the source

- **URL** → WebFetch. Record the URL, author and date if visible.
- **Local file** → Read (images too — describe what they show; mark interpretations as inferred).
- **Pasted text / current conversation** → use it directly. The source `resource` becomes a
  scope descriptor like `conversation 2026-08-16 with human:akasaka`.

## Step 2 — Create the source summary page

Every ingested source gets exactly one page in `sources/<slug>.md` with
`type: Source Summary`: what the source is, its key claims, and links to the pages it
touched. This is the wiki's memory of *where knowledge came from* — queries cite it, and
re-ingesting the same source becomes an update instead of a duplicate. Check `sources/`
for an existing page for this source first.

## Step 3 — Distill into concept pages

Identify every concept, entity, guide-worthy procedure, and cross-cutting insight the source
contains. For each one, decide: update an existing page, or create a new one in the right
category. One source legitimately touches many pages — 5–15 edits is normal, not excessive.

On every page you touch:

- Frontmatter per the cheatsheet: `type` (required), `title`, `description`, `tags`,
  `sources` (append this source with an `id`), `generated` (this actor, now, ISO-8601).
- Attribute specific claims with footnotes: `claim.[^source-id]`
- Cross-link related pages with bundle-relative links (`/concepts/foo.md`) — in **both
  directions**. A page nobody links to is invisible to queries (and `lint` flags it as an orphan).
- If the source **contradicts** an existing page, do not silently overwrite: state both claims
  in the page with their sources, and flag the conflict in your final report to the user.
- Never delete or rewrite `verified` entries — verification belongs to whoever verified.
  If your edit invalidates what was verified, remove nothing; add the new content, keep the
  old `verified` list, and mention in the report that re-verification is needed.

## Step 4 — Update the catalog and the log

- `index.md` (bundle root, and any per-directory index a touched page lives under):
  every new page gets a `* [Title](/path.md) - one-line description` entry in the right section.
- `log.md`: add entries under today's `## YYYY-MM-DD` heading (create it at the **top** —
  newest first): `* **Creation**: ...` / `* **Update**: ...` with links.

## Step 5 — Self-verify

Run `llmwiki lint --format json` and fix any errors you introduced (warnings: fix broken
links and unindexed pages; report the rest). If the CLI is not installed, say so and
re-check your own edits against the cheatsheet's conformance rules instead.

## Step 6 — Archive only on request

Do **not** copy source content into the bundle by default — provenance lives in
`sources[].resource`. Only when the user explicitly asks to archive (ephemeral page,
conversation, paste), save the original into `references/` under the bundle root.
If the original is a `.md` file, save it as `<name>.md.txt` so it isn't parsed as a
concept document.

## Report

End with a short report: pages created / updated (as links), conflicts found,
lint result, and anything needing human verification.
