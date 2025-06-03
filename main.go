package main

import (
	"bytes"
	"encoding/json"
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

// ------------------------------------------------------------
// Template used for each generated vanity page
// ------------------------------------------------------------
const htmlTmpl = `<!DOCTYPE html>
<html><head>
  <meta charset="utf-8"/>
  <meta name="go-import" content="{{ .Domain }}/{{ .Suffix }} git https://{{ .Repo }}">
  <meta name="go-source" content="{{ .Domain }}/{{ .Suffix }} https://{{ .Repo }} https://{{ .Repo }}/tree/master{/dir} https://{{ .Repo }}/blob/master{/dir}/{file}#L{line}">
  <meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{ .Domain }}/{{ .Suffix }}">
</head><body>
  Redirecting…
</body></html>`

//------------------------------------------------------------
// Types
//------------------------------------------------------------

type tplData struct {
	Domain string
	Repo   string
	Suffix string
}

type Manifest struct {
	// Global custom domain; can be overridden by --domain flag.
	Domain   string     `json:"domain"`
	Packages []PkgEntry `json:"packages"`
}

type PkgEntry struct {
	// Import suffix under the domain, e.g. "project" or "foo/bar".
	Suffix string `json:"suffix"`
	// Full backing repo URL, e.g. "github.com/hungtrd/project".
	Repo string `json:"repo"`
}

//------------------------------------------------------------
// Cobra root & sub‑commands
//------------------------------------------------------------

var rootCmd = &cobra.Command{
	Use:   "vanityimport",
	Short: "Utilities for Go vanity imports",
	Long:  "Generate vanity import pages and rewrite import paths in a Go codebase.",
}

//-------------------------------------------------- html cmd

var htmlCmd = &cobra.Command{
	Use:   "html",
	Short: "Generate a single index.html for one vanity import path",
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		repo, _ := cmd.Flags().GetString("repo")
		outDir, _ := cmd.Flags().GetString("out")

		if domain == "" || repo == "" {
			return fmt.Errorf("--domain and --repo are required")
		}
		return generateHTMLForRepo(domain, repo, outDir)
	},
}

//-------------------------------------------------- rewrite cmd

var rewriteCmd = &cobra.Command{
	Use:   "rewrite",
	Short: "Rewrite Go import paths in source files",
	RunE: func(cmd *cobra.Command, args []string) error {
		oldPath, _ := cmd.Flags().GetString("old")
		newPath, _ := cmd.Flags().GetString("new")
		dir, _ := cmd.Flags().GetString("dir")

		if oldPath == "" || newPath == "" {
			return fmt.Errorf("--old and --new are required")
		}
		return rewriteImports(dir, oldPath, newPath)
	},
}

//-------------------------------------------------- build cmd

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate HTML pages for every package specified in a manifest file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := cmd.Flags().GetString("config")
		outDir, _ := cmd.Flags().GetString("out")
		overrideDomain, _ := cmd.Flags().GetString("domain")

		if cfg == "" {
			return fmt.Errorf("manifest --config is required")
		}
		m, err := loadManifest(cfg)
		if err != nil {
			return err
		}
		// allow CLI flag to override domain
		if overrideDomain != "" {
			m.Domain = overrideDomain
		}
		if m.Domain == "" {
			return fmt.Errorf("domain is missing (manifest or flag)")
		}
		for _, p := range m.Packages {
			if err := generateHTML(m.Domain, p.Suffix, p.Repo, outDir); err != nil {
				return fmt.Errorf("generate %s: %w", p.Suffix, err)
			}
		}
		return nil
	},
}

func init() {
	// html
	htmlCmd.Flags().StringP("domain", "d", "", "custom domain (e.g. go.example.com)")
	htmlCmd.Flags().StringP("repo", "r", "", "VCS repository path (e.g. github.com/user/project)")
	htmlCmd.Flags().StringP("out", "o", ".", "output directory for index.html")

	// rewrite
	rewriteCmd.Flags().StringP("old", "o", "", "old import prefix to replace (e.g. github.com/user/project)")
	rewriteCmd.Flags().StringP("new", "n", "", "new import prefix (e.g. go.example.com/project)")
	rewriteCmd.Flags().StringP("dir", "d", ".", "directory to scan recursively for .go files")

	// build
	buildCmd.Flags().StringP("config", "c", "vanity.json", "path to manifest file (json)")
	buildCmd.Flags().StringP("out", "o", "public", "output directory for generated pages")
	buildCmd.Flags().StringP("domain", "d", "", "override domain defined in manifest")

	rootCmd.AddCommand(htmlCmd, rewriteCmd, buildCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

//------------------------------------------------------------
// Manifest helper
//------------------------------------------------------------

func loadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

//------------------------------------------------------------
// HTML generation helpers
//------------------------------------------------------------

// generateHTMLForRepo keeps old CLI compatibility (html command).
func generateHTMLForRepo(domain, repo, outDir string) error {
	parts := strings.Split(strings.TrimSuffix(repo, "/"), "/")
	if len(parts) < 3 {
		return fmt.Errorf("repo should be in the form github.com/user/project")
	}
	suffix := parts[len(parts)-1]
	return generateHTML(domain, suffix, repo, outDir)
}

// generateHTML creates an index.html with vanity meta tags at outDir/<suffix>/index.html.
func generateHTML(domain, suffix, repo, outDir string) error {
	data := tplData{Domain: domain, Repo: repo, Suffix: suffix}

	fullDir := filepath.Join(outDir, filepath.FromSlash(suffix))
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(fullDir, "index.html")
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := template.Must(template.New("vanity").Parse(htmlTmpl)).Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Generated %s\n", filePath)
	return nil
}

//------------------------------------------------------------
// Import‑rewriter (unchanged logic except helper refactor)
//------------------------------------------------------------

func rewriteImports(root, oldPath, newPath string) error {
	fset := token.NewFileSet()
	absRoot, _ := filepath.Abs(root)

	return filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != absRoot && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || filepath.Ext(path) != ".go" {
			return nil
		}

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
