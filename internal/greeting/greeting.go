// Package greeting is a small "feature package" that demonstrates a layered
// design inside one package:
//
//	handler.go     HTTP layer     (parse request, call service, write JSON)
//	service.go     business logic (validation, rules, no HTTP knowledge)
//	repository.go  storage        (interface + in-memory implementation)
//	greeting.go    domain types and errors (this file)
//
// Go projects usually group code *by feature* (greeting/, user/, order/)
// rather than *by layer* (handlers/, services/, models/) as many Django or
// FastAPI projects do. See docs/05-project-layout.md for why.
package greeting

import (
	"errors"
	"fmt"
	"time"
)

// Greeting is the domain model. Think of it as a @dataclass.
type Greeting struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Language  Language  `json:"language"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Language is a *defined type* based on string. It behaves like a string
// but the compiler won't let you pass an arbitrary string where a Language is
// expected without an explicit conversion. Closest Python idea: a StrEnum.
type Language string

// Constants for the supported languages. Go has no `enum` keyword; a set of
// typed constants is the idiom.
const (
	English Language = "en"
	Spanish Language = "es"
	French  Language = "fr"
	Bangla  Language = "bn"
)

// templates maps each language to a format string. A map literal is like a
// dict literal. Package-level vars are created once at program start.
var templates = map[Language]string{
	English: "Hello, %s!",
	Spanish: "¡Hola, %s!",
	French:  "Bonjour, %s !",
	Bangla:  "হ্যালো, %s!",
}

// Valid reports whether l is a supported language.
func (l Language) Valid() bool {
	_, ok := templates[l] // "comma ok": ok is false if the key is missing
	return ok
}

// Render builds the greeting text for name in language l.
func (l Language) Render(name string) string {
	return fmt.Sprintf(templates[l], name)
}

// Sentinel errors: package-level error values callers compare against with
// errors.Is. Python analogy: custom exception classes you `except` on.
var (
	ErrNotFound = errors.New("greeting not found")
)

// ValidationError is a custom error *type* carrying extra data. Callers pull
// it out with errors.As (like `except ValidationError as e: e.field`).
//
// Any type with an `Error() string` method satisfies the built-in `error`
// interface. No "implements" keyword is needed.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
