// Package middleware contains the project's custom HTTP middleware.
//
// In Go, middleware is just a function with this shape:
//
//	func(next http.Handler) http.Handler
//
// It receives the next handler in the chain and returns a new handler that
// does something before and/or after calling next. Compare with a Python
// decorator, or a Starlette/Django middleware's `__call__`:
//
//	def middleware(get_response):
//	    def handler(request):
//	        ...before...
//	        response = get_response(request)
//	        ...after...
//	        return response
//	    return handler
//
// See docs/06-http-chi-and-middleware.md.
package middleware

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// RequestLogger does two jobs:
//
//  1. Builds a child logger carrying request_id/method/path and stores it in
//     the request's context, so any code handling this request can call
//     logger.FromContext(ctx) and get correlated log lines for free.
//  2. Writes one "access log" line when the request finishes, with status,
//     bytes written and duration.
//
// It must run *after* chi's RequestID middleware so the ID is available.
func RequestLogger(base zerolog.Logger) func(http.Handler) http.Handler {
	// The outer function runs once, at router setup. The returned closure
	// runs for every request. Closures capture `base`, just like Python.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := chimw.GetReqID(r.Context())

			// Echo the ID back so clients can quote it in bug reports.
			w.Header().Set("X-Request-Id", reqID)

			l := base.With().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			// Contexts are immutable: WithContext returns a *new* request
			// that carries a new context containing the logger.
			r = r.WithContext(l.WithContext(r.Context()))

			// http.ResponseWriter does not expose the status code after it
			// is written, so we wrap it. chi provides a ready-made wrapper.
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			// defer runs when this function returns, even if next panics
			// and the panic is later recovered further up. Like `finally`.
			defer func() {
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK // handler wrote nothing
				}

				var evt *zerolog.Event
				switch {
				case status >= 500:
					evt = l.Error()
				case status >= 400:
					evt = l.Warn()
				default:
					evt = l.Info()
				}

				evt.Int("status", status).
					Int("bytes", ww.BytesWritten()).
					Dur("duration", time.Since(start)).
					Str("remote_ip", r.RemoteAddr).
					Str("user_agent", r.UserAgent()).
					Msg("request completed")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
