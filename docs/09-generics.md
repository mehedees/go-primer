# 09 · Generics (type parameters)

> **Goal:** read and write generic functions, and know when *not* to.

Go got generics in 1.18. They look like Python's `TypeVar`, but the types are
checked and specialised at compile time.

## The one generic function in this project

[`httpx.go`](../internal/httpx/httpx.go):

```go
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
    var v T
    ...
    err := dec.Decode(&v)
    return v, err
}

// call site, handler.go
req, err := httpx.DecodeJSON[createRequest](w, r)
```

Python equivalent:

```python
T = TypeVar("T")
def decode_json(cls: type[T], request) -> T: ...
req = decode_json(CreateRequest, request)
```

Without generics you'd write `var req createRequest; err := decode(r, &req)`,
which works fine. Generics just remove a line and a pointer at each call site.

## Syntax

```go
func Map[T, U any](xs []T, f func(T) U) []U {
    out := make([]U, 0, len(xs))
    for _, x := range xs {
        out = append(out, f(x))
    }
    return out
}

names := Map(greetings, func(g Greeting) string { return g.Name })  // T, U inferred
```

### Constraints

The thing in brackets after the type parameter is a **constraint**, an
interface describing what `T` can do:

```go
any                    // anything
comparable             // supports == and != (usable as map keys)
cmp.Ordered            // supports < > (ints, floats, strings)
interface{ ~int | ~string }  // a union: types whose underlying type is int or string
```

```go
func Max[T cmp.Ordered](a, b T) T { if a > b { return a }; return b }
```

`~string` means "string or any type defined on string", so our `Language`
type satisfies `~string`.

### Generic types

```go
type Page[T any] struct {
    Items []T `json:"items"`
    Next  string `json:"next,omitempty"`
}
var p Page[Greeting]
```

## When to use generics

The Go community's rule: **write the concrete code first.** Reach for
generics when you've written the same function for two or more types and the
only difference is the type: containers, `slices`/`maps`-style helpers,
decoders, result wrappers. Don't make a generic `Repository[T]` on day one;
interfaces already give you polymorphism over *behaviour*. Generics are for
polymorphism over *types*.

Most of what you'd use generics for is already in the stdlib:
`slices.Sort`, `slices.Index`, `maps.Keys`, `min`, `max`, `sync.OnceValue`.

## Exercise

Replace the `listResponse` struct in `handler.go` with a generic
`httpx.List[T any]` type with `Items []T` and `Count int`, and a constructor
`httpx.NewList[T any](items []T) List[T]`.

Next: [10 · Concurrency](10-concurrency.md)
