package inbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func statuses(entries []Entry) map[string]string {
	m := map[string]string{}
	for _, e := range entries {
		m[e.Path] = e.Status
	}
	return m
}

// The inbox lives outside the wiki project (default ~/wiki_raw), while the
// manifest lives at the project root — the test mirrors that split.
func TestInboxLifecycle(t *testing.T) {
	projectRoot := t.TempDir()
	inboxDir := filepath.Join(t.TempDir(), "wiki_raw")
	if err := os.MkdirAll(filepath.Join(inboxDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(inboxDir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("paper.txt", "v1")
	write("sub/notes.md", "notes")
	write(".gitkeep", "")

	m, err := Load(projectRoot)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Fresh files are new, keyed relative to the inbox; dotfiles invisible.
	entries, err := Scan(inboxDir, m)
	if err != nil {
		t.Fatal(err)
	}
	got := statuses(entries)
	if got["paper.txt"] != StatusNew || got["sub/notes.md"] != StatusNew {
		t.Fatalf("want new+new, got %v", got)
	}
	if len(entries) != 2 {
		t.Fatalf("dotfiles must be skipped, got %v", entries)
	}

	// 2. Mark → ingested, surviving a save/load round-trip at the project root.
	if err := m.Mark(inboxDir, "paper.txt", []string{"sources/paper.md"}, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(projectRoot); err != nil {
		t.Fatal(err)
	}
	m2, err := Load(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(inboxDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusIngested || got["sub/notes.md"] != StatusNew {
		t.Fatalf("want ingested+new after mark+reload, got %v", got)
	}

	// 3. Edit → changed.
	write("paper.txt", "v2")
	entries, _ = Scan(inboxDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusChanged {
		t.Fatalf("want changed, got %v", got)
	}

	// 4. Delete → missing.
	if err := os.Remove(filepath.Join(inboxDir, "paper.txt")); err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(inboxDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusMissing {
		t.Fatalf("want missing, got %v", got)
	}
}

func TestScanWithoutInboxDir(t *testing.T) {
	m, _ := Load(t.TempDir())
	entries, err := Scan(filepath.Join(t.TempDir(), "does-not-exist"), m)
	if err != nil {
		t.Fatalf("missing inbox dir must not error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("want empty, got %v", entries)
	}
}
