# 07 · Structured logging with zerolog

> **Goal:** produce logs a production log system can query, with every line
> from one request tied together by `request_id`.

## Structured, not formatted

```python
logger.info(f"created greeting {g.id} for {name}")          # a sentence
logger.info("greeting created", extra={"greeting_id": g.id}) # structured
```

In production you search logs by field (`request_id="..." AND status>=500`), so
we log **fields + a constant message**:

```go
log.Info().Str("greeting_id", g.ID).Msg("greeting created")
```

Output in production (`APP_ENV=production`):

```json
{"level":"info","service":"go-primer-api","version":"dev","request_id":"host/abc-000003","method":"POST","path":"/api/v1/greetings","greeting_id":"835a24caa8fa972f","time":"2026-11-09T01:46:18.815Z","message":"greeting created"}
```

In development the same event prints as coloured, aligned console text.

## Why zerolog

Zero allocations per log line in the common path, a chained API, JSON by
default. Alternatives: stdlib `log/slog` (Go 1.21+, increasingly standard,
similar shape), `zap` (Uber). The concepts in this lesson transfer to all of
them.

## The API

```go
log.Info().                       // start an event at a level (nil if level disabled → ~free)
    Str("name", name).            // typed field helpers:
    Int("status", 200).           //   Str, Int, Bool, Dur, Time, Err, Interface...
    Err(err).                     //   adds "error": err.Error()
    Msg("something happened")     // MUST end with Msg/Msgf/Send, or nothing is written
```

Forgetting `.Msg(...)` is the classic zerolog bug: the event is built and
silently dropped.

Levels, lowest to highest: `trace, debug, info, warn, error, fatal, panic`.
Avoid `Fatal()` in libraries and handlers: it calls `os.Exit(1)`, skipping
deferred cleanup and graceful shutdown.

## Child loggers: context fields

```go
l := base.With().
    Str("request_id", reqID).
    Str("method", r.Method).
    Logger()
```

`With()` creates a new logger that stamps those fields on **every** event.
That is how `service`/`version` get on every line
([`logger.go`](../internal/logger/logger.go)) and how `request_id` gets on every
line of a request ([`logging.go`](../internal/middleware/logging.go)).

## The request-scoped logger pattern

1. `chi`'s `RequestID` middleware puts an ID in the context.
2. Our `RequestLogger` builds a child logger with that ID and stores it in the
   context (`l.WithContext(ctx)`).
3. Anything downstream calls `logger.FromContext(r.Context())` and gets the
   correlated logger. No globals, no passing loggers through every function.
4. When the handler returns, `RequestLogger` writes the access-log line,
   choosing the level by status: 5xx → `error`, 4xx → `warn`, else `info`.

Run the server, make a request, and see `request_id` on both the
"greeting created" line and the "request completed" line. The same ID comes back
in the `X-Request-Id` response header and in error bodies, so a user's bug
report leads you straight to the logs.

## What NOT to log

Passwords, tokens, full request bodies, personal data. Once it's shipped to a
log store it is very hard to delete.

## Exercise

1. Add `Str("lang", string(lang))` to the "greeting created" log.
2. Run with `LOG_LEVEL=warn go run ./cmd/api` and confirm only the 4xx/5xx
   access lines appear.
3. Run in production mode and filter the JSON with `jq`:
   `APP_ENV=production go run ./cmd/api | jq 'select(.status >= 400)'`.

(We use `go run` directly here because `make run` loads `.env`, whose values
would override variables set on the command line.)

Next: [08 · Configuration and graceful shutdown](08-config-and-graceful-shutdown.md)
