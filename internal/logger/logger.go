// Package logger builds the application's zerolog.Logger.
//
// Python analogy: this is your `logging.config.dictConfig(...)` call, done
// once at startup. Unlike Python's global `logging.getLogger(__name__)`, we
// create one logger and *pass it* to whatever needs it (dependency
// injection). Per-request loggers travel inside context.Context.
//
// See docs/07-logging-with-zerolog.md.
package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New returns a configured logger.
//
//   - production:  one JSON object per line on stdout (what log shippers like
//     Loki, Datadog, CloudWatch expect)
//   - development: coloured, human-friendly console output
func New(level string, development bool, version string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano

	var out io.Writer = os.Stdout
	if development {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05.000"}
	}

	return zerolog.New(out).
		Level(lvl).
		With().
		Timestamp().
		Str("service", "go-primer-api").
		Str("version", version).
		Logger()
}

// FromContext returns the request-scoped logger stored by the request-logger
// middleware, or a disabled logger if none is present (so callers never need a
// nil check). zerolog.Ctx does the lookup for us.
func FromContext(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}
