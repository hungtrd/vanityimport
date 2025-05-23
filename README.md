# vanity

*A tiny CLI to generate Go vanity‑import pages and mass‑rewrite import paths in your source tree.*

---

## ✨ Features

| Capability           | Description                                                                                                                                                                  |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`vanityimport html`**    | Generate an `index.html` with the correct `<meta>` tags for a custom domain (e.g. `go.example.com/project`).                                                                 |
| **`vanityimport rewrite`** | Recursively scan a directory and rewrite matching `import` paths from a VCS host (e.g. `github.com/user/project`) to your vanity domain. Skips hidden folders and `vendor/`. |

## 🚀 Installation

> Requires **Go 1.20+**.

```bash
go install github.com/hungtrd/vanityimport@latest  # once published
# or clone & build locally

git clone https://github.com/hungtrd/vanityimport.git
cd vanityimport
go build -o vanityimport .
```

The binary `vanityimport` will be placed in the current directory. Add it to your `$PATH` if desired.

## 🛠 Usage

```bash
vanityimport <command> [flags]
```

### 1. Generate an HTML vanity page

```bash
vanityimport html \
  --domain go.example.com \
  --repo   github.com/user/project \
  --out    ./site          # default: current directory
```

*Creates `./site/index.html` containing the necessary `go-import` & `go-source` meta tags + a redirect to pkg.go.dev.*

Once generated, push the `index.html` file (or the whole `site/` folder) to the branch/ repo served by **GitHub Pages** and configure your custom domain. Go tools can now `go get go.example.com/project` over HTTPS.

### 2. Rewrite import paths in existing code

```bash
vanityimport rewrite \
  --old github.com/user/project \
  --new go.example.com/project \
  --dir ./                # default: .
```

*Recursively processes all `.go` files (excluding dot‑files and `vendor/`) and rewrites import specs using the Go AST to ensure syntactic correctness.*

---

## Flags summary

| Command   | Flag           | Default        | Description                                    |
| --------- | -------------- | -------------- | ---------------------------------------------- |
| `html`    | `-d, --domain` | **(required)** | Vanity domain prefix (e.g. `go.example.com`).  |
|           | `-r, --repo`   | **(required)** | VCS repo URL (e.g. `github.com/user/project`). |
|           | `-o, --out`    | `.`            | Output directory for `index.html`.             |
| `rewrite` | `-o, --old`    | **(required)** | Import prefix to search for.                   |
|           | `-n, --new`    | **(required)** | Replacement vanity prefix.                     |
|           | `-d, --dir`    | `.`            | Directory to scan recursively.                 |

---

## Example workflow

1. **Create vanity page** for your project:

   ```bash
   vanityimport html -d go.hung.rocks -r github.com/hungtrd/project -o ./docs
   ```
2. **Publish** the `docs/` folder via GitHub Pages and point DNS `CNAME` `go.hung.rocks` ➜ `username.github.io`.
3. **Rewrite code base** to use the new domain:

   ```bash
   vanityimport rewrite -o github.com/hungtrd/project -n go.hung.rocks/project -d .
   ```
4. Commit & push changes. From now on, users can:

   ```bash
   go install go.hung.rocks/project/cmd/app@latest
   ```

---

## Contributing

Pull requests and issues are welcome! Please run `go vet ./...` and `go test ./...` before submitting PRs.

---

## License

Distributed under the MIT License. See `LICENSE` for more information.
