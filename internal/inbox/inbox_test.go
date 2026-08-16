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

func TestInboxLifecycle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("inbox/paper.txt", "v1")
	write("inbox/.gitkeep", "")

	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Fresh file is new; dotfiles are invisible.
	entries, err := Scan(root, "inbox", m)
	if err != nil {
		t.Fatal(err)
	}
	got := statuses(entries)
	if got["inbox/paper.txt"] != StatusNew {
		t.Fatalf("want new, got %v", got)
	}
	if len(entries) != 1 {
		t.Fatalf("dotfiles must be skipped, got %v", entries)
	}

	// 2. Mark → ingested, and the mark survives a save/load round-trip.
	if err := m.Mark(root, "inbox/paper.txt", []string{"sources/paper.md"}, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(root); err != nil {
		t.Fatal(err)
	}
	m2, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(root, "inbox", m2)
	if got := statuses(entries); got["inbox/paper.txt"] != StatusIngested {
		t.Fatalf("want ingested after mark+reload, got %v", got)
	}

	// 3. Edit → changed.
	write("inbox/paper.txt", "v2")
	entries, _ = Scan(root, "inbox", m2)
	if got := statuses(entries); got["inbox/paper.txt"] != StatusChanged {
		t.Fatalf("want changed, got %v", got)
	}

	// 4. Delete → missing.
	if err := os.Remove(filepath.Join(root, "inbox/paper.txt")); err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(root, "inbox", m2)
	if got := statuses(entries); got["inbox/paper.txt"] != StatusMissing {
		t.Fatalf("want missing, got %v", got)
	}
}

func TestScanWithoutInboxDir(t *testing.T) {
	root := t.TempDir()
	m, _ := Load(root)
	entries, err := Scan(root, "inbox", m)
	if err != nil {
		t.Fatalf("missing inbox dir must not error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("want empty, got %v", entries)
	}
}
