# Semantic check procedure

These checks need judgment, which is why they live here and not in the CLI. Work from
evidence — quote the conflicting sentences, name the files — so the user can decide fast.

## 1. Contradictions

Group pages by topic (index sections and shared tags are your grouping hints). Within a
group, compare concrete claims: numbers, dates, "X is/does Y" statements, recommendations.
A real contradiction is two pages a reader could act on differently — not a difference in
emphasis or scope.

For each: quote both claims, check both pages' `sources` and `generated.at` (newer +
better-sourced usually wins), then recommend: update the losing page, or keep both claims
side-by-side in one page with their sources when the sources genuinely disagree.

## 2. Duplicates

Scan `index.md` for entries whose titles/descriptions overlap, and pages sharing most of
their tags. Read suspects. Two pages are duplicates when a reader needing one would always
need the other. Recommend a merge direction: keep the better slug, fold content in, merge
`sources` (dedupe by resource, keep all `verified` entries — they're attestations), and turn
the losing file into a link from every page that pointed at it (update, don't just delete;
the CLI's broken-link check will confirm you got them all).

## 3. Index drift

For each `index.md` entry, does the one-line description still match the page's current
`description`/content? Ingest updates pages more often than it rewrites index lines, so
drift accumulates silently. Fix lines that would mislead someone choosing what to read.

## 4. Knowledge gaps

- Links pointing at pages that don't exist yet (the CLI lists these as broken-link warnings —
  the semantic question is *which are worth creating*).
- Pages cited as `sources` by 3+ pages that have no `sources/` summary page of their own.
- Thin pages: a title and one line where the topic clearly deserves more — flag, don't pad.

Output gaps as a prioritized "worth ingesting/writing next" list — this is the wiki asking
for its next meal, and the most valuable part of the report for an active user.

## 5. Staleness triage

For each page past `stale_after` (CLI lists them), read it and sort into:

- **Still true** → recommend re-verification (`verified` entry) and a new `stale_after`.
- **Probably outdated** → recommend re-ingesting a current source on the topic.
- **Superseded** → recommend `status: deprecated` plus a link to the replacement page.

Don't change dates yourself just to silence the warning — a fresh `stale_after` is a claim
that someone checked, and only the check itself justifies it.
