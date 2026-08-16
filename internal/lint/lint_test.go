package lint

import (
	"os"
	"path/filepath"
	"testing"
)

const today = "2026-08-16"

// write creates a bundle from rel→content pairs and returns its root.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func rules(issues []Issue) map[string]int {
	m := map[string]int{}
	for _, i := range issues {
		m[i.Rule]++
	}
	return m
}

func TestConformantBundle(t *testing.T) {
	dir := write(t, map[string]string{
		"index.md": "---\nokf_version: \"0.2\"\n---\n\n# Index\n\n## Concepts\n\n" +
			"* [Foo](/concepts/foo.md) - a concept.\n* [Bar](/concepts/bar.md) - another.\n",
		"log.md": "# Log\n\n## 2026-08-16\n\n* **Creation**: added [Foo](/concepts/foo.md).\n",
		"concepts/foo.md": "---\ntype: Concept\ntitle: Foo\ndescription: About foo.\n" +
			"generated: { by: llmwiki/test, at: 2026-08-16T00:00:00Z }\n---\n\nSee [Bar](/concepts/bar.md).\n",
		"concepts/bar.md": "---\ntype: Concept\ntitle: Bar\ndescription: About bar.\n" +
			"sources:\n  - id: s1\n    resource: https://example.com\n    last_modified: 2026-08-01\n" +
			"verified:\n  by: human:alice\n  at: 2026-08-16T00:00:00Z\n---\n\nSee [Foo](./foo.md).\n",
	})
	res, err := Run(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 0 {
		t.Errorf("want no errors, got %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("want no warnings, got %v", res.Warnings)
	}
	if res.Stats.Pages != 4 {
		t.Errorf("want 4 pages, got %d", res.Stats.Pages)
	}
}

func TestConformanceErrors(t *testing.T) {
	dir := write(t, map[string]string{
		"index.md":     "# Index\n\n* [A](/a.md) - a\n* [B](/b.md) - b\n* [C](/c.md) - c\n* [D](/d.md) - d\n",
		"a.md":         "no frontmatter at all\n",
		"b.md":         "---\ntitle: no type\ndescription: x\n---\nbody [a](/a.md) [c](/c.md) [d](/d.md)\n",
		"c.md":         "---\ntype: Concept\ntitle: C\ndescription: x\nstatus: wip\ngenerated: { at: 2026-01-01T00:00:00Z }\n---\nbody\n",
		"d.md":         "---\ntype: Concept\ntitle: D\ndescription: x\nsources:\n  - id: s1\n    title: missing resource\nstale_after: not-a-date\n---\nbody\n",
		"sub/index.md": "---\nokf_version: \"0.2\"\n---\n\n# Sub\n",
		"log.md":       "# Log\n\n## yesterday\n\n* **Update**: things.\n",
	})
	res, err := Run(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	got := rules(res.Errors)
	for rule, want := range map[string]int{
		"frontmatter-parse":  1, // a.md
		"missing-type":       1, // b.md
		"invalid-status":     1, // c.md
		"invalid-generated":  1, // c.md
		"invalid-sources":    1, // d.md
		"invalid-date":       1, // d.md stale_after
		"reserved-structure": 2, // sub/index.md frontmatter + log.md heading
	} {
		if got[rule] != want {
			t.Errorf("rule %s: want %d, got %d (all: %v)", rule, want, got[rule], res.Errors)
		}
	}
}

func TestWarnings(t *testing.T) {
	dir := write(t, map[string]string{
		"index.md": "# Index\n\n* [Linked](/linked.md) - fine.\n",
		"linked.md": "---\ntype: Concept\ntitle: Linked\ndescription: x\nstale_after: 2026-01-01\n---\n" +
			"[gone](/nowhere.md)\n\n```\n[not a real link](/code-block.md)\n```\n",
		"orphan.md": "---\ntype: Concept\n---\nnobody links here\n",
	})
	res, err := Run(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	got := rules(res.Warnings)
	for rule, want := range map[string]int{
		"broken-link":         1, // /nowhere.md only — the code-block link must be ignored
		"stale":               1, // linked.md
		"orphan":              1, // orphan.md
		"unindexed":           1, // orphan.md
		"missing-title":       1, // orphan.md
		"missing-description": 1, // orphan.md
	} {
		if got[rule] != want {
			t.Errorf("rule %s: want %d, got %d (all: %v)", rule, want, got[rule], res.Warnings)
		}
	}
	if len(res.Errors) != 0 {
		t.Errorf("want no errors, got %v", res.Errors)
	}
}

func TestVerifiedBareMappingAccepted(t *testing.T) {
	dir := write(t, map[string]string{
		"index.md": "# I\n\n* [A](/a.md) - a.\n",
		"a.md": "---\ntype: Concept\ntitle: A\ndescription: x\n" +
			"verified: { by: process:nightly, at: 2026-08-16T00:00:00Z }\n---\n[i](/index.md)\n",
	})
	res, err := Run(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	if n := rules(res.Errors)["invalid-verified"]; n != 0 {
		t.Errorf("bare verified mapping must be accepted, got %v", res.Errors)
	}
}
