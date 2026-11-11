# 04 · Errors are values (and panics are bugs)

> **Goal:** replace your `try/except` instincts with Go's explicit error flow.
> This is the single biggest mindset change coming from Python.

## The pattern

```python
try:
    g = repo.get(id)
except NotFound:
    return 404
```

```go
g, err := repo.Get(ctx, id)
if err != nil {
    return err            // or handle it
}
// use g
```

`error` is a built-in interface with one method, `Error() string`. Functions
that can fail return it as their **last** value. You check it immediately.
Yes, `if err != nil` appears a lot. The upside: every failure path is
visible in the code you're reading, and there is no "which line threw?".

## Adding context: wrapping with `%w`

A bare error deep in a stack ("connection refused") is useless in a log.
Each layer adds what it was doing:

```go
// repository.go
return Greeting{}, fmt.Errorf("get %q: %w", id, ErrNotFound)

// service.go
return Greeting{}, fmt.Errorf("create greeting: %w", err)
```

Final message: `create greeting: get "abc": greeting not found`. That reads
like a traceback, built by hand. `%w` (not `%v`) keeps the original error
inside, so it can still be inspected. It is Python's `raise X from e`.

## Inspecting errors: `errors.Is` and `errors.As`

[`handler.go`](../internal/greeting/handler.go) maps errors to status codes in
one place:

```go
var vErr *ValidationError
switch {
case errors.As(err, &vErr):          // except ValidationError as vErr:
    httpx.Error(w, r, 422, vErr.Error())
case errors.Is(err, ErrNotFound):    // except NotFound:
    httpx.Error(w, r, 404, "greeting not found")
default:                             // except Exception:
    log it; 500
}
```

- `errors.Is(err, target)` → is `target` anywhere in the wrap chain? Use
  for **sentinel errors** (`var ErrNotFound = errors.New(...)`).
- `errors.As(err, &ptr)` → is there an error of this **type** in the chain?
  If so, fill `ptr`. Use for error types carrying data (`ValidationError`
  has `Field` and `Reason`).

Never compare messages with `err.Error() == "..."`. Never `==` a wrapped
error; use `errors.Is`.

## Handle once

An error should be *either* logged *or* returned, not both. Otherwise the same
failure appears five times in your logs. In this project, inner layers wrap
and return; only the HTTP layer (`writeError`) logs unexpected errors.

## Multiple errors

`errors.Join(a, b, c)` combines errors ([`config.go`](../internal/config/config.go)
uses it to report all bad env vars at once). `errors.Is` searches inside it.

## Panics

`panic` exists, but it is for **programmer bugs and impossible states**:
nil map writes, out-of-range indexes, a failed invariant at startup. You do not
`panic` for "user not found" or "invalid input".

`recover()` inside a deferred function stops a panic. Server code uses it at
exactly one place, the outermost boundary, so a bug in one request doesn't
take down the process. See [`recoverer.go`](../internal/middleware/recoverer.go)
and try `curl localhost:8080/api/v1/panic` in development.

## Python → Go cheat sheet

| Python                           | Go |
|----------------------------------|----|
| `raise ValueError("x")`          | `return errors.New("x")` |
| `class NotFound(Exception)`      | `var ErrNotFound = errors.New("not found")` |
| exception with attributes        | `type ValidationError struct{...}` + `Error()` method |
| `raise X from e`                 | `fmt.Errorf("doing x: %w", e)` |
| `except NotFound`                | `errors.Is(err, ErrNotFound)` |
| `except ValidationError as e`    | `errors.As(err, &e)` |
| `finally`                        | `defer` |
| top-level crash handler          | `recover()` in middleware |

## Exercise

Add `ErrDuplicate` to `greeting.go`, return it (wrapped) from
`MemoryRepository.Create` when the ID already exists, and map it to
`409 Conflict` in `writeError`. Add a test case.

Next: [05 · Project layout](05-project-layout.md)
