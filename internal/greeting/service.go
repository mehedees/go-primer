package greeting

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const maxNameLength = 64

// Service holds the business rules. It knows nothing about HTTP, JSON or
// which database is used, which makes it trivial to unit test.
type Service struct {
	repo  Repository       // lowercase field: private to this package
	now   func() time.Time // injectable clock, so tests can freeze time
	newID func() string
}

// NewService wires a Service to its dependencies. This is "dependency
// injection" Go-style: plain constructor arguments, no framework.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, newID: randomID}
}

// Hello builds a greeting without storing it.
//
// Every function that may do I/O takes ctx context.Context as its first
// argument. The context carries the request deadline, cancellation and
// request-scoped values (our logger). See docs/10-concurrency.md.
func (s *Service) Hello(ctx context.Context, name string, lang Language) (string, error) {
	name, lang, err := s.validate(name, lang)
	if err != nil {
		return "", err
	}
	return lang.Render(name), nil
}

// Create validates the input, renders the message and persists it.
func (s *Service) Create(ctx context.Context, name string, lang Language) (Greeting, error) {
	name, lang, err := s.validate(name, lang)
	if err != nil {
		return Greeting{}, err
	}

	g := Greeting{
		ID:        s.newID(),
		Name:      name,
		Language:  lang,
		Message:   lang.Render(name),
		CreatedAt: s.now().UTC(),
	}

	if err := s.repo.Create(ctx, g); err != nil {
		return Greeting{}, fmt.Errorf("create greeting: %w", err)
	}
	return g, nil
}

// Get returns one greeting by ID. A missing ID yields an error wrapping
// ErrNotFound; we simply pass the repository error up.
func (s *Service) Get(ctx context.Context, id string) (Greeting, error) {
	return s.repo.Get(ctx, id)
}

// List returns all greetings, newest first.
func (s *Service) List(ctx context.Context) ([]Greeting, error) {
	return s.repo.List(ctx)
}

// validate normalises and checks input. Multiple return values replace the
// tuple-returning functions you'd write in Python.
func (s *Service) validate(name string, lang Language) (string, Language, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", &ValidationError{Field: "name", Reason: "is required"}
	}
	// len(name) counts BYTES; utf8.RuneCountInString counts characters.
	// Python's len(str) counts code points, so this is the equivalent.
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", "", &ValidationError{Field: "name", Reason: fmt.Sprintf("must be at most %d characters", maxNameLength)}
	}

	if lang == "" {
		lang = English
	}
	if !lang.Valid() {
		return "", "", &ValidationError{Field: "language", Reason: fmt.Sprintf("unsupported language %q", lang)}
	}
	return name, lang, nil
}

// randomID returns a 16-hex-character random identifier.
func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error on supported platforms
	return hex.EncodeToString(b)
}
