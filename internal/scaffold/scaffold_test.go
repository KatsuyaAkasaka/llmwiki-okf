package scaffold

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestUpdateRefreshesManagedOnly(t *testing.T) {
	src := fstest.MapFS{
		"template/wiki/index.html": {Data: []byte("<html>v2</html>")},
		"template/llmwiki.yaml":    {Data: []byte("bundle_dir: wiki\n")},
	}
	root := t.TempDir()
	bundle := filepath.Join(root, "kb") // renamed bundle dir must be remapped
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	viewer := filepath.Join(bundle, "index.html")
	if err := os.WriteFile(viewer, []byte("<html>v1</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	userPage := filepath.Join(bundle, "index.md")
	if err := os.WriteFile(userPage, []byte("# mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Update(src, "template", root, "kb")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 1 || res.Updated[0] != filepath.Join("kb", "index.html") {
		t.Fatalf("want viewer updated, got %+v", res)
	}
	got, _ := os.ReadFile(viewer)
	if string(got) != "<html>v2</html>" {
		t.Fatalf("viewer not refreshed: %s", got)
	}
	page, _ := os.ReadFile(userPage)
	if string(page) != "# mine" {
		t.Fatalf("user content must be untouched: %s", page)
	}

	// Second run: identical → unchanged.
	res, err = Update(src, "template", root, "kb")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 0 || len(res.Unchanged) != 1 {
		t.Fatalf("want idempotent update, got %+v", res)
	}
}
