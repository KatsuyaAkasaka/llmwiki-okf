// Package lint implements the deterministic OKF v0.2 conformance and quality
// checks for an llmwiki bundle. Errors are conformance violations (OKF §11);
// warnings are quality debt (broken links, staleness, orphans, index gaps).
// Semantic checks (contradictions, duplicates) are out of scope here — they
// need judgment and live in the Claude Code lint skill.
package lint

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Issue is one finding, addressed by bundle-relative file path.
type Issue struct {
	File    string `json:"file"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Stats summarizes what was checked.
type Stats struct {
	Pages int `json:"pages"`
	Links int `json:"links"`
}

// Result is the full lint outcome; it marshals to the JSON contract used by
// the lint skill and CI.
type Result struct {
	Bundle   string  `json:"bundle"`
	Errors   []Issue `json:"errors"`
	Warnings []Issue `json:"warnings"`
	Stats    Stats   `json:"stats"`
}

func (r *Result) errorf(file, rule, format string, a ...any) {
	r.Errors = append(r.Errors, Issue{File: file, Rule: rule, Message: fmt.Sprintf(format, a...)})
}

func (r *Result) warnf(file, rule, format string, a ...any) {
	r.Warnings = append(r.Warnings, Issue{File: file, Rule: rule, Message: fmt.Sprintf(format, a...)})
}

var (
	linkRe        = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	fenceRe       = regexp.MustCompile("(?ms)^```.*?^```[ \t]*$")
	inlineCodeRe  = regexp.MustCompile("`[^`\n]*`")
	dateRe        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	logHeadingRe  = regexp.MustCompile(`(?m)^## +(.+)$`)
	validStatuses = map[string]bool{"draft": true, "stable": true, "deprecated": true}
)

type page struct {
	rel      string // bundle-relative path, slash-separated
	reserved bool   // index.md or log.md
	raw      string
	fm       map[string]any // nil if absent/unparseable
	body     string
	links    []string // resolved bundle-relative targets of internal .md links
}

// Run lints the bundle rooted at dir. The returned error covers I/O problems
// only; lint findings are in the Result. `today` in YYYY-MM-DD decides
// staleness (pass "" for time.Now).
func Run(dir, today string) (*Result, error) {
	if today == "" {
		today = time.Now().UTC().Format("2006-01-02")
	}
	res := &Result{Bundle: dir, Errors: []Issue{}, Warnings: []Issue{}}

	var pages []*page
	root := os.DirFS(dir)
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		data, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		base := path.Base(p)
		pages = append(pages, &page{
			rel:      p,
			reserved: base == "index.md" || base == "log.md",
			raw:      string(data),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lint: %w", err)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].rel < pages[j].rel })

	exists := map[string]bool{}
	for _, pg := range pages {
		exists[pg.rel] = true
	}

	for _, pg := range pages {
		parseFrontmatter(pg, res)
		if pg.reserved {
			checkReserved(pg, res)
		} else {
			checkConcept(pg, res, today)
		}
		extractLinks(pg)
	}

	checkLinks(pages, exists, res)
	checkGraph(pages, res)

	res.Stats.Pages = len(pages)
	for _, pg := range pages {
		res.Stats.Links += len(pg.links)
	}
	return res, nil
}

// parseFrontmatter splits raw into fm/body. Missing or unparseable frontmatter
// on a concept doc is a conformance error; reserved files are handled later.
func parseFrontmatter(pg *page, res *Result) {
	pg.body = pg.raw
	rest, ok := strings.CutPrefix(pg.raw, "---\n")
	if !ok {
		if !pg.reserved {
			res.errorf(pg.rel, "frontmatter-parse", "missing YAML frontmatter (--- block)")
		}
		return
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		res.errorf(pg.rel, "frontmatter-parse", "unterminated frontmatter block")
		return
	}
	yamlSrc := rest[:idx]
	after := rest[idx+len("\n---"):]
	if nl := strings.IndexByte(after, '\n'); nl >= 0 {
		pg.body = after[nl+1:]
	} else {
		pg.body = ""
	}
	fm := map[string]any{}
	if err := yaml.Unmarshal([]byte(yamlSrc), &fm); err != nil {
		res.errorf(pg.rel, "frontmatter-parse", "invalid YAML: %v", err)
		return
	}
	pg.fm = fm
}

func checkConcept(pg *page, res *Result, today string) {
	if pg.fm == nil {
		return // frontmatter-parse already reported
	}
	if t, _ := pg.fm["type"].(string); strings.TrimSpace(t) == "" {
		res.errorf(pg.rel, "missing-type", "frontmatter must have a non-empty `type`")
	}
	if s, ok := pg.fm["status"]; ok {
		if str, _ := s.(string); !validStatuses[str] {
			res.errorf(pg.rel, "invalid-status", "status %q is not draft|stable|deprecated", s)
		}
	}
	if g, ok := pg.fm["generated"]; ok {
		m, ok := g.(map[string]any)
		by, _ := "", ""
		if ok {
			by, _ = m["by"].(string)
		}
		if !ok || strings.TrimSpace(by) == "" {
			res.errorf(pg.rel, "invalid-generated", "generated must be a mapping with a non-empty `by`")
		}
	}
	if s, ok := pg.fm["sources"]; ok {
		entries, ok := s.([]any)
		if !ok {
			res.errorf(pg.rel, "invalid-sources", "sources must be a list of entries")
		} else {
			for i, e := range entries {
				m, ok := e.(map[string]any)
				resrc, _ := "", ""
				if ok {
					resrc, _ = m["resource"].(string)
				}
				if !ok || strings.TrimSpace(resrc) == "" {
					res.errorf(pg.rel, "invalid-sources", "sources[%d] must have a non-empty `resource`", i)
				} else if lm, ok := m["last_modified"]; ok && !isDate(lm) {
					res.errorf(pg.rel, "invalid-date", "sources[%d].last_modified %v is not YYYY-MM-DD", i, lm)
				}
			}
		}
	}
	if v, ok := pg.fm["verified"]; ok {
		// Spec: a bare mapping is treated as a one-element list.
		var entries []any
		switch vv := v.(type) {
		case []any:
			entries = vv
		case map[string]any:
			entries = []any{vv}
		default:
			res.errorf(pg.rel, "invalid-verified", "verified must be a mapping or list of mappings")
		}
		for i, e := range entries {
			m, ok := e.(map[string]any)
			by, _ := "", ""
			if ok {
				by, _ = m["by"].(string)
			}
			if !ok || strings.TrimSpace(by) == "" {
				res.errorf(pg.rel, "invalid-verified", "verified[%d] must have a non-empty `by`", i)
			}
		}
	}
	if sa, ok := pg.fm["stale_after"]; ok {
		if !isDate(sa) {
			res.errorf(pg.rel, "invalid-date", "stale_after %v is not YYYY-MM-DD", sa)
		} else if dateString(sa) <= today {
			res.warnf(pg.rel, "stale", "stale_after %v has passed", sa)
		}
	}
	if t, _ := pg.fm["title"].(string); strings.TrimSpace(t) == "" {
		res.warnf(pg.rel, "missing-title", "recommended field `title` is missing")
	}
	if d, _ := pg.fm["description"].(string); strings.TrimSpace(d) == "" {
		res.warnf(pg.rel, "missing-description", "recommended field `description` is missing")
	}
}

func checkReserved(pg *page, res *Result) {
	base := path.Base(pg.rel)
	if base == "index.md" {
		if pg.fm != nil {
			if pg.rel != "index.md" {
				res.errorf(pg.rel, "reserved-structure", "only the bundle-root index.md may have frontmatter")
			} else {
				for k := range pg.fm {
					if k != "okf_version" {
						res.errorf(pg.rel, "reserved-structure", "root index.md frontmatter allows only okf_version, found %q", k)
					}
				}
			}
		}
		return
	}
	// log.md: every ## heading must be an ISO date.
	if pg.fm != nil {
		res.errorf(pg.rel, "reserved-structure", "log.md must not have frontmatter")
	}
	for _, m := range logHeadingRe.FindAllStringSubmatch(stripCode(pg.body), -1) {
		if !dateRe.MatchString(strings.TrimSpace(m[1])) {
			res.errorf(pg.rel, "reserved-structure", "log.md heading %q is not ## YYYY-MM-DD", m[1])
		}
	}
}

// extractLinks collects internal .md link targets, resolved bundle-relative.
// Code blocks are stripped first: links inside examples aren't claims.
func extractLinks(pg *page) {
	for _, m := range linkRe.FindAllStringSubmatch(stripCode(pg.body), -1) {
		target := m[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") ||
			strings.HasPrefix(target, "#") {
			continue
		}
		target, _, _ = strings.Cut(target, "#")
		if !strings.HasSuffix(target, ".md") {
			continue
		}
		var resolved string
		if strings.HasPrefix(target, "/") {
			resolved = path.Clean(strings.TrimPrefix(target, "/"))
		} else {
			resolved = path.Clean(path.Join(path.Dir(pg.rel), target))
		}
		pg.links = append(pg.links, resolved)
	}
}

func checkLinks(pages []*page, exists map[string]bool, res *Result) {
	for _, pg := range pages {
		for _, target := range pg.links {
			if !exists[target] {
				res.warnf(pg.rel, "broken-link", "link target /%s does not exist", target)
			}
		}
	}
}

// checkGraph flags concept docs nobody links to (orphan) and concept docs no
// index.md lists (unindexed). Invisible pages can't be found by queries.
func checkGraph(pages []*page, res *Result) {
	inbound := map[string]bool{}
	indexed := map[string]bool{}
	for _, pg := range pages {
		fromIndex := path.Base(pg.rel) == "index.md"
		for _, target := range pg.links {
			if target == pg.rel {
				continue
			}
			inbound[target] = true
			if fromIndex {
				indexed[target] = true
			}
		}
	}
	for _, pg := range pages {
		if pg.reserved {
			continue
		}
		if !inbound[pg.rel] {
			res.warnf(pg.rel, "orphan", "no other page links here")
		}
		if !indexed[pg.rel] {
			res.warnf(pg.rel, "unindexed", "not listed in any index.md")
		}
	}
}

func stripCode(s string) string {
	return inlineCodeRe.ReplaceAllString(fenceRe.ReplaceAllString(s, ""), "")
}

func isDate(v any) bool {
	return dateRe.MatchString(dateString(v))
}

func dateString(v any) string {
	switch d := v.(type) {
	case string:
		return d
	case time.Time:
		return d.Format("2006-01-02")
	default:
		return fmt.Sprintf("%v", v)
	}
}
