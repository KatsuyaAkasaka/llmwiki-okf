# llmwiki-okf

An LLM-maintained knowledge wiki, in the open.

- **Pattern**: [Karpathy's LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) —
  the wiki is a *compounding artifact* the LLM builds and maintains, not a RAG index that
  re-derives everything per query.
- **Format**: [Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) —
  plain markdown + YAML frontmatter with provenance (`sources`), trust (`generated`/`verified`),
  and freshness (`status`/`stale_after`). Vendor-neutral, readable by humans and LLMs alike.
- **Interface**: [Claude Code](https://claude.com/claude-code) skills — `ingest`, `query`, `lint` —
  plus a deterministic Go CLI for scaffolding and conformance checking.
- **Viewing**: raw markdown is the primary artifact — LLMs and agents read pages
  directly. Humans get a self-contained single-file viewer (`index.html`) that renders
  the same markdown client-side. No build step, no SSG: the bundle is served as-is by
  any static file server.

## Quickstart

```bash
# 1. Install the CLI
go install github.com/KatsuyaAkasaka/llmwiki-okf/cmd/llmwiki@latest

# 2. Scaffold a wiki project
llmwiki init my-wiki && cd my-wiki
# → llmwiki.yaml (config) + wiki/ (the OKF bundle: index.md, log.md, seed pages, viewer)

# 3. Install the Claude Code plugin (inside claude):
#    /plugin marketplace add KatsuyaAkasaka/llmwiki-okf
#    /plugin install llmwiki@llmwiki-okf
```

Then, in Claude Code, inside your wiki project:

| You say | What happens |
|---|---|
| "Ingest https://example.com/paper into the wiki" | `ingest` distills the source into cross-linked pages with provenance, updates `index.md` + `log.md`, self-checks with `llmwiki lint` |
| "What do we know about X?" | `query` navigates `index.md` → pages and answers **with receipts**: source pages, trust tier, freshness |
| "Lint the wiki" | `lint` runs the deterministic CLI checks, then the semantic ones (contradictions, duplicates, gaps) and offers fixes |

The `query` skill also works **without a local checkout**: point it at a hosted wiki via
`remote_url` in `llmwiki.yaml` or the `LLMWIKI_REMOTE_URL` environment variable, and it
fetches `index.md` and pages over HTTP. Install the plugin, set the URL, ask questions.

## The bundle

```
my-wiki/
├── llmwiki.yaml           # bundle_dir, remote_url, actor, categories
└── wiki/                  # OKF bundle = exactly what you upload
    ├── index.html         # self-contained viewer (hash-routing SPA, no CDN deps)
    ├── index.md           # catalog — the map both humans and LLMs navigate by
    ├── log.md             # append-only history, newest first
    ├── concepts/ entities/ guides/ sources/ synthesis/
    └── references/        # opt-in archives of ephemeral sources
```

Every page is a markdown file with OKF frontmatter. Trust is derived, never asserted:
no `verified` → *unverified*; machine actors → *machine-confirmed*; any `human:` entry →
*human-reviewed*. See [docs/okf-cheatsheet.md](docs/okf-cheatsheet.md) for the format subset
this project implements, and [DESIGN.md](DESIGN.md) for the design decisions (Japanese).

## Linting

```bash
llmwiki lint                  # human-readable; exit 1 on conformance errors
llmwiki lint --format json    # for CI and the lint skill
llmwiki lint --strict         # warnings fail too
```

Errors are OKF §11 conformance violations (frontmatter parseability, non-empty `type`,
reserved-file structure, malformed field shapes). Warnings are quality debt: broken links,
pages past `stale_after`, orphans, pages missing from every `index.md`, missing
title/description. Contradiction/duplicate/gap detection needs judgment and lives in the
lint *skill*, layered on top of the CLI's JSON output.

## The viewer

`wiki/index.html` is scaffolded by `llmwiki init` and needs no generation step afterwards:
it is one self-contained file (inline CSS/JS, no CDN dependencies) that renders the bundle's
markdown client-side. There is nothing to rebuild when pages change — publishing a page *is*
writing its `.md` file.

How it works:

- **Hash routing** — `index.html#/concepts/foo.md` fetches `concepts/foo.md` with a
  relative request and renders it in place; the sidebar is built from `index.md`.
  Deep links to any page are shareable URLs.
- **Frontmatter panel** — each page shows its OKF metadata as badges: `type`, `status`,
  derived trust tier (*unverified / machine-confirmed / human-reviewed*), tags, and a
  red *stale* badge once `stale_after` has passed. The raw frontmatter sits behind a
  collapsible disclosure.
- **Link rewriting** — internal `.md` links become viewer links (`#/...`); external links
  open normally. Markdown rendering covers headings, lists, tables, fenced code,
  blockquotes, and OKF's source footnotes. Dark/light follows the system theme.

To view it, serve the bundle directory over HTTP — the viewer fetches pages, so opening
the file via `file://` won't work:

```bash
cd wiki && python3 -m http.server 8000
# → http://localhost:8000/index.html
```

The same pair of views holds wherever the bundle is served from, since the viewer uses
relative fetch paths only:

- Humans: `<base-url>/index.html` (or `<base-url>/index.html#/path/page.md`)
- LLMs/agents: `<base-url>/index.md`, and every page as raw markdown

Deployment itself is deliberately out of scope — sync the `wiki/` directory to your
static host however you like, ideally gated on `llmwiki lint --strict`.

## License

Apache-2.0
