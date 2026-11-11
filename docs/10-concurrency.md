# 10 · Concurrency: goroutines, channels, context, sync

> **Goal:** understand the concurrency that's already running in this service,
> and use the four tools that cover 95% of real needs.

## Mental model shift from Python

| Python                                    | Go |
|-------------------------------------------|----|
| GIL: one thread runs Python at a time     | no GIL: goroutines run truly in parallel on all cores |
| threads are heavy (MBs, OS-managed)       | goroutines are cheap (~2 KB stack, millions OK) |
| `asyncio`: colored functions (`async`/`await`) | no colored functions: any function can block; the runtime parks the goroutine |
| `asyncio.Queue`                           | channel |
| `asyncio.gather` / `TaskGroup`            | `sync.WaitGroup` / `errgroup.Group` |
| task cancellation                         | `context.Context` |

The important consequence: **your HTTP handlers already run concurrently**.
`net/http` starts a goroutine per connection. Two simultaneous `POST`s execute
`MemoryRepository.Create` at the same time, in parallel, on different cores.
Shared mutable state needs protection. There's no GIL to save you.

## 1. Goroutines

```go
go doWork(x)            // start and don't wait. That's it.
go func() { ... }()     // anonymous function
```

A goroutine has no handle and no return value. You coordinate with the tools
below. Every goroutine you start must have a clear way to **end**; a goroutine
blocked forever is a leak.

## 2. `sync.Mutex` / `RWMutex`: protect shared state

[`repository.go`](../internal/greeting/repository.go):

```go
type MemoryRepository struct {
    mu    sync.RWMutex
    items map[string]Greeting
}

func (r *MemoryRepository) Create(...) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.items[g.ID] = g
}

func (r *MemoryRepository) Get(...) (Greeting, error) {
    r.mu.RLock()            // many readers at once
    defer r.mu.RUnlock()
    ...
}
```

For a single counter or flag, `sync/atomic` is lighter: `health.Handler`
uses `atomic.Bool` for readiness.

### The race detector

`go test -race` (what `make test` runs) instruments memory access and fails if
two goroutines touch the same memory without synchronisation.
`TestConcurrentCreates` in [`service_test.go`](../internal/greeting/service_test.go)
spawns 50 goroutines. **Try it:** comment out the `Lock`/`Unlock` lines in
`Create` and run `make test`. You'll get a `DATA RACE` report (or even a
`concurrent map writes` crash). Put them back.

## 3. Channels and `select`

A channel is a typed, thread-safe pipe. Sending blocks until there's room;
receiving blocks until there's a value.

```go
ch := make(chan int)      // unbuffered: send waits for a receiver
ch := make(chan int, 10)  // buffered: send waits only when full
ch <- 42                  // send
v := <-ch                 // receive
v, ok := <-ch             // ok == false when the channel is closed and drained
close(ch)                 // only the sender closes
for v := range ch {}      // receive until closed
```

`select` waits on several channel operations and runs the first one ready.
[`server.go`](../internal/server/server.go) uses it to wait for *either* a
listen error *or* a shutdown signal:

```go
select {
case err := <-errCh:
    return err
case <-ctx.Done():
    // shut down
}
```

> "Don't communicate by sharing memory; share memory by communicating."
> In practice: use channels to pass work and signals between goroutines; use a
> mutex to guard a shared struct. Both are idiomatic. Pick the simpler one.

## 4. `context.Context`: cancellation and deadlines

A `Context` is passed as the **first argument** to anything that does I/O or
may take a while. It carries:

- a **cancellation signal**: `ctx.Done()` is a channel closed on cancel
- an optional **deadline**: `chimw.Timeout` sets one per request
- request-scoped **values**: our logger

```go
func (s *Service) Create(ctx context.Context, ...) (Greeting, error)
```

Contexts form a tree. When a client disconnects, `net/http` cancels
`r.Context()`; everything derived from it sees `Done()` close. Database
drivers and `http.Client` honour it automatically. In your own loops:

```go
select {
case <-ctx.Done():
    return ctx.Err()            // context.Canceled or context.DeadlineExceeded
case result := <-work:
    return result
}
```

Create derived contexts with `context.WithTimeout`, `WithCancel`,
`WithValue`. Always `defer cancel()`.

## 5. Waiting for a group: `sync.WaitGroup` and `errgroup`

```go
var wg sync.WaitGroup
for i := range 50 {
    wg.Add(1)
    go func() {
        defer wg.Done()
        work(i)
    }()
}
wg.Wait()
```

When the goroutines can fail and you want the first error plus cancellation
of the rest (like `asyncio.TaskGroup`), use
[`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup):

```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return fetchA(ctx) })
g.Go(func() error { return fetchB(ctx) })
if err := g.Wait(); err != nil { ... }
```

## Common mistakes

- Starting a goroutine in a handler that uses `r.Context()` after the
  handler returns: the context is already cancelled.
- Forgetting that an unbuffered send with no receiver blocks forever (leak).
  That's why `Run` uses `make(chan error, 1)`.
- Closing a channel from the receiver side, or twice (panic).
- Copying a struct containing a mutex (`go vet` warns).

## Exercise

Add `GET /api/v1/hello/batch?names=a,b,c` that renders each greeting in its own
goroutine, collects results through a channel, and returns them in the
original order. Then rewrite it with `errgroup` and a results slice indexed by
position. Which version is clearer?

Next: [11 · Testing](11-testing.md)
