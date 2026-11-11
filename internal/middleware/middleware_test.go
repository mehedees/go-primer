package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// httptest gives us an in-memory request and a ResponseRecorder, so we can
// test handlers and middleware without opening a socket. It is Go's version
// of Django's test Client / FastAPI's TestClient.

func TestRecovererReturns500(t *testing.T) {
	panicky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Recoverer(panicky).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRequestLoggerLogsAndSetsHeader(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	handler := chimw.RequestID(RequestLogger(log)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}),
	))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tea", nil))

	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header not set")
	}

	// Decode the JSON log line into a map, like json.loads.
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log is not JSON: %v (%s)", err, buf.String())
	}
	if got := entry["status"]; got != float64(http.StatusTeapot) {
		t.Errorf("logged status = %v, want 418", got)
	}
	if entry["path"] != "/tea" {
		t.Errorf("logged path = %v, want /tea", entry["path"])
	}
}
