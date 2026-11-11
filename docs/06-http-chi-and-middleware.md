# 06 · HTTP, chi and middleware

> **Goal:** understand a Go HTTP request from socket to JSON, and write
> middleware confidently.

## net/http in two interfaces

Everything HTTP in Go hangs off one interface:

```go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

and a convenience adapter so plain functions qualify:

```go
type HandlerFunc func(w http.ResponseWriter, r *http.Request)
```

So a handler is any `func(w, r)`. You **write** the response into `w` (headers,
status, body), you don't return it. Compare to a Flask view that returns a
response object, or WSGI's `start_response`.

```go
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    ...
    httpx.JSON(w, r, http.StatusOK, helloResponse{Message: msg})
}
```

Order of operations on `w` matters: set headers → `WriteHeader(status)` →
`Write(body)`. After the first body write, headers are frozen.

The stdlib *is* the production server: there is no gunicorn/uvicorn layer.
Each request runs in its own goroutine, so handlers are concurrent by default.

## Why chi?

Go 1.22's `http.ServeMux` gained method and path-parameter routing, so tiny
services can skip a router entirely. chi adds what you want in a real API,
while staying 100% `net/http`-compatible (every chi handler is an
`http.Handler`, every stdlib middleware works):

- route groups and sub-routers with their own middleware (`r.Route`, `r.Mount`)
- a solid library of middleware (`RequestID`, `RealIP`, `Timeout`, ...)
- URL params: `chi.URLParam(r, "id")`

Compared with FastAPI: no automatic validation, OpenAPI or DI. You trade magic
for explicitness. Gin/Echo/Fiber are alternatives; chi is the one that stays
closest to the standard library.

## The router in this project

[`server.go`](../internal/server/server.go):

```go
r := chi.NewRouter()
r.Use(chimw.RequestID)
r.Use(chimw.RealIP)
r.Use(middleware.RequestLogger(d.Logger))
r.Use(middleware.Recoverer)
r.Use(chimw.Timeout(d.Config.HTTP.RequestTimeout))
r.Use(chimw.CleanPath)

r.Get("/healthz", hh.Live)
r.Get("/readyz",  hh.Ready)

r.Route("/api/v1", func(r chi.Router) {
    r.Use(chimw.AllowContentType("application/json"))   // only for /api/v1/*
    r.Get("/hello", d.Greeting.Hello)
    r.Mount("/greetings", d.Greeting.Routes())          // a sub-router
})
```

| Endpoint                          | Handler                     |
|-----------------------------------|-----------------------------|
| `GET  /healthz`, `/readyz`        | `health.Handler`            |
| `GET  /api/v1/hello?name=&lang=`  | `greeting.Handler.Hello`    |
| `GET  /api/v1/greetings`          | list                        |
| `POST /api/v1/greetings`          | create (JSON body)          |
| `GET  /api/v1/greetings/{id}`     | get by id                   |
| `GET  /api/v1/panic` (dev only)   | deliberate panic            |

## Middleware

A middleware is a function `func(http.Handler) http.Handler`. Python
decorator analogy:

```python
def timing(view):
    @functools.wraps(view)
    def wrapper(request):
        start = time.time()
        resp = view(request)          # call the next layer
        log(time.time() - start)
        return resp
    return wrapper
```

```go
func Timing(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)          // call the next layer
        log.Println(time.Since(start))
    })
}
```

If middleware needs configuration (a logger, a timeout), add an outer
function that returns the middleware: `RequestLogger(logger)` returns a
`func(http.Handler) http.Handler`. That's a decorator factory, like
`@retry(times=3)`.

### Order: the onion

`r.Use` calls wrap in order: the first `Use` is the outermost layer.

```
request ─► RequestID ─► RealIP ─► RequestLogger ─► Recoverer ─► Timeout ─► handler
response ◄─────────────────────────────────────────────────────────────────┘
```

- `RequestID` first: everything after can use the ID.
- `RequestLogger` **outside** `Recoverer`: when a handler panics,
  `Recoverer` writes a 500, and the logger still sees and logs that 500.
  Swap them and panicked requests vanish from your access log.

### Passing data down: `context.Context`

Middleware enriches the request by putting values in its context:

```go
r = r.WithContext(l.WithContext(r.Context()))   // store logger in ctx
...
logger.FromContext(r.Context()).Info().Msg("greeting created")  // later, in a handler
```

This is how request-scoped data travels in Go (like Flask's `g` or a
`contextvars.ContextVar`, but passed explicitly). Use it for request-scoped
metadata (request ID, logger, authenticated user), **not** for optional
function arguments.

### Capturing the status code

`http.ResponseWriter` won't tell you what status was written. So
`RequestLogger` wraps it with `chimw.NewWrapResponseWriter` and reads
`ww.Status()` after `next` returns. Wrapping `w` is the standard trick for any
middleware that inspects responses (metrics, compression, caching).

### Short-circuiting

To reject a request (auth failure, rate limit), write a response and **don't
call** `next.ServeHTTP`. That is all.

## JSON in and out

[`httpx.go`](../internal/httpx/httpx.go):

- `httpx.JSON` sets `Content-Type`, the status, and encodes any value.
- `httpx.DecodeJSON[T]` caps the body size (`MaxBytesReader`), rejects unknown
  fields (catches client typos), and checks there's only one JSON object.
  pydantic did all this for you; here you see each guard explicitly.

## Exercises

1. Write an `APIKey(key string)` middleware that returns 401 unless the
   `X-API-Key` header matches. Apply it **only** to the `/api/v1` group.
   Probes must remain open.
2. Add `DELETE /api/v1/greetings/{id}`: repository → service → handler → route.
3. Move `chimw.Timeout` below `CleanPath` and think about whether it matters.

Next: [07 · Structured logging with zerolog](07-logging-with-zerolog.md)
