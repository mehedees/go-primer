package greeting

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestService returns a Service with a frozen clock and predictable IDs.
// Because the fields are unexported, only tests in this same package can set
// them. That is why these tests use `package greeting`, not `greeting_test`.
func newTestService() *Service {
	svc := NewService(NewMemoryRepository())
	svc.now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	n := 0
	svc.newID = func() string { n++; return "id-" + string(rune('0'+n)) }
	return svc
}

// TestHello is a *table-driven test*, the most common Go testing pattern.
// It is the equivalent of @pytest.mark.parametrize.
func TestHello(t *testing.T) {
	tests := []struct {
		name      string // sub-test name
		inName    string
		inLang    Language
		want      string
		wantField string // non-empty => expect a ValidationError on this field
	}{
		{name: "english default", inName: "Ada", want: "Hello, Ada!"},
		{name: "spanish", inName: "Ada", inLang: Spanish, want: "¡Hola, Ada!"},
		{name: "trims spaces", inName: "  Ada  ", want: "Hello, Ada!"},
		{name: "empty name", inName: "   ", wantField: "name"},
		{name: "too long", inName: strings.Repeat("a", 65), wantField: "name"},
		{name: "bad language", inName: "Ada", inLang: "xx", wantField: "language"},
	}

	svc := newTestService()
	for _, tt := range tests {
		// t.Run creates a named sub-test: `go test -run 'TestHello/spanish'`
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Hello(context.Background(), tt.inName, tt.inLang)

			if tt.wantField != "" {
				var vErr *ValidationError
				if !errors.As(err, &vErr) {
					t.Fatalf("want ValidationError, got %v", err)
				}
				if vErr.Field != tt.wantField {
					t.Errorf("Field = %q, want %q", vErr.Field, tt.wantField)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Hello() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	created, err := svc.Create(ctx, "Grace", French)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Message != "Bonjour, Grace !" {
		t.Errorf("Message = %q", created.Message)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Structs made only of comparable fields can be compared with ==.
	if got != created {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestGetNotFound(t *testing.T) {
	_, err := newTestService().Get(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// TestConcurrentCreates fires many goroutines at the repository at once.
// Run with -race (make test does) to prove the mutex is doing its job:
// delete the Lock/Unlock lines in repository.go and watch it fail.
func TestConcurrentCreates(t *testing.T) {
	svc := NewService(NewMemoryRepository()) // real random IDs
	ctx := context.Background()

	const workers = 50
	var wg sync.WaitGroup // like a counter you can wait on

	for i := range workers { // range over an int (Go 1.22+): 0..workers-1
		wg.Add(1)
		go func() { // `go` starts a goroutine: a very cheap concurrent task
			defer wg.Done()
			if _, err := svc.Create(ctx, "user", English); err != nil {
				t.Errorf("worker %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	items, _ := svc.List(ctx)
	if len(items) != workers {
		t.Fatalf("got %d items, want %d", len(items), workers)
	}
}
