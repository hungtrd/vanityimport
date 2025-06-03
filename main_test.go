package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to write a file with directories created.
func write(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestGenerateHTML(t *testing.T) {
	td := t.TempDir()
	domain := "go.example.com"
	repo := "github.com/user/project"

	if err := generateHTMLForRepo(domain, repo, td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := filepath.Join(td, "index.html")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "meta name=\"go-import\"") {
		t.Error("missing go-import meta tag")
	}
	if !strings.Contains(html, domain) {
		t.Error("domain not present in output")
	}
	if !strings.Contains(html, repo) {
		t.Error("repo URL not present in output")
	}
}

func TestGenerateHTML_InvalidRepo(t *testing.T) {
	err := generateHTMLForRepo("go.example.com", "invalid", t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid repo, got nil")
	}
}

func TestRewriteImports(t *testing.T) {
	root := t.TempDir()

	old := "github.com/hungtrd/project"
	new := "go.hung.rocks/project"

	// file to be modified
	write(t, root, "main.go", `package main
import "github.com/hungtrd/project/foo"
func main() {}
`)

	// vendor file should remain untouched
	vendorPath := write(t, root, "vendor/vendor.go", `package vendor
import "github.com/hungtrd/project/bar"
`)

	// hidden directory file
	hiddenDirPath := write(t, root, ".idea/hidden.go", `package hidden
import "github.com/hungtrd/project/baz"
`)

	// dot file
	dotFilePath := write(t, root, ".ignore.go", `package ignore
import "github.com/hungtrd/project/qux"
`)

	if err := rewriteImports(root, old, new); err != nil {
		t.Fatalf("rewrite error: %v", err)
	}

	// main.go should be rewritten
	data, _ := os.ReadFile(filepath.Join(root, "main.go"))
	if !strings.Contains(string(data), new+"/foo") {
		t.Errorf("main.go not rewritten correctly: %s", data)
	}

	// vendor file must stay unchanged
	vData, _ := os.ReadFile(vendorPath)
	if strings.Contains(string(vData), new) {
		t.Error("vendor file should not be rewritten")
	}

	// hidden dir file stays unchanged
	hData, _ := os.ReadFile(hiddenDirPath)
	if strings.Contains(string(hData), new) {
		t.Error("hidden dir file should not be rewritten")
	}

	// dot file stays unchanged
	dData, _ := os.ReadFile(dotFilePath)
	if strings.Contains(string(dData), new) {
		t.Error("dot file should not be rewritten")
	}
}
