# 03 · Types, structs, pointers, slices, maps and interfaces

> **Goal:** understand how Go models data and behaviour without classes or
> inheritance.

## Structs ≈ dataclasses

```python
@dataclass
class Greeting:
    id: str
    name: str
    created_at: datetime
```

```go
type Greeting struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

g := Greeting{ID: "1", Name: "Ada"}  // unspecified fields get zero values
g.Name = "Grace"
```

Source: [`greeting.go`](../internal/greeting/greeting.go). The backtick
strings are **struct tags**: metadata for `encoding/json` (like pydantic's
`Field(alias=...)`).

## Methods

A method is a function with a *receiver*. The receiver is `self`, written
explicitly and named by convention after the type (`s` for `Service`):

```go
func (s *Service) Create(ctx context.Context, name string, lang Language) (Greeting, error)
```

You can attach methods to **any type you define**, not just structs:

```go
type Language string
func (l Language) Valid() bool { ... }
```

## No inheritance. Composition instead.

There are no classes, no `super()`, no MRO. You build bigger types by holding
smaller ones as fields (`Server` holds an `*http.Server`, a logger, a health
handler). Go also has *embedding* (`struct { sync.Mutex; ... }`) which
promotes the inner type's methods, but it is composition, not subtyping.

## Pointers (they're not scary)

Python: every variable is a reference to a heap object; mutation through any
reference is visible to all. Go: **values are copied** on assignment and when
passed to functions, unless you pass a pointer.

```go
func rename(g Greeting)  { g.Name = "X" }  // modifies a COPY. Caller sees nothing.
func renameP(g *Greeting) { g.Name = "X" } // modifies the caller's value.

g := Greeting{Name: "Ada"}
renameP(&g)   // & = "address of"
```

No pointer arithmetic. `p.Name` auto-dereferences (no `->`).

**Pointer or value receiver?** Rule of thumb used in this project:

- The method mutates the receiver, or the struct holds a mutex, or it is big
  → pointer receiver `(r *MemoryRepository)`.
- Small, immutable-ish value → value receiver `(l Language)`, `(c Config)`.
- Be consistent within a type.

Copying a struct that contains a `sync.Mutex` is a bug; `go vet` catches it.

## Slices ≈ lists (with a twist)

```go
items := []Greeting{}               // empty, non-nil
out := make([]Greeting, 0, 10)      // len 0, capacity 10 (pre-allocated)
out = append(out, g)                // ALWAYS reassign the result of append
first := out[0]
sub := out[1:3]                     // shares memory with out! not a copy
```

A slice is a small header (pointer, length, capacity) over an array.
`append` may reallocate, hence `out = append(out, ...)`. Sub-slices share the
backing array, so mutating `sub[0]` changes `out[1]`. When in doubt, copy:
`slices.Clone(out)`.

The `slices` and `maps` stdlib packages give you the familiar helpers:
`slices.Contains`, `slices.Sort`, `slices.Reverse` (used in
[`repository.go`](../internal/greeting/repository.go)), `maps.Keys`.

## Maps ≈ dicts

```go
m := map[string]int{"a": 1}
m["b"] = 2
v := m["missing"]        // 0, no KeyError!
v, ok := m["missing"]    // ok == false: the "comma ok" idiom
delete(m, "a")
```

Two traps: **iteration order is randomised on purpose**, and **writing to a
nil map panics** (that's the deliberate bug behind `/api/v1/panic` in
development). Maps are not safe for concurrent writes; see
[10 · Concurrency](10-concurrency.md).

## Interfaces = `typing.Protocol`, checked by the compiler

```go
type Repository interface {
    Create(ctx context.Context, g Greeting) error
    Get(ctx context.Context, id string) (Greeting, error)
    List(ctx context.Context) ([]Greeting, error)
}
```

`*MemoryRepository` never says `implements Repository`. It just *has the
methods*, so it qualifies. This is structural typing, exactly like a
`Protocol`, except it's enforced at compile time, everywhere.

Idioms you'll see in this project:

- **Define interfaces where they are consumed**, and keep them small. The
  service needs a repository, so the service's package defines it.
  (`io.Reader` has one method. That's the gold standard.)
- **Accept interfaces, return concrete types.** `NewService(repo Repository)`
  takes an interface; `NewMemoryRepository()` returns `*MemoryRepository`.
- **Compile-time assertion:** `var _ Repository = (*MemoryRepository)(nil)`
  fails the build if a method goes missing.

`any` (alias of `interface{}`) is the empty interface: every type satisfies it.
Use it sparingly, as you'd use `typing.Any`.

### Nil interface gotcha

An interface value is (type, value). An interface holding a nil `*T` is **not**
equal to `nil`. Never return a typed nil pointer as an `error`; return a
literal `nil`.

## Exercise

Write a `LoggingRepository` struct that wraps another `Repository` (a field
`next Repository`) and logs each call, then satisfies `Repository` itself
(decorator pattern). Wire it in `cmd/api/main.go`:
`repo := NewLoggingRepository(greeting.NewMemoryRepository(), log)`.
Notice that `Service` doesn't change at all.

Next: [04 · Errors and panics](04-errors.md)
