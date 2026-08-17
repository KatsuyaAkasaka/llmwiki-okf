// llmwiki is the deterministic companion CLI to the llmwiki Claude Code plugin.
//
//	llmwiki init [dir]                        scaffold a wiki project
//	llmwiki lint [--format json] [--strict] [bundle-dir]
//	llmwiki raw [--format json]             list raw sources vs manifest
//	llmwiki raw mark <file> [--pages a,b]   record a source as ingested
//	llmwiki version
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	llmwikiokf "github.com/KatsuyaAkasaka/llmwiki-okf"
	"github.com/KatsuyaAkasaka/llmwiki-okf/internal/lint"
	"github.com/KatsuyaAkasaka/llmwiki-okf/internal/raw"
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
	case "raw":
		os.Exit(runRaw(os.Args[2:]))
	case "update":
		os.Exit(runUpdate(os.Args[2:]))
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
  llmwiki raw [--format text|json]              list raw sources (new/changed/ingested/missing)
  llmwiki raw mark <file> [--pages a.md,b.md]   record a source as ingested at its current hash
  llmwiki update [dir]                            refresh tool-owned files (viewer) from the current template
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
		"your first source with the llmwiki plugin's ingest skill.",
		"\nDrop files into ~/wiki_raw (or $LLMWIKI_RAW_DIR) and run `llmwiki raw` to see what's waiting.")
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

// runUpdate propagates mechanism updates into an existing wiki project:
// tool-owned files (the viewer) are refreshed from the embedded template, and
// config keys the template has gained since the project was scaffolded are
// reported — but llmwiki.yaml and wiki pages are user content, never rewritten.
// The target is the path argument, or the enclosing project when run inside one.
func runUpdate(args []string) int {
	var cfg *config
	var err error
	if len(args) > 0 {
		cfg, err = parseConfigAt(args[0])
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "update: %s is not a llmwiki project (no llmwiki.yaml there) — pass the directory that contains llmwiki.yaml, or run `llmwiki init %s` to create one\n", args[0], args[0])
			return 2
		}
	} else {
		cfg, err = loadConfig()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		return 2
	}
	res, err := scaffold.Update(llmwikiokf.Template, "template", cfg.root, cfg.bundleDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		return 2
	}
	fmt.Println("project:", cfg.root)
	for _, f := range res.Updated {
		fmt.Println("updated  ", f)
	}
	for _, f := range res.Unchanged {
		fmt.Println("unchanged", f)
	}
	for _, k := range missingConfigKeys(cfg.root) {
		fmt.Printf("note: llmwiki.yaml has no `%s` key — see the template llmwiki.yaml for what it does (defaults still apply)\n", k)
	}
	return 0
}

// missingConfigKeys diffs the template llmwiki.yaml's top-level keys against
// the project's, so update can surface knobs added after the project was
// scaffolded without ever editing the user's config.
func missingConfigKeys(root string) []string {
	tmpl, err := llmwikiokf.Template.ReadFile("template/llmwiki.yaml")
	if err != nil {
		return nil
	}
	proj, err := os.ReadFile(filepath.Join(root, "llmwiki.yaml"))
	if err != nil {
		return nil
	}
	keys := func(data []byte) map[string]bool {
		m := map[string]any{}
		_ = yaml.Unmarshal(data, &m)
		set := map[string]bool{}
		for k := range m {
			set[k] = true
		}
		return set
	}
	have := keys(proj)
	var missing []string
	for k := range keys(tmpl) {
		if !have[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

// runRaw lists raw sources against the manifest (default), or with
// `mark <file>` records a source as ingested at its current content hash.
// The list is how the ingest skill discovers the delta; the mark is how it
// commits that a file's exact content has been distilled into pages.
func runRaw(args []string) int {
	if len(args) > 0 && args[0] == "mark" {
		return runRawMark(args[1:])
	}
	fs := flag.NewFlagSet("raw", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text or json")
	_ = fs.Parse(args)

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw:", err)
		return 2
	}
	m, err := raw.Load(cfg.root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw:", err)
		return 2
	}
	entries, err := raw.Scan(cfg.rawDir, m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw:", err)
		return 2
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		out := struct {
			Raw     string      `json:"raw"`
			Entries []raw.Entry `json:"entries"`
		}{cfg.rawDir, entries}
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, "raw:", err)
			return 2
		}
		return 0
	}
	fmt.Printf("raw: %s\n", cfg.rawDir)
	if len(entries) == 0 {
		fmt.Println("empty — nothing waiting")
		return 0
	}
	pending := 0
	for _, e := range entries {
		fmt.Printf("%-9s %s\n", e.Status, e.Path)
		if e.Status == raw.StatusNew || e.Status == raw.StatusChanged {
			pending++
		}
	}
	fmt.Printf("%d file(s), %d pending ingestion\n", len(entries), pending)
	return 0
}

func runRawMark(args []string) int {
	fs := flag.NewFlagSet("raw mark", flag.ExitOnError)
	pages := fs.String("pages", "", "comma-separated bundle-relative pages this source touched")
	_ = fs.Parse(args)
	file := fs.Arg(0)
	if file == "" {
		fmt.Fprintln(os.Stderr, "raw mark: a source file path is required")
		return 2
	}
	// The flag package stops at the first positional argument, so parse again
	// past it — `raw mark <file> --pages ...` must work, not only the
	// flags-first order.
	_ = fs.Parse(fs.Args()[1:])
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw mark:", err)
		return 2
	}
	m, err := raw.Load(cfg.root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw mark:", err)
		return 2
	}
	// Accept absolute, cwd-relative, and raw-relative paths; the manifest
	// key is always raw-relative so it survives the raw dir moving.
	key, err := rawKey(cfg.rawDir, file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw mark:", err)
		return 2
	}
	var pageList []string
	for _, p := range strings.Split(*pages, ",") {
		if p = strings.TrimSpace(p); p != "" {
			pageList = append(pageList, p)
		}
	}
	if err := m.Mark(cfg.rawDir, key, pageList, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "raw mark:", err)
		return 2
	}
	if err := m.Save(cfg.root); err != nil {
		fmt.Fprintln(os.Stderr, "raw mark:", err)
		return 2
	}
	fmt.Println("marked", key)
	return 0
}

// rawKey normalizes a user-supplied path (absolute, cwd-relative, or
// already raw-relative) to the raw-relative manifest key.
func rawKey(rawDir, file string) (string, error) {
	abs := ""
	if filepath.IsAbs(file) {
		abs = filepath.Clean(file)
	} else if _, err := os.Stat(file); err == nil {
		if a, err := filepath.Abs(file); err == nil {
			abs = a
		}
	}
	if abs != "" {
		r, err := filepath.Rel(rawDir, abs)
		if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%s is not inside the raw directory %s", file, rawDir)
		}
		return filepath.ToSlash(r), nil
	}
	return filepath.ToSlash(filepath.Clean(file)), nil
}

// config is the llmwiki.yaml contents plus the project root that held it —
// the same discovery rule the skills use. rawDir is fully resolved.
type config struct {
	root      string
	bundleDir string
	rawDir    string
}

// loadConfig walks up from the working directory to the nearest llmwiki.yaml,
// so every subcommand works from anywhere inside a wiki project. Outside one,
// pass the target explicitly (`llmwiki update <dir>`, `llmwiki lint <bundle>`).
//
// The raw directory resolves as: llmwiki.yaml `raw_dir` (if set) →
// $LLMWIKI_RAW_DIR → ~/wiki_raw. It deliberately defaults to a location
// OUTSIDE the wiki project: raw sources don't belong in a possibly
// git-managed, possibly deployed wiki repo. Relative values are resolved
// against the project root; "~" is expanded.
func loadConfig() (*config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if cfg, err := parseConfigAt(dir); err == nil {
			return cfg, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, fmt.Errorf("no llmwiki.yaml found upward from the working directory; pass the wiki project path explicitly")
}

// parseConfigAt reads dir/llmwiki.yaml. A missing file is reported with
// os.IsNotExist so callers can keep searching.
func parseConfigAt(dir string) (*config, error) {
	cfgPath := filepath.Join(dir, "llmwiki.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	var raw struct {
		BundleDir string `yaml:"bundle_dir"`
		RawDir    string `yaml:"raw_dir"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", cfgPath, err)
	}
	if raw.BundleDir == "" {
		raw.BundleDir = "wiki"
	}
	rawDir, err := resolveRawDir(dir, raw.RawDir)
	if err != nil {
		return nil, err
	}
	return &config{root: dir, bundleDir: raw.BundleDir, rawDir: rawDir}, nil
}

func resolveRawDir(projectRoot, fromYAML string) (string, error) {
	v := fromYAML
	if v == "" {
		v = os.Getenv("LLMWIKI_RAW_DIR")
	}
	if v == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving default raw ~/wiki_raw: %w", err)
		}
		return filepath.Join(home, "wiki_raw"), nil
	}
	v = expandHome(v)
	if filepath.IsAbs(v) {
		return filepath.Clean(v), nil
	}
	return filepath.Join(projectRoot, v), nil
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}
