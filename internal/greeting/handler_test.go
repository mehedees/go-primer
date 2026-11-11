// This file uses the *external* test package `greeting_test`. It can only
// touch exported identifiers, exactly like a real caller would. Use it for
// black-box tests of the public API.
package greeting_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mehedees/go-primer/internal/greeting"
)

func newRouter() http.Handler {
	h := greeting.NewHandler(greeting.NewService(greeting.NewMemoryRepository()))
	r := chi.NewRouter()
	r.Get("/hello", h.Hello)
	r.Mount("/greetings", h.Routes())
	return r
}

func TestHelloEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello?name=Ada&lang=es", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "¡Hola, Ada!" {
		t.Errorf("message = %q", body.Message)
	}
}

func TestCreateThenFetch(t *testing.T) {
	router := newRouter()

	// 1. Create
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/greetings/", strings.NewReader(`{"name":"Linus","language":"fr"}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	var created greeting.Greeting
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	if want := "/api/v1/greetings/" + created.ID; rec.Header().Get("Location") != want {
		t.Errorf("Location = %q, want %q", rec.Header().Get("Location"), want)
	}

	// 2. Fetch it back by ID
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/greetings/"+created.ID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	var fetched greeting.Greeting
	if err := json.NewDecoder(rec.Body).Decode(&fetched); err != nil {
		t.Fatal(err)
	}
	if fetched.ID != created.ID || fetched.Message != "Bonjour, Linus !" {
		t.Errorf("fetched = %+v", fetched)
	}
}

func TestErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"unknown id", http.MethodGet, "/greetings/missing", "", http.StatusNotFound},
		{"invalid json", http.MethodPost, "/greetings/", `{"name":`, http.StatusBadRequest},
		{"unknown field", http.MethodPost, "/greetings/", `{"nmae":"x"}`, http.StatusBadRequest},
		{"empty body", http.MethodPost, "/greetings/", ``, http.StatusBadRequest},
		{"validation", http.MethodPost, "/greetings/", `{"name":""}`, http.StatusUnprocessableEntity},
		{"bad lang", http.MethodGet, "/hello?lang=xx", "", http.StatusUnprocessableEntity},
	}

	router := newRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body)
			}
		})
	}
}
