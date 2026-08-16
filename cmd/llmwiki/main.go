// llmwiki is the deterministic companion CLI to the llmwiki Claude Code plugin.
//
//	llmwiki init [dir]                        scaffold a wiki project
//	llmwiki lint [--format json] [--strict] [bundle-dir]
//	llmwiki version
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	llmwikiokf "github.com/KatsuyaAkasaka/llmwiki-okf"
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
		var err error
		if dir, err = findBundle(); err != nil {
			fmt.Fprintln(os.Stderr, "lint:", err)
			return 2
		}
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

// findBundle walks up from the working directory to the nearest llmwiki.yaml
// and returns its bundle_dir, so `llmwiki lint` works from anywhere in a
// wiki project — the same discovery rule the skills use.
func findBundle() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		cfgPath := filepath.Join(dir, "llmwiki.yaml")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg struct {
				BundleDir string `yaml:"bundle_dir"`
			}
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return "", fmt.Errorf("%s: %w", cfgPath, err)
			}
			if cfg.BundleDir == "" {
				cfg.BundleDir = "wiki"
			}
			return filepath.Join(dir, cfg.BundleDir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no llmwiki.yaml found upward from the working directory; pass a bundle dir explicitly")
		}
		dir = parent
	}
}
