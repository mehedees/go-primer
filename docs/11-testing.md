# 11 · Testing

> **Goal:** write the three kinds of tests this project uses, with only the
> standard library.

## Basics

- Test files end in `_test.go` and sit next to the code.
- A test is `func TestXxx(t *testing.T)`.
- `t.Errorf` records a failure and continues; `t.Fatalf` stops this test.
- No `assert` keyword. You write `if got != want { t.Errorf(...) }`. The
  convention for messages is `got X, want Y`.

```bash
make test                                   # go test -race -count=1 ./...
go test ./internal/greeting/                 # one package
go test -run TestHello/spanish -v ./...      # one sub-test, verbose
make cover                                   # coverage report in the browser
```

`-count=1` disables the test result cache (Go caches passing results of
unchanged packages).

| pytest                        | Go |
|-------------------------------|----|
| `@pytest.mark.parametrize`    | table-driven tests + `t.Run` |
| fixtures with `yield`         | helper funcs + `t.Cleanup` |
| `monkeypatch.setenv`          | `t.Setenv` |
| `tmp_path`                    | `t.TempDir()` |
| `TestClient(app)`             | `net/http/httptest` |
| `unittest.mock`               | hand-written fakes satisfying an interface |
| `pytest -k name`              | `go test -run Regex` |
| `pytest-xdist`                | built in (packages run in parallel, `t.Parallel()`) |

Assertion libraries exist (`github.com/stretchr/testify`) and are popular.
This project sticks to the stdlib so you learn the base idioms first.

## 1. Unit tests: table-driven

[`service_test.go`](../internal/greeting/service_test.go):

```go
tests := []struct {
    name      string
    inName    string
    inLang    Language
    want      string
    wantField string
}{
    {name: "english default", inName: "Ada", want: "Hello, Ada!"},
    {name: "empty name", inName: "   ", wantField: "name"},
    ...
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

Adding a case is one line. This is *the* Go testing idiom; you'll see it in
the standard library itself.

### Controlling time and randomness

`Service` has unexported `now func() time.Time` and `newID func() string`
fields. Production uses `time.Now` and a crypto-random ID; tests in the same
package swap in fixed ones. Function-typed fields are the lightest form of
dependency injection.

### Internal vs external test packages

- `package greeting` (service_test.go): can touch unexported things.
- `package greeting_test` (handler_test.go): only the exported API, like a
  real consumer. Good for black-box tests.

## 2. HTTP handler tests: `httptest.NewRecorder`

[`handler_test.go`](../internal/greeting/handler_test.go):

```go
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/hello?name=Ada&lang=es", nil)
router.ServeHTTP(rec, req)          // call the handler directly, no network
if rec.Code != http.StatusOK { ... }
```

[`middleware_test.go`](../internal/middleware/middleware_test.go) uses the
same approach and also captures log output into a `bytes.Buffer` to assert on
the JSON log line.

## 3. Integration tests: `httptest.NewServer`

[`server_test.go`](../internal/server/server_test.go) starts the **real**
router, with every middleware, on a random port:

```go
ts := httptest.NewServer(NewRouter(deps, health.NewHandler("test")))
t.Cleanup(ts.Close)
resp, err := ts.Client().Do(req)
defer resp.Body.Close()
```

This catches wiring bugs (wrong middleware order, missing routes, content-type
checks) that unit tests can't.

## Fakes instead of mocks

Because `Service` depends on the `Repository` interface, a test can pass any
type with those three methods:

```go
// Embed the interface: failingRepo gets Get and List from the wrapped repo...
type failingRepo struct{ greeting.Repository }

// ...and overrides just Create.
func (failingRepo) Create(context.Context, greeting.Greeting) error {
    return errors.New("disk full")
}

svc := greeting.NewService(failingRepo{greeting.NewMemoryRepository()})
```

No mocking library needed. This is the payoff of small interfaces.

(Why embed the *interface* and not `greeting.MemoryRepository` itself? Its
methods have pointer receivers, so a plain `MemoryRepository` value doesn't
have them in its method set. Embedding `*greeting.MemoryRepository` would also
work.)

## Other built-ins worth knowing

- `t.Parallel()` runs sub-tests concurrently.
- `t.Helper()` makes failures point at the caller's line.
- Benchmarks: `func BenchmarkXxx(b *testing.B)`, run with `go test -bench .`.
- Fuzzing: `func FuzzXxx(f *testing.F)`, run with `go test -fuzz FuzzXxx`.
- Example tests: `func ExampleXxx()` with an `// Output:` comment are run as
  tests *and* shown in `go doc`.

## Exercises

1. Write a test proving that `POST /api/v1/greetings` returns 500 (not a crash)
   when the repository fails, using the `failingRepo` fake above.
2. Write `FuzzHello` that feeds random strings to `Service.Hello` and asserts
   it never panics and that a valid result always contains the trimmed name.
3. Add a benchmark for `MemoryRepository.List` with 10 000 items.

Next: [12 · Building and deploying](12-building-and-deploying.md)
