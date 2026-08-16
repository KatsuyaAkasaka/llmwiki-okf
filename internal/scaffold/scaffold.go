// Package scaffold writes the embedded project template to disk.
package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

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
