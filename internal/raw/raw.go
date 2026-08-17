// Package raw implements directory-based source intake: files dropped into
// the configured raw directory are tracked in a manifest by content hash, so
// repeated ingest runs process only the delta — new and changed files — never
// the whole library again. Distillation itself is the ingest skill's job; this
// package only answers "what is waiting?" and "what has been done?"
// deterministically.
package raw

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

// ManifestName is the tracking file, kept at the wiki project root (next to
// llmwiki.yaml) and outside the bundle so it is never deployed. The raw
// directory itself lives elsewhere (default ~/wiki_raw): the manifest belongs
// to the wiki that consumed the sources, so several wikis can ingest from the
// same drop directory independently.
const ManifestName = ".llmwiki-manifest.json"

// Statuses reported by Scan.
const (
	StatusNew      = "new"      // on disk, not in manifest
	StatusChanged  = "changed"  // on disk, hash differs from manifest
	StatusIngested = "ingested" // on disk, hash matches manifest
	StatusMissing  = "missing"  // in manifest, no longer on disk
)

// ManifestEntry records one ingested source.
type ManifestEntry struct {
	SHA256     string   `json:"sha256"`
	IngestedAt string   `json:"ingested_at"`
	Pages      []string `json:"pages,omitempty"` // bundle-relative pages it touched
}

// Manifest maps raw-relative source paths to their ingest record.
type Manifest struct {
	Version int                      `json:"version"`
	Sources map[string]ManifestEntry `json:"sources"`
}

// normKey returns the canonical manifest key for a path: slash-separated and
// Unicode NFC. macOS reports NFD filenames from directory walks while
// user-typed paths are NFC; without one canonical form the same file shows up
// as both "missing" (the marked NFC key) and "new" (the on-disk NFD name).
func normKey(p string) string {
	return norm.NFC.String(filepath.ToSlash(p))
}

// Load reads the manifest at root, returning an empty one if none exists yet.
// Keys are re-normalized on load so manifests written before normalization
// (or edited by hand) converge to canonical form.
func Load(root string) (*Manifest, error) {
	m := &Manifest{Version: 1, Sources: map[string]ManifestEntry{}}
	data, err := os.ReadFile(filepath.Join(root, ManifestName))
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	normalized := map[string]ManifestEntry{}
	for k, v := range m.Sources {
		normalized[normKey(k)] = v
	}
	m.Sources = normalized
	return m, nil
}

// Save writes the manifest back to root.
func (m *Manifest) Save(root string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ManifestName), append(data, '\n'), 0o644)
}

// Entry is one Scan result row.
type Entry struct {
	Path   string `json:"path"`   // raw-relative, slash-separated
	Status string `json:"status"` // new | changed | ingested | missing
	SHA256 string `json:"sha256,omitempty"`
}

// Scan walks rawDir (an absolute or resolved path) recursively and
// classifies every file against the manifest. Dotfiles (.gitkeep etc.) are
// skipped. Manifest entries whose file has disappeared are reported as
// missing. A non-existent raw directory simply means nothing is waiting.
func Scan(rawDir string, m *Manifest) ([]Entry, error) {
	var entries []Entry
	seen := map[string]bool{}

	err := filepath.WalkDir(rawDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == rawDir {
				return filepath.SkipAll // no raw directory yet — nothing waiting
			}
			return err
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rawDir, p)
		if err != nil {
			return err
		}
		key := normKey(rel)
		sum, err := hashFile(p)
		if err != nil {
			return err
		}
		seen[key] = true
		status := StatusNew
		if prev, ok := m.Sources[key]; ok {
			if prev.SHA256 == sum {
				status = StatusIngested
			} else {
				status = StatusChanged
			}
		}
		entries = append(entries, Entry{Path: key, Status: status, SHA256: sum})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("raw scan: %w", err)
	}

	for rel := range m.Sources {
		if !seen[rel] {
			entries = append(entries, Entry{Path: rel, Status: StatusMissing})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// Mark records the raw-relative file as ingested at its current content
// hash. Call it only after the pages are actually written — the mark is the
// claim that this exact content has been distilled. The stored key is
// canonical (NFC); for reading the file, the given spelling plus its NFC and
// NFD forms are tried, so a hand-typed (NFC) path works even where the
// filesystem stores the name decomposed and compares byte-wise.
func (m *Manifest) Mark(rawDir, file string, pages []string, now time.Time) error {
	var sum string
	var err error
	for _, cand := range []string{file, norm.NFC.String(file), norm.NFD.String(file)} {
		sum, err = hashFile(filepath.Join(rawDir, filepath.FromSlash(cand)))
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("mark: %w", err)
	}
	m.Sources[normKey(file)] = ManifestEntry{
		SHA256:     sum,
		IngestedAt: now.UTC().Format(time.RFC3339),
		Pages:      pages,
	}
	return nil
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
