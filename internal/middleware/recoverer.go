package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/mehedees/go-primer/internal/httpx"
	"github.com/mehedees/go-primer/internal/logger"
)

// Recoverer turns a panic inside a handler into a 500 response and a
// structured log line, instead of killing the connection.
//
// Go has panics, but they are NOT the error mechanism (errors are values
// you return). A panic means "a bug happened" (nil pointer, index out of
// range). recover() stops the panic, but only when called inside a deferred
// function. Python analogy: a top-level `except Exception` in your WSGI app.
//
// chi ships its own middleware.Recoverer; we write ours so the panic is logged
// through zerolog with the request ID and returned in our JSON error format.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler is a deliberate signal to abort; re-panic.
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			logger.FromContext(r.Context()).Error().
				Interface("panic", rec).
				Bytes("stack", debug.Stack()).
				Msg("panic recovered")

			httpx.Error(w, r, http.StatusInternalServerError, "internal server error")
		}()

		next.ServeHTTP(w, r)
	})
}
