# vanityimport

*A lightweight CLI to generate Go **vanity‑import** pages and safely rewrite import paths — now with manifest‑driven multi‑package support.*

---

## ✨ Features

| Capability                | Description                                                                                                                                                                                         |
|---------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **`vanityimport build`**  | Read a **manifest** (`vanity.json` by default) and generate: <br>• one HTML file per package at `<out>/<suffix>/index.html` <br>• a root **index.html** listing all vanity packages. |
| **`vanityimport html`**   | Generate a single `index.html` for a custom domain + repo (handy for quick tests).                                                                                                                   |
| **`vanityimport rewrite`**| Recursively scan a directory and rewrite matching `import` paths from a VCS host (e.g. `github.com/user/project`) to your vanity domain. Skips hidden folders and `vendor/`.                           |

---

## 🚀 Installation

> Requires **Go 1.20+**.

```bash
go install go.hung.rocks/vanityimport@latest  # once published

# or clone & build locally

git clone https://github.com/hungtrd/vanityimport.git
cd vanityimport
go build -o vanityimport .
```

The binary **`vanityimport`** will be produced in the current directory. Add it to your `$PATH` if desired.

---

## 📦 Manifest format (`vanity.json`)

```json
{
  "domain": "go.hung.rocks",
  "packages": [
    { "suffix": "project",      "repo": "github.com/hungtrd/project" },
    { "suffix": "foo/bar",      "repo": "github.com/hungtrd/foo/bar" },
    { "suffix": "tools/runner", "repo": "github.com/hungtrd/runner" }
  ]
}
```
* `domain` – vanity root (can override with `--domain`).
* `suffix` – path segment after the domain → `go.hung.rocks/<suffix>`.
* `repo` – canonical VCS repository URL.

---

## 🛠 Usage

```bash
vanityimport <command> [flags]
```

### 1. Build all pages from manifest

```bash
vanityimport build \
  --config vanity.json \   # default
  --out    public          # default
```

*Creates `public/<suffix>/index.html` for every package **and** a `public/index.html` overview.*  
Deploy the **`public/`** folder (e.g. via GitHub Pages) and point DNS `CNAME` `go.hung.rocks` ➜ `username.github.io`.

### 2. Generate a single vanity page (ad‑hoc)

```bash
vanityimport html \
  --domain go.example.com \
  --repo   github.com/user/project \
  --out    ./site
```

### 3. Rewrite import paths in existing code

```bash
vanityimport rewrite \
  --old github.com/user/project \
  --new go.example.com/project \
  --dir ./
```

---

## 🔧 Flags summary

| Command      | Flag               | Default / Req. | Description                                                    |
|--------------|--------------------|----------------|----------------------------------------------------------------|
| `build`      | `-c, --config`     | `vanity.json`  | Manifest file path.                                             |
|              | `-o, --out`        | `public`       | Output directory for generated pages.                          |
|              | `-d, --domain`     | *(manifest)*   | Override domain defined in the manifest.                       |
| `html`       | `-d, --domain`     | **required**   | Vanity domain prefix (e.g. `go.example.com`).                  |
|              | `-r, --repo`       | **required**   | VCS repo URL (e.g. `github.com/user/project`).                 |
|              | `-o, --out`        | `.`            | Output directory for the generated `index.html`.               |
| `rewrite`    | `-o, --old`        | **required**   | Import prefix to search for.                                   |
|              | `-n, --new`        | **required**   | Replacement vanity prefix.                                     |
|              | `-d, --dir`        | `.`            | Directory to scan recursively (skips dot & `vendor/`).         |

---

## 🖥 Example workflow (GitHub Actions)

```yaml
name: Publish vanity pages

permissions:
  contents: write

on:
  push:
    paths: [vanity.json, '**/*.go']
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 1.22 }
      - run: go run . build --config vanity.json --out public
      - uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: public
          publish_branch: gh-pages
```

Push changes to **`vanity.json`** and the workflow rebuilds & deploys the site automatically.

---

## 🤝 Contributing

Pull requests and issues are welcome! Please run `go vet ./...` and `go test ./...` before submitting PRs.

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.
