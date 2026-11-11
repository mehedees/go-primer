# 05 · Production project layout

> **Goal:** know where code goes in a real Go service, and why.

## This project

```
go-primer/
├── cmd/
│   └── api/
│       └── main.go          # entry point: config → logger → wiring → run
├── internal/                # private application code (compiler-enforced)
│   ├── config/              # env-based settings
│   ├── logger/              # zerolog setup + ctx helper
│   ├── httpx/               # JSON/error/decode helpers shared by handlers
│   ├── middleware/          # our custom HTTP middleware
│   ├── health/              # /healthz, /readyz
│   ├── greeting/            # a FEATURE: model, repo, service, handler, tests
│   └── server/              # router assembly + http.Server lifecycle
├── docs/                    # you are here
├── Dockerfile               # multi-stage, distroless
├── Makefile                 # task runner
├── .env.example
├── go.mod / go.sum
```

## `cmd/` — one directory per binary

`cmd/api/main.go` builds a binary named `api`. Add `cmd/worker/` or
`cmd/migrate/` later and each builds independently, sharing `internal/`.
`main` stays thin: everything testable lives in `internal/`.

## `internal/` — the only Go directory name with special meaning

Packages under `internal/` can be imported **only** by code rooted at the
parent of `internal/`. Another module trying to import
`github.com/mehedees/go-primer/internal/greeting` gets a compile error.
It's Python's `_private` convention, but enforced. Put everything here
unless you deliberately want to publish a library API.

You'll see `pkg/` in some projects for "public library code". For an
application, you usually don't need it. Don't create it by reflex.

## Package by feature, layer inside

Many Python projects split by layer:

```
app/models/  app/services/  app/routers/  app/repositories/
```

Go prefers splitting by **feature** (also called domain):

```
internal/greeting/   handler.go  service.go  repository.go  greeting.go
internal/user/       handler.go  service.go  repository.go  user.go
```

Why it fits Go better:

1. **Packages are the unit of encapsulation.** Lowercase identifiers are
   private to the package. With a by-feature layout, `Service.repo`,
   `validate`, `randomID` stay hidden. With a by-layer layout, the service
   package would have to export everything the handler package needs.
2. **Import cycles are compile errors.** `models` ↔ `services` cycles are a
   common Python headache; Go simply refuses to build them. Feature packages
   that depend "downwards" avoid them naturally.
3. **Names read well.** `greeting.Service`, `greeting.NewHandler` instead of
   `services.GreetingService`. Avoid stutter: not `greeting.GreetingService`.

The layering (handler → service → repository) is still there, just inside
the package as files. When a feature grows large, split *that* feature into
sub-packages (`greeting/postgres` for a DB-backed repository, for example).

### Dependency direction

```
cmd/api ──► server ──► greeting ──► httpx, logger
                 └──► health   ──► httpx
                 └──► middleware ─► httpx, logger
         ──► config, logger
```

Arrows only go one way. `greeting` doesn't know `server` exists. `main`
is the **composition root**: the only place that knows the concrete types and
wires them together.

## Package naming rules

- short, lowercase, single word: `config`, `httpx`, not `http_utils`
- no `util`, `common`, `helpers`, `models` grab-bags; name by what it provides
- the directory name equals the package name

## Exercise

Sketch (on paper) where a `user` feature with Postgres storage would go.
Then compare with the layout used by a well-known Go service, e.g.
[github.com/grafana/grafana](https://github.com/grafana/grafana/tree/main/pkg)
or [github.com/gitea/gitea](https://github.com/go-gitea/gitea).

Next: [06 · HTTP, chi and middleware](06-http-chi-and-middleware.md)
