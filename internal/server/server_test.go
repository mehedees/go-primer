package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mehedees/go-primer/internal/config"
	"github.com/mehedees/go-primer/internal/greeting"
	"github.com/mehedees/go-primer/internal/health"
)

// newTestServer spins up the *whole* router (all middleware included) on a
// random local port using httptest.NewServer. This is an integration test:
// real HTTP over a real socket, but no Docker and no config files.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper() // failures are reported at the caller's line, not here

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		Config:   cfg,
		Logger:   zerolog.Nop(), // silence logs in tests
		Version:  "test",
		Greeting: greeting.NewHandler(greeting.NewService(greeting.NewMemoryRepository())),
	}
	ts := httptest.NewServer(NewRouter(deps, health.NewHandler("test")))
	t.Cleanup(ts.Close) // like a pytest fixture's teardown after `yield`
	return ts
}

func TestRoutes(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		ctype    string
		want     int
		contains string
	}{
		{"liveness", "GET", "/healthz", "", "", 200, `"status":"ok"`},
		{"readiness", "GET", "/readyz", "", "", 200, `"ready"`},
		{"hello", "GET", "/api/v1/hello?name=Go", "", "", 200, "Hello, Go!"},
		{"create", "POST", "/api/v1/greetings", `{"name":"Rob"}`, "application/json", 201, `"Hello, Rob!"`},
		{"wrong content type", "POST", "/api/v1/greetings", `name=Rob`, "text/plain", 415, ""},
		{"unknown route", "GET", "/nope", "", "", 404, "route not found"},
		{"panic is recovered", "GET", "/api/v1/panic", "", "", 500, "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			if tt.ctype != "" {
				req.Header.Set("Content-Type", tt.ctype)
			}

			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close() // ALWAYS close response bodies

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, tt.want, body)
			}
			if !strings.Contains(string(body), tt.contains) {
				t.Errorf("body %q does not contain %q", body, tt.contains)
			}
			if resp.Header.Get("X-Request-Id") == "" {
				t.Error("missing X-Request-Id header")
			}
		})
	}
}
