// llmwiki is the deterministic companion CLI to the llmwiki Claude Code plugin.
//
//	llmwiki init [dir]                        scaffold a wiki project
//	llmwiki lint [--format json] [--strict] [bundle-dir]
//	llmwiki inbox [--format json]             list inbox sources vs manifest
//	llmwiki inbox mark <file> [--pages a,b]   record a source as ingested
//	llmwiki version
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	llmwikiokf "github.com/KatsuyaAkasaka/llmwiki-okf"
	"github.com/KatsuyaAkasaka/llmwiki-okf/internal/inbox"
	"github.com/KatsuyaAkasaka/llmwiki-okf/internal/lint"
	"github.com/KatsuyaAkasaka/llmwiki-okf/internal/scaffold"
	"gopkg.in/yaml.v3"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "lint":
		os.Exit(runLint(os.Args[2:]))
	case "inbox":
		os.Exit(runInbox(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println("llmwiki", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  llmwiki init [dir]                              scaffold a wiki project (never overwrites)
  llmwiki lint [--format text|json] [--strict] [bundle-dir]
  llmwiki inbox [--format text|json]              list inbox sources (new/changed/ingested/missing)
  llmwiki inbox mark <file> [--pages a.md,b.md]   record a source as ingested at its current hash
  llmwiki version`)
}

func runInit(args []string) int {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	res, err := scaffold.Write(llmwikiokf.Template, "template", dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init:", err)
		return 2
	}
	for _, f := range res.Created {
		fmt.Println("created", filepath.Join(dir, f))
	}
	for _, f := range res.Skipped {
		fmt.Println("skipped", filepath.Join(dir, f), "(exists)")
	}
	fmt.Println("\nNext: set `actor` (and later `remote_url`) in llmwiki.yaml, then ingest",
		"your first source with the llmwiki plugin's ingest skill.")
	return 0
}

func runLint(args []string) int {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text or json")
	strict := fs.Bool("strict", false, "treat warnings as failures")
	_ = fs.Parse(args)

	dir := fs.Arg(0)
	if dir == "" {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "lint:", err)
			return 2
		}
		dir = filepath.Join(cfg.root, cfg.bundleDir)
	}
	res, err := lint.Run(dir, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "lint:", err)
		return 2
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(os.Stderr, "lint:", err)
			return 2
		}
	default:
		for _, i := range res.Errors {
			fmt.Printf("error   %-20s %s: %s\n", i.Rule, i.File, i.Message)
		}
		for _, i := range res.Warnings {
			fmt.Printf("warning %-20s %s: %s\n", i.Rule, i.File, i.Message)
		}
		fmt.Printf("%d pages, %d links, %d errors, %d warnings\n",
			res.Stats.Pages, res.Stats.Links, len(res.Errors), len(res.Warnings))
	}

	if len(res.Errors) > 0 || (*strict && len(res.Warnings) > 0) {
		return 1
	}
	return 0
}

// runInbox lists inbox sources against the manifest (default), or with
// `mark <file>` records a source as ingested at its current content hash.
// The list is how the ingest skill discovers the delta; the mark is how it
// commits that a file's exact content has been distilled into pages.
func runInbox(args []string) int {
	if len(args) > 0 && args[0] == "mark" {
		return runInboxMark(args[1:])
	}
	fs := flag.NewFlagSet("inbox", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text or json")
	_ = fs.Parse(args)

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "inbox:", err)
		return 2
	}
	m, err := inbox.Load(cfg.root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inbox:", err)
		return 2
	}
	entries, err := inbox.Scan(cfg.root, cfg.inboxDir, m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inbox:", err)
		return 2
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		out := struct {
			Inbox   string        `json:"inbox"`
			Entries []inbox.Entry `json:"entries"`
		}{cfg.inboxDir, entries}
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, "inbox:", err)
			return 2
		}
		return 0
	}
	if len(entries) == 0 {
		fmt.Printf("inbox %s/ is empty\n", cfg.inboxDir)
		return 0
	}
	pending := 0
	for _, e := range entries {
		fmt.Printf("%-9s %s\n", e.Status, e.Path)
		if e.Status == inbox.StatusNew || e.Status == inbox.StatusChanged {
			pending++
		}
	}
	fmt.Printf("%d file(s), %d pending ingestion\n", len(entries), pending)
	return 0
}

func runInboxMark(args []string) int {
	fs := flag.NewFlagSet("inbox mark", flag.ExitOnError)
	pages := fs.String("pages", "", "comma-separated bundle-relative pages this source touched")
	_ = fs.Parse(args)
	file := fs.Arg(0)
	if file == "" {
		fmt.Fprintln(os.Stderr, "inbox mark: a source file path is required")
		return 2
	}
	// The flag package stops at the first positional argument, so parse again
	// past it — `inbox mark <file> --pages ...` must work, not only the
	// flags-first order.
	_ = fs.Parse(fs.Args()[1:])
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "inbox mark:", err)
		return 2
	}
	m, err := inbox.Load(cfg.root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inbox mark:", err)
		return 2
	}
	// Accept both project-root-relative and cwd-relative paths.
	rel := file
	if abs, err := filepath.Abs(file); err == nil {
		if r, err := filepath.Rel(cfg.root, abs); err == nil && !strings.HasPrefix(r, "..") {
			rel = filepath.ToSlash(r)
		}
	}
	var pageList []string
	for _, p := range strings.Split(*pages, ",") {
		if p = strings.TrimSpace(p); p != "" {
			pageList = append(pageList, p)
		}
	}
	if err := m.Mark(cfg.root, rel, pageList, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "inbox mark:", err)
		return 2
	}
	if err := m.Save(cfg.root); err != nil {
		fmt.Fprintln(os.Stderr, "inbox mark:", err)
		return 2
	}
	fmt.Println("marked", rel)
	return 0
}

// config is the llmwiki.yaml contents plus the project root that held it —
// the same discovery rule the skills use.
type config struct {
	root      string
	bundleDir string
	inboxDir  string
}

// loadConfig walks up from the working directory to the nearest llmwiki.yaml,
// so every subcommand works from anywhere inside a wiki project.
func loadConfig() (*config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		cfgPath := filepath.Join(dir, "llmwiki.yaml")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var raw struct {
				BundleDir string `yaml:"bundle_dir"`
				InboxDir  string `yaml:"inbox_dir"`
			}
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return nil, fmt.Errorf("%s: %w", cfgPath, err)
			}
			if raw.BundleDir == "" {
				raw.BundleDir = "wiki"
			}
			if raw.InboxDir == "" {
				raw.InboxDir = "inbox"
			}
			return &config{root: dir, bundleDir: raw.BundleDir, inboxDir: raw.InboxDir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no llmwiki.yaml found upward from the working directory")
		}
		dir = parent
	}
}
