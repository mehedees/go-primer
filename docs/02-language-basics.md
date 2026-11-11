# 02 · Language basics for Pythonistas

> **Goal:** read any line of this project's code. Side-by-side Python ↔ Go.

## Variables and zero values

```python
name = "Ada"          # type inferred at runtime
count: int = 0
```

```go
name := "Ada"         // short declaration, type inferred at COMPILE time (inside functions only)
var count int         // declared without value -> zero value: 0
var ok bool           // false
var s string          // "" (never None!)
var p *Greeting       // nil
```

**Zero values** are a big deal. Every type has one, and nothing is ever
"uninitialised". Good Go code makes the zero value useful: a zero
`sync.Mutex` is an unlocked mutex, a zero `bytes.Buffer` is empty and ready.

`:=` declares; `=` assigns. `x := 1` twice in the same scope is an error.

## Basic types

| Python          | Go                                    | Notes |
|-----------------|---------------------------------------|-------|
| `int`           | `int`, `int64`, `uint8` (`byte`)...   | fixed size; overflow wraps silently |
| `float`         | `float64`                             | |
| `str`           | `string`                              | immutable **bytes**, UTF-8 by convention |
| one character   | `rune` (alias for `int32`)            | a Unicode code point |
| `bool`          | `bool`                                | |
| `None`          | `nil`                                 | only for pointers, slices, maps, channels, funcs, interfaces |
| `list`          | slice `[]T`                           | see [03](03-types-and-interfaces.md) |
| `dict`          | map `map[K]V`                         | |
| `bytes`         | `[]byte`                              | |

No implicit conversions, ever: `float64(n)`, `string(b)`, `Language(s)`.

**Strings are bytes.** `len("হ্যালো")` is 18, not 6. To count characters, use
`utf8.RuneCountInString`, exactly as
[`service.go`](../internal/greeting/service.go) does for name validation.

## Functions and multiple returns

```python
def divide(a: float, b: float) -> tuple[float, str | None]: ...
```

```go
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

q, err := divide(1, 0)
_, err = divide(1, 2)     // _ discards a value, like Python
```

No default arguments, no keyword arguments, no `*args` (well: variadic
`func f(xs ...int)` exists). Instead Go uses **structs of options**, see
`server.Deps` in [`server.go`](../internal/server/server.go).

Functions are values and closures work like Python's:

```go
dur := func(key string, def time.Duration) time.Duration { ... }  // config.go
```

## Control flow

```go
if v, ok := os.LookupEnv(key); ok && v != "" {   // statement; condition
    return v
}

for i := 0; i < 10; i++ {}    // C-style
for cond {}                   // while
for {}                        // while True
for i, item := range items {} // enumerate(items)
for key, val := range m {}    // m.items()  (random order!)
for i := range 10 {}          // range(10)  (Go 1.22+)

switch {                      // switch with no value = if/elif chain
case status >= 500:
case status >= 400:
default:
}
```

No `while` keyword, no ternary `a if c else b`, no list comprehensions. Write
the loop. It's fine.

## Visibility: capitalisation is the access modifier

```go
func NewService(...)   // Exported: usable from other packages
func randomID() string // unexported: package-private (like a leading _ but ENFORCED)
```

Applies to functions, types, struct fields, constants, methods. Note that
`encoding/json` can only see **exported** fields, which is why `Greeting`'s
fields are capitalised and use tags to get lowercase JSON keys.

## `defer`

```go
r.mu.Lock()
defer r.mu.Unlock()    // runs when the surrounding FUNCTION returns
```

It's `try/finally` and `with` rolled into one line. Deferred calls run LIFO,
and they run even during a panic, which is how
[`Recoverer`](../internal/middleware/recoverer.go) works.

## Formatting strings

```go
fmt.Sprintf("Hello, %s! You are %d", name, age)  // f-string equivalent
fmt.Printf("%v\n", x)    // default format, like str(x)
fmt.Printf("%+v\n", x)   // struct with field names
fmt.Printf("%#v\n", x)   // Go-syntax repr, like repr(x)
fmt.Printf("%q\n", s)    // quoted string
fmt.Errorf("get %q: %w", id, err)  // build an error, %w wraps another error
```

## Exercise

Open [`internal/config/config.go`](../internal/config/config.go) and identify:
a short declaration, a closure, the comma-ok idiom, a switch without
fallthrough, a variadic call (`errors.Join(errs...)`, the `...` is the `*`
of Python's `f(*args)`).

Next: [03 · Types, structs, methods and interfaces](03-types-and-interfaces.md)
