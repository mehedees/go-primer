# 08 · Configuration and graceful shutdown

> **Goal:** a service that is configured by its environment and stops without
> dropping requests.

## Config from the environment (12-factor)

[`config.go`](../internal/config/config.go) reads env vars into a typed
`Config` struct, once, at startup in `main`. Then the struct is *passed* to
whatever needs it. Nothing else reads `os.Getenv`.

| Variable                 | Default        | Meaning |
|--------------------------|----------------|---------|
| `APP_ENV`                | `development`  | `development` (console logs, `/panic` route) or `production` (JSON logs) |
| `LOG_LEVEL`              | `info`         | trace/debug/info/warn/error |
| `HTTP_ADDR`              | `:8080`        | listen address |
| `HTTP_READ_TIMEOUT`      | `5s`           | max time to read request (headers + body) |
| `HTTP_WRITE_TIMEOUT`     | `10s`          | max time to write the response |
| `HTTP_IDLE_TIMEOUT`      | `60s`          | keep-alive idle time |
| `HTTP_REQUEST_TIMEOUT`   | `8s`           | per-request context deadline (chi `Timeout`) |
| `HTTP_SHUTDOWN_TIMEOUT`  | `15s`          | max time to drain in-flight requests |

Durations use Go syntax: `250ms`, `5s`, `1m30s`.

Design choices worth noting:

- **Fail fast:** invalid config returns an error and the process exits before
  listening. All problems are reported together via `errors.Join`.
- **No library:** with ~8 variables, hand-written parsing is clearer. For
  bigger configs, `github.com/caarlos0/env` gives pydantic-settings-style
  struct tags (`env:"HTTP_ADDR" envDefault:":8080"`).
- **Locally:** `cp .env.example .env`; `make run` exports it. In containers
  the orchestrator sets real env vars.

## Why timeouts matter

An `http.Server{}` with no timeouts will wait forever for a slow client
(slowloris attack, stuck mobile connection). Every production server sets
`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` and `IdleTimeout`. Gunicorn
had `--timeout` for you; here it's your job.

`chimw.Timeout` is different: it puts a **deadline on the request context**.
Code that respects `ctx` (DB drivers, HTTP clients) will abort when it fires.
That's why every service and repository method takes `ctx context.Context`
first.

## Graceful shutdown, step by step

When Kubernetes or `docker stop` wants your container gone, it sends
**SIGTERM**, waits (default 30s), then sends SIGKILL. In that window you must:
stop taking new work, finish in-flight requests, release resources.

[`main.go`](../cmd/api/main.go):

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
...
srv.Run(ctx)
```

`signal.NotifyContext` returns a context that gets cancelled when a signal
arrives. Then [`server.go`](../internal/server/server.go) `Run`:

```go
go func() { errCh <- s.http.ListenAndServe() }()   // serve in background

select {
case err := <-errCh:   // failed to listen (port in use, ...)
    return err
case <-ctx.Done():     // SIGTERM / Ctrl+C
}

s.health.SetReady(false)                          // 1. /readyz -> 503
shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
defer cancel()
s.http.Shutdown(shutdownCtx)                      // 2. stop accepting, wait for in-flight
```

`Shutdown` closes listeners immediately, then waits until active requests
finish or `shutdownCtx` expires. Keep `HTTP_SHUTDOWN_TIMEOUT` below your
orchestrator's grace period.

### The `run() error` pattern

`main` calls `run()` and only translates the error into an exit code. Why?
`os.Exit` and `log.Fatal` exit **immediately**, skipping every `defer`.
Returning errors up to `main` lets cleanup run.

## Try it

```bash
make run
# in another terminal: a request that takes a while, then Ctrl+C the server
```

You'll see `shutdown signal received` → `http server stopped took=...` →
`bye`.

## Exercise

Add a slow endpoint (`time.Sleep(5 * time.Second)` but written as a `select`
on `time.After(5*time.Second)` and `r.Context().Done()`, see
[10 · Concurrency](10-concurrency.md)). Call it, press Ctrl+C during the
request, and observe that the response still arrives before shutdown completes.
Then set `HTTP_REQUEST_TIMEOUT=2s` and see the context cancel the work.

Next: [09 · Generics](09-generics.md)
