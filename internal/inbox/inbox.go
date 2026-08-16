// Package inbox implements directory-based source intake: files dropped into
// the configured inbox directory are tracked in a manifest by content hash, so
// repeated ingest runs process only the delta — new and changed files — never
// the whole library again. Distillation itself is the ingest skill's job; this
// package only answers "what is waiting?" and "what has been done?"
// deterministically.
package inbox

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
)

// ManifestName is the tracking file, kept at the wiki project root (next to
// llmwiki.yaml) and outside the bundle so it is never deployed. The inbox
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

// Manifest maps inbox-relative source paths to their ingest record.
type Manifest struct {
	Version int                      `json:"version"`
	Sources map[string]ManifestEntry `json:"sources"`
}

// Load reads the manifest at root, returning an empty one if none exists yet.
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
	if m.Sources == nil {
		m.Sources = map[string]ManifestEntry{}
	}
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
	Path   string `json:"path"`   // inbox-relative, slash-separated
	Status string `json:"status"` // new | changed | ingested | missing
	SHA256 string `json:"sha256,omitempty"`
}

// Scan walks inboxDir (an absolute or resolved path) recursively and
// classifies every file against the manifest. Dotfiles (.gitkeep etc.) are
// skipped. Manifest entries whose file has disappeared are reported as
// missing. A non-existent inbox directory simply means nothing is waiting.
func Scan(inboxDir string, m *Manifest) ([]Entry, error) {
	var entries []Entry
	seen := map[string]bool{}

	err := filepath.WalkDir(inboxDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == inboxDir {
				return filepath.SkipAll // no inbox directory yet — nothing waiting
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
		rel, err := filepath.Rel(inboxDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		sum, err := hashFile(p)
		if err != nil {
			return err
		}
		seen[rel] = true
		status := StatusNew
		if prev, ok := m.Sources[rel]; ok {
			if prev.SHA256 == sum {
				status = StatusIngested
			} else {
				status = StatusChanged
			}
		}
		entries = append(entries, Entry{Path: rel, Status: status, SHA256: sum})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inbox scan: %w", err)
	}

	for rel := range m.Sources {
		if !seen[rel] {
			entries = append(entries, Entry{Path: rel, Status: StatusMissing})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// Mark records the inbox-relative file as ingested at its current content
// hash. Call it only after the pages are actually written — the mark is the
// claim that this exact content has been distilled.
func (m *Manifest) Mark(inboxDir, file string, pages []string, now time.Time) error {
	sum, err := hashFile(filepath.Join(inboxDir, filepath.FromSlash(file)))
	if err != nil {
		return fmt.Errorf("mark: %w", err)
	}
	m.Sources[filepath.ToSlash(file)] = ManifestEntry{
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
