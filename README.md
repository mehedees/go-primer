# go-primer

A Go primer for experienced Python developers, taught through **one small,
production-shaped HTTP service** built with [chi](https://github.com/go-chi/chi)
and [zerolog](https://github.com/rs/zerolog).

There are no toy snippets: every lesson in [`docs/`](docs/) points at real code
in this repository, and every Go file is commented with Python comparisons.

---

## Quick start

Requirements: Go 1.24+ (`go version`), optionally Docker.

```bash
cp .env.example .env     # optional: local settings
make run                 # starts on :8080 with coloured dev logs
```

```bash
curl 'localhost:8080/api/v1/hello?name=Ada&lang=es'
# {"message":"¡Hola, Ada!"}

curl -i -X POST localhost:8080/api/v1/greetings \
  -H 'Content-Type: application/json' \
  -d '{"name":"Rob","language":"fr"}'
# 201 Created, Location: /api/v1/greetings/<id>, X-Request-Id: ...

curl localhost:8080/api/v1/greetings
curl localhost:8080/api/v1/panic      # dev only: see the Recoverer middleware work
```

Press `Ctrl+C` to see graceful shutdown in the logs.

Other tasks: `make help`

| Command            | What it does |
|--------------------|--------------|
| `make test`        | all tests with the race detector |
| `make cover`       | coverage report in your browser |
| `make check`       | fmt + tidy + vet/lint + test: run before pushing |
| `make build`       | static binary at `bin/api` |
| `make docker-build`/`docker-run` | ~10 MB distroless image |

## API

| Method | Path                         | Description |
|--------|------------------------------|-------------|
| GET    | `/healthz`                   | liveness probe (returns version) |
| GET    | `/readyz`                    | readiness probe (503 while shutting down) |
| GET    | `/api/v1/hello?name=&lang=`  | render a greeting (`en`, `es`, `fr`, `bn`) |
| GET    | `/api/v1/greetings`          | list stored greetings, newest first |
| POST   | `/api/v1/greetings`          | create `{"name": "...", "language": "en"}` |
| GET    | `/api/v1/greetings/{id}`     | fetch one |

Errors always look like:

```json
{"error":"Unprocessable Entity","message":"name: is required","request_id":"host/abc-000005"}
```

## Project layout

```
cmd/api/main.go            entry point: config → logger → wiring → run
internal/
  config/                  env vars → typed Config (fail fast)
  logger/                  zerolog setup, request-scoped logger helper
  middleware/              RequestLogger (access log + ctx logger), Recoverer
  httpx/                   JSON responses, error envelope, generic DecodeJSON[T]
  health/                  /healthz and /readyz
  greeting/                feature package: handler → service → repository
  server/                  chi router + http.Server with graceful shutdown
docs/                      the lessons
Dockerfile, Makefile, .env.example
```

Request flow:

```
client
  │
  ▼
http.Server (timeouts)                                     server/server.go
  │  RequestID → RealIP → RequestLogger → Recoverer → Timeout → CleanPath
  ▼
greeting.Handler   parse HTTP, map errors → status codes    greeting/handler.go
  ▼
greeting.Service   validation & business rules              greeting/service.go
  ▼
greeting.Repository (interface) ← MemoryRepository          greeting/repository.go
```

## Lessons

Read in order. Each ends with a hands-on exercise that modifies this codebase.

| #  | Lesson | Key ideas |
|----|--------|-----------|
| 01 | [Toolchain and modules](docs/01-toolchain-and-modules.md) | `go` commands vs pip/pytest/black, `go.mod`, no virtualenvs |
| 02 | [Language basics](docs/02-language-basics.md) | zero values, `:=`, multiple returns, `defer`, exported names |
| 03 | [Types and interfaces](docs/03-types-and-interfaces.md) | structs, methods, pointers, slices, maps, implicit interfaces |
| 04 | [Errors](docs/04-errors.md) | errors as values, `%w`, `errors.Is/As`, panic vs error |
| 05 | [Project layout](docs/05-project-layout.md) | `cmd/`, `internal/`, package-by-feature, dependency direction |
| 06 | [HTTP, chi and middleware](docs/06-http-chi-and-middleware.md) | `http.Handler`, routing, middleware onion, context values |
| 07 | [Logging with zerolog](docs/07-logging-with-zerolog.md) | structured logs, child loggers, request correlation |
| 08 | [Config and graceful shutdown](docs/08-config-and-graceful-shutdown.md) | 12-factor config, timeouts, SIGTERM, draining |
| 09 | [Generics](docs/09-generics.md) | type parameters, constraints, when not to use them |
| 10 | [Concurrency](docs/10-concurrency.md) | goroutines, mutexes, channels, `select`, `context`, race detector |
| 11 | [Testing](docs/11-testing.md) | table-driven tests, `httptest`, fakes over mocks |
| 12 | [Building and deploying](docs/12-building-and-deploying.md) | static binaries, ldflags, distroless Docker, probes |
| 13 | [Gotchas and next steps](docs/13-gotchas-and-next-steps.md) | common traps, capstone exercises, resources |

**Before lesson 01**, spend two hours on [A Tour of Go](https://go.dev/tour/).
These lessons assume you've seen the syntax once and focus on *why* and on
production practice.

## Configuration

All settings come from environment variables. See
[`.env.example`](.env.example) and [lesson 08](docs/08-config-and-graceful-shutdown.md).

| Variable | Default |
|----------|---------|
| `APP_ENV` | `development` (`production` = JSON logs, no `/panic`) |
| `LOG_LEVEL` | `info` |
| `HTTP_ADDR` | `:8080` |
| `HTTP_READ_TIMEOUT` / `HTTP_WRITE_TIMEOUT` / `HTTP_IDLE_TIMEOUT` | `5s` / `10s` / `60s` |
| `HTTP_REQUEST_TIMEOUT` | `8s` |
| `HTTP_SHUTDOWN_TIMEOUT` | `15s` |

## Editor setup

Install the official Go extension (VS Code) or use GoLand. Both run `gopls`
(the Go language server) and `gofmt` on save. Turn format-on-save on and never
think about formatting again.
