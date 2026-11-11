# 12 · Building and deploying

> **Goal:** ship a small, secure container and know what each build flag does.

## One static binary

```bash
make build      # → bin/api, ~6 MB
./bin/api
```

That file is the entire application: no interpreter, no site-packages, no
virtualenv. Copy it to any Linux box of the same architecture and it runs.

```makefile
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/api ./cmd/api
```

| Flag              | Purpose |
|-------------------|---------|
| `CGO_ENABLED=0`   | pure Go, no libc dependency → fully static binary |
| `-trimpath`       | remove your local file paths from the binary |
| `-ldflags -s -w`  | strip symbol/debug tables (smaller) |
| `-X main.version=`| set the `version` variable in `main` at link time |

`VERSION` comes from `git describe`, and shows up in every log line and in
`/healthz`. Commit and tag (`git tag v0.1.0`) to see it change.

### Cross-compiling

```bash
GOOS=linux GOARCH=amd64 go build -o bin/api-linux-amd64 ./cmd/api
GOOS=windows GOARCH=arm64 go build -o bin/api.exe ./cmd/api
```

From your Mac, no Docker or toolchain installs needed. That's a superpower
compared with Python wheels and native extensions.

## The Dockerfile

[`Dockerfile`](../Dockerfile) is multi-stage:

1. **build** (`golang:1.24-alpine`): copy `go.mod`/`go.sum`, `go mod download`
   (cached layer), copy source, build.
2. **runtime** (`gcr.io/distroless/static-debian12:nonroot`): copy only the
   binary. No shell, no package manager, runs as a non-root user.

```bash
make docker-build
make docker-run
curl localhost:8080/healthz
```

The final image is roughly 10 MB, compared to 150 MB+ for a slim Python image.
Fewer files means fewer CVEs to patch.

Because there's no shell in distroless, `docker exec -it ... sh` won't work.
That's intentional. Debug with logs, or temporarily use the `:debug` tag.

## Running in an orchestrator

What this service already does right for Kubernetes/ECS/Nomad/Cloud Run:

- config only from env vars
- logs JSON to stdout (`APP_ENV=production`)
- `/healthz` for liveness, `/readyz` for readiness
- handles SIGTERM: readiness → 503, drain in-flight, exit 0
- runs as non-root

Example Kubernetes probe config:

```yaml
livenessProbe:  { httpGet: { path: /healthz, port: 8080 } }
readinessProbe: { httpGet: { path: /readyz,  port: 8080 } }
terminationGracePeriodSeconds: 30   # > HTTP_SHUTDOWN_TIMEOUT (15s)
```

## Next production steps (not built here)

| Concern            | Typical Go choice |
|--------------------|-------------------|
| Linting            | `golangci-lint` (aggregates staticcheck, errcheck, revive ...) |
| CI                 | GitHub Actions running `make check` |
| Metrics            | `prometheus/client_golang` + a metrics middleware |
| Tracing            | OpenTelemetry (`otelhttp` wraps handlers) |
| Database           | `jackc/pgx` (Postgres) + `sqlc` (generate typed Go from SQL) |
| Migrations         | `pressly/goose` or `golang-migrate` |
| API docs           | OpenAPI via `oapi-codegen` (spec-first) or `swaggo/swag` |
| Profiling          | `net/http/pprof` on an internal port |

Next: [13 · Gotchas and next steps](13-gotchas-and-next-steps.md)
