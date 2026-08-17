package raw

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"
)

func statuses(entries []Entry) map[string]string {
	m := map[string]string{}
	for _, e := range entries {
		m[e.Path] = e.Status
	}
	return m
}

// The raw lives outside the wiki project (default ~/wiki_raw), while the
// manifest lives at the project root — the test mirrors that split.
func TestRawLifecycle(t *testing.T) {
	projectRoot := t.TempDir()
	rawDir := filepath.Join(t.TempDir(), "wiki_raw")
	if err := os.MkdirAll(filepath.Join(rawDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(rawDir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
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

	// 1. Fresh files are new, keyed relative to the raw; dotfiles invisible.
	entries, err := Scan(rawDir, m)
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
	if err := m.Mark(rawDir, "paper.txt", []string{"sources/paper.md"}, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(projectRoot); err != nil {
		t.Fatal(err)
	}
	m2, err := Load(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(rawDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusIngested || got["sub/notes.md"] != StatusNew {
		t.Fatalf("want ingested+new after mark+reload, got %v", got)
	}

	// 3. Edit → changed.
	write("paper.txt", "v2")
	entries, _ = Scan(rawDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusChanged {
		t.Fatalf("want changed, got %v", got)
	}

	// 4. Delete → missing.
	if err := os.Remove(filepath.Join(rawDir, "paper.txt")); err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(rawDir, m2)
	if got := statuses(entries); got["paper.txt"] != StatusMissing {
		t.Fatalf("want missing, got %v", got)
	}
}

// macOS reports NFD filenames from directory walks while user-typed paths are
// NFC. Keys must converge to one canonical form or the same file shows up as
// both "missing" (marked NFC key) and "new" (on-disk NFD name).
func TestUnicodeNormalizationKeys(t *testing.T) {
	rawDir := t.TempDir()
	projectRoot := t.TempDir()
	nfd := "ガイドライン.md" // カ+濁点(分解形)で始まるファイル名
	nfc := norm.NFC.String(nfd)
	if nfd == nfc {
		t.Fatal("test fixture must differ between NFC and NFD")
	}
	if err := os.WriteFile(filepath.Join(rawDir, nfd), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := Load(projectRoot)
	entries, err := Scan(rawDir, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != nfc {
		t.Fatalf("scan must report the NFC key, got %+v", entries)
	}

	// Mark with the NFC spelling (as a user would type it) — even though the
	// on-disk name may be stored decomposed.
	if err := m.Mark(rawDir, nfc, []string{"sources/x.md"}, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(projectRoot); err != nil {
		t.Fatal(err)
	}
	m2, err := Load(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ = Scan(rawDir, m2)
	if len(entries) != 1 || entries[0].Status != StatusIngested {
		t.Fatalf("want single ingested entry (no missing/new split), got %+v", entries)
	}

	// A pre-normalization manifest with an NFD key must converge on Load.
	m2.Sources = map[string]ManifestEntry{nfd: m2.Sources[nfc]}
	if err := m2.Save(projectRoot); err != nil {
		t.Fatal(err)
	}
	m3, err := Load(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m3.Sources[nfc]; !ok {
		t.Fatalf("Load must normalize legacy keys to NFC, got %v", m3.Sources)
	}
}

func TestScanWithoutRawDir(t *testing.T) {
	m, _ := Load(t.TempDir())
	entries, err := Scan(filepath.Join(t.TempDir(), "does-not-exist"), m)
	if err != nil {
		t.Fatalf("missing raw dir must not error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("want empty, got %v", entries)
	}
}
