# 13 · Gotchas for Pythonistas, and where to go next

## Things that will bite you in week one

1. **Forgetting `return` after writing an error response.** The handler keeps
   running and writes a second response (`superfluous WriteHeader` in logs).
2. **`append` without reassigning:** `append(xs, x)` alone does nothing useful.
3. **Writing to a nil map panics.** Use `make(map[K]V)` or a literal.
4. **Map iteration order is random.** Sort keys if order matters.
5. **`len(string)` is bytes.** Use `utf8.RuneCountInString` for characters.
6. **Loop variables and closures.** Since Go 1.22 each iteration has its own
   variable (old blog posts warn about this; it's fixed).
7. **Value vs pointer receivers.** A method with a value receiver that
   "modifies" the struct modifies a copy.
8. **Nil interface ≠ interface holding a nil pointer.** Return literal `nil`
   errors.
9. **Ignoring errors with `_`.** Legal, sometimes right, usually a bug.
   `errcheck` (in golangci-lint) catches it.
10. **Not closing `resp.Body`** from `http.Client` leaks connections.
11. **Shadowing with `:=`** inside an `if`/`for` block creates a *new*
    variable; the outer one stays unchanged.
12. **zerolog event without `.Msg()`** is silently dropped.
13. **Integer division and overflow**: `7/2 == 3`, and `int8(127)+1 == -128`.

## Idioms to internalise

- Errors are values; wrap with context, handle once.
- Accept interfaces, return structs. Keep interfaces small.
- Make the zero value useful.
- `ctx` first, `err` last.
- A little copying is better than a little dependency.
- Clear is better than clever. `gofmt` settles style debates.

## Suggested path through this repo

1. Read [`cmd/api/main.go`](../cmd/api/main.go) top to bottom.
2. Follow one request: `server.go` (router) → `middleware/logging.go` →
   `greeting/handler.go` → `service.go` → `repository.go`.
3. Do the exercises at the end of each lesson, running `make test` after each.
4. Then the capstone exercises below.

## Capstone exercises

Work on a branch per exercise and keep `make check` green.

1. **Postgres repository.** Add `internal/greeting/postgres.go` implementing
   `Repository` with `jackc/pgx/v5`, a `DATABASE_URL` setting, and a
   `docker-compose.yml`. Only `main.go` should change to switch storage.
2. **Pagination.** `GET /api/v1/greetings?limit=20&cursor=...` using a generic
   `Page[T]` type.
3. **Metrics middleware.** Count requests by route pattern and status with
   Prometheus, exposed at `/metrics`. (Hint: `chi.RouteContext(r.Context()).RoutePattern()`.)
4. **Rate limiting** per client IP with `golang.org/x/time/rate`, returning 429.
5. **A second binary**: `cmd/greetctl/` CLI that calls the API with
   `net/http`'s client and prints results (uses the same module).
6. **CI**: a GitHub Actions workflow running `gofmt -l`, `go vet`,
   `golangci-lint` and `go test -race`.

## Resources

- [A Tour of Go](https://go.dev/tour/): interactive, 2 to 3 hours. Do it.
- [Effective Go](https://go.dev/doc/effective_go): the idioms document.
- [Go by Example](https://gobyexample.com/): quick snippets per topic.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): what
  reviewers will flag.
- [Go Proverbs](https://go-proverbs.github.io/).
- Book: *Let's Go* and *Let's Go Further* by Alex Edwards (web services).
- Book: *Learning Go* (2nd ed.) by Jon Bodner (language, for experienced devs).
- [pkg.go.dev](https://pkg.go.dev): documentation for every public module.

[← Back to the README](../README.md)
