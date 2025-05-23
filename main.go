package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const htmlTmpl = `<!DOCTYPE html>
<html><head>
  <meta charset="utf-8"/>
  <meta name="go-import" content="{{ .Domain }}/{{ .Suffix }} git https://{{ .Repo }}">
  <meta name="go-source" content="{{ .Domain }}/{{ .Suffix }} https://{{ .Repo }} https://{{ .Repo }}/tree/master{/dir} https://{{ .Repo }}/blob/master{/dir}/{file}#L{line}">
  <meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{ .Domain }}/{{ .Suffix }}">
</head><body>
  Redirecting…
</body></html>`

// tplData holds data passed to the HTML template.
type tplData struct {
	Domain string
	Repo   string
	Suffix string
}

var rootCmd = &cobra.Command{
	Use:   "vanityimport",
	Short: "Utilities for Go vanity imports",
	Long:  "A small CLI that can generate vanity import HTML files and rewrite Go import paths.",
}

var htmlCmd = &cobra.Command{
	Use:   "html",
	Short: "Generate index.html for a vanity import path",
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		repo, _ := cmd.Flags().GetString("repo")
		outDir, _ := cmd.Flags().GetString("out")

		if domain == "" || repo == "" {
			_ = cmd.Help()
			os.Exit(1)
		}

		if err := generateHTML(domain, repo, outDir); err != nil {
			log.Fatalf("generate html: %v", err)
		}
	},
}

var rewriteCmd = &cobra.Command{
	Use:   "rewrite",
	Short: "Rewrite Go import paths in source files",
	Run: func(cmd *cobra.Command, args []string) {
		oldPath, _ := cmd.Flags().GetString("old")
		newPath, _ := cmd.Flags().GetString("new")
		dir, _ := cmd.Flags().GetString("dir")

		if oldPath == "" || newPath == "" {
			_ = cmd.Help()
			os.Exit(1)
		}

		if err := rewriteImports(dir, oldPath, newPath); err != nil {
			log.Fatalf("rewrite imports: %v", err)
		}
	},
}

func init() {
	// Flags for the html sub‑command
	htmlCmd.Flags().StringP("domain", "d", "", "custom domain (e.g. go.example.com)")
	htmlCmd.Flags().StringP("repo", "r", "", "VCS repository path (e.g. github.com/user/project)")
	htmlCmd.Flags().StringP("out", "o", ".", "output directory for index.html")

	// Flags for the rewrite sub‑command
	rewriteCmd.Flags().StringP("old", "o", "", "old import prefix to replace (e.g. github.com/user/project)")
	rewriteCmd.Flags().StringP("new", "n", "", "new import prefix (e.g. go.example.com/project)")
	rewriteCmd.Flags().StringP("dir", "d", ".", "directory to scan recursively for .go files")

	rootCmd.AddCommand(htmlCmd)
	rootCmd.AddCommand(rewriteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// generateHTML creates an index.html with vanity import meta‑tags in outDir.
func generateHTML(domain, repo, outDir string) error {
	parts := strings.Split(strings.TrimSuffix(repo, "/"), "/")
	if len(parts) < 3 {
		return fmt.Errorf("repo should be in the form github.com/user/project")
	}
	suffix := parts[len(parts)-1]

	data := tplData{Domain: domain, Repo: repo, Suffix: suffix}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(outDir, "index.html")
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck

	if err := template.Must(template.New("vanity").Parse(htmlTmpl)).Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Generated %s\n", filePath)
	return nil
}

// rewriteImports walks root and rewrites import paths that start with oldPath to newPath.
func rewriteImports(root, oldPath, newPath string) error {
	fset := token.NewFileSet()
	absRoot, _ := filepath.Abs(root)

	return filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Handle directories first
		if d.IsDir() {
			// Skip hidden or vendor directories, **except** the root itself even if root == "."
			if path != absRoot && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-Go files or hidden files
		if strings.HasPrefix(d.Name(), ".") || filepath.Ext(path) != ".go" {
			return nil
		}

		// Read & parse the Go file
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return err
		}

		modified := false
		for _, imp := range file.Imports {
			impPath, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(impPath, oldPath) {
				imp.Path.Value = strconv.Quote(strings.Replace(impPath, oldPath, newPath, 1))
				modified = true
			}
		}

		if !modified {
			return nil
		}

		var buf bytes.Buffer
		if err := format.Node(&buf, fset, file); err != nil {
			return err
		}

		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			return err
		}

		fmt.Printf("Rewrote imports in %s\n", path)
		return nil
	})
}
