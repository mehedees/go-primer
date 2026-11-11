# 01 · Toolchain, modules and dependencies

> **Goal:** know what every `go` command does and how dependencies work,
> mapped to the Python tools you already use.

## The one tool: `go`

In Python you juggle `python`, `pip`, `venv`/`poetry`/`uv`, `pytest`, `black`,
`mypy`, `pyinstaller`. Go ships almost all of that as subcommands of one binary:

| Python                         | Go                         | What it does                                  |
|--------------------------------|----------------------------|-----------------------------------------------|
| `python app.py`                | `go run ./cmd/api`         | compile to a temp binary and run it           |
| `pyinstaller` / wheel          | `go build ./cmd/api`       | produce a native executable                   |
| `pytest`                       | `go test ./...`            | run tests (`./...` = this dir and all below)  |
| `black` / `ruff format`        | `gofmt -w .`               | format code. **Not optional, not configurable** |
| `mypy` / `ruff check`          | `go vet ./...` (+ compiler) | static checks                                |
| `pip install x`                | `go get x@version`         | add/upgrade a dependency                      |
| `pip-compile` / `poetry lock`  | `go mod tidy`              | sync `go.mod`/`go.sum` with your imports      |
| `pydoc`                        | `go doc net/http.Server`   | read docs from the terminal                   |

Try right now:

```bash
go doc github.com/go-chi/chi/v5 Router
go doc -all ./internal/greeting | less
```

## Modules: `go.mod` and `go.sum`

Open [`go.mod`](../go.mod). It is the `pyproject.toml` of this project:

```go
module github.com/mehedees/go-primer   // the import path prefix for our packages

go 1.24.3                               // minimum language version

require (
    github.com/go-chi/chi/v5 v5.5.0      // direct dependency
    github.com/rs/zerolog v1.35.1
    golang.org/x/sys v0.29.0 // indirect  // a dependency of a dependency
)
```

`go.sum` holds cryptographic hashes of every module version, like the hashes
in a `poetry.lock`/`uv.lock`. **Commit both files.**

### There is no virtualenv

Downloaded modules live in a shared, read-only cache (`go env GOMODCACHE`,
usually `~/go/pkg/mod`). Each project's `go.mod` pins exact versions, so two
projects can use different versions of chi side by side without any
activation step. This removes a whole category of "wrong venv" bugs.

### Versions and the `/v5` suffix

Go modules use semantic versioning. When a library makes a breaking major
release (v2+), its **import path changes**: `github.com/go-chi/chi/v5`. That
means v4 and v5 can coexist in one build, and an accidental major upgrade
can't happen: you have to change your import lines.

### Minimal Version Selection

pip resolves "the newest version that satisfies all constraints". Go does the
opposite: it picks the **minimum** version that satisfies every `require`.
Result: builds are reproducible even without a lockfile, and upgrading is
always an explicit act (`go get -u ./...`).

## Imports map to directories

```go
import "github.com/mehedees/go-primer/internal/greeting"
```

That is: module path + directory. Every `.go` file in
`internal/greeting/` belongs to `package greeting`. There is no
`__init__.py`; a directory *is* a package. You refer to its contents as
`greeting.NewService(...)`, always qualified (like `import greeting` and never
`from greeting import *`).

Unused imports are a **compile error**. So are unused local variables. It feels
strict for a week, then you stop noticing.

## Exercise

1. Run `go list -m all` to see the full dependency graph.
2. Run `go mod why golang.org/x/sys` to see *why* an indirect dep is needed.
3. Add `"strings"` to the imports of `cmd/api/main.go` without using it and
   run `go build ./...` to see the error. Remove it.

Next: [02 · Language basics for Pythonistas](02-language-basics.md)
