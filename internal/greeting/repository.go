package greeting

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// Repository is the storage abstraction the Service depends on.
//
// Go interfaces are satisfied *implicitly*: any type that has these methods
// is a Repository, without declaring it. This is structural typing, the same
// idea as Python's typing.Protocol.
//
// Idiom: "accept interfaces, return structs". The interface lives next to the
// code that *uses* it (the service), not next to the implementation.
type Repository interface {
	Create(ctx context.Context, g Greeting) error
	Get(ctx context.Context, id string) (Greeting, error)
	List(ctx context.Context) ([]Greeting, error)
}

// MemoryRepository stores greetings in a map, guarded by a mutex.
//
// HTTP handlers run concurrently, one goroutine per request, so two requests
// can touch this map at the same time. Go maps are NOT safe for concurrent
// writes (there is no GIL!). sync.RWMutex allows many readers or one writer.
// Run `make test` (which uses -race) to have the race detector check this.
type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]Greeting
	order []string // insertion order, because map iteration order is random
}

// Compile-time check that *MemoryRepository satisfies Repository.
// If a method is missing, the build fails right here with a clear message.
var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository is a constructor. Go has no __init__; by convention a
// function named NewXxx returns a ready-to-use value.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]Greeting)}
}

func (r *MemoryRepository) Create(ctx context.Context, g Greeting) error {
	r.mu.Lock()
	defer r.mu.Unlock() // released when the function returns, like `with lock:`

	if _, exists := r.items[g.ID]; exists {
		return fmt.Errorf("greeting %q already exists", g.ID)
	}
	r.items[g.ID] = g
	r.order = append(r.order, g.ID)
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, id string) (Greeting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.items[id]
	if !ok {
		// %w wraps the sentinel so callers can still errors.Is(err, ErrNotFound)
		// while the message gains context. Like `raise NotFound(...) from e`.
		return Greeting{}, fmt.Errorf("get %q: %w", id, ErrNotFound)
	}
	return g, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Greeting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// make([]T, 0, n) creates an empty slice with capacity n pre-allocated.
	out := make([]Greeting, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.items[id])
	}
	// Newest first. slices (Go 1.21+) is the stdlib's list-helper package.
	slices.Reverse(out)
	return out, nil
}
