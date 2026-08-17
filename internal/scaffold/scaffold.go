// Package scaffold writes the embedded project template to disk.
package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Managed lists template-relative paths the TOOL owns: generated artifacts a
// user is not expected to edit. `update` may overwrite these; everything else
// in a project (config, wiki pages) is user content and is never touched.
var Managed = []string{
	"wiki/index.html", // the viewer SPA
}

// Result reports what Write did.
type Result struct {
	Created []string
	Skipped []string // already existed; never overwritten
}

// Write copies src (rooted at srcRoot) into destDir, creating directories as
// needed. Existing files are left untouched so init is safe to re-run on a
// project that already has content.
func Write(src fs.FS, srcRoot, destDir string) (*Result, error) {
	res := &Result{}
	err := fs.WalkDir(src, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if _, err := os.Stat(dest); err == nil {
			res.Skipped = append(res.Skipped, rel)
			return nil
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
		res.Created = append(res.Created, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scaffold: %w", err)
	}
	return res, nil
}

// UpdateResult reports what Update did.
type UpdateResult struct {
	Updated   []string // rewritten to the current template version
	Unchanged []string // already identical
}

// Update refreshes the Managed files of an existing project from the template,
// so mechanism updates (a new viewer, future tool-owned assets) propagate to
// wikis scaffolded from older versions. The template keeps its bundle at
// "wiki/"; projects may have renamed theirs, so that prefix is remapped to
// bundleDir.
func Update(src fs.FS, srcRoot, projectRoot, bundleDir string) (*UpdateResult, error) {
	res := &UpdateResult{}
	for _, rel := range Managed {
		data, err := fs.ReadFile(src, path.Join(srcRoot, rel))
		if err != nil {
			return nil, fmt.Errorf("update: %w", err)
		}
		destRel := rel
		if after, ok := strings.CutPrefix(rel, "wiki/"); ok {
			destRel = filepath.Join(bundleDir, after)
		}
		dest := filepath.Join(projectRoot, filepath.FromSlash(destRel))
		if old, err := os.ReadFile(dest); err == nil && bytes.Equal(old, data) {
			res.Unchanged = append(res.Unchanged, destRel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, fmt.Errorf("update: %w", err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return nil, fmt.Errorf("update: %w", err)
		}
		res.Updated = append(res.Updated, destRel)
	}
	return res, nil
}
