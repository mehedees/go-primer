// Package config loads application settings from environment variables.
//
// Python analogy: this plays the role of a pydantic `BaseSettings` class.
// Go has no reflection-heavy magic by default, so we parse each variable
// explicitly. It is more typing, but there is zero hidden behaviour and the
// compiler checks every field.
//
// See docs/08-config-and-graceful-shutdown.md.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config is the full set of runtime settings for the API.
//
// Exported (capitalised) fields are visible to other packages. There are no
// `public`/`private` keywords in Go: the first letter decides.
type Config struct {
	Env      string // "development" or "production"
	HTTP     HTTPConfig
	LogLevel string // trace, debug, info, warn, error
}

// HTTPConfig groups server settings. Nested structs are how Go models
// "sub-sections" of settings.
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration
}

// IsDevelopment reports whether we are running locally.
// This is a *method*: a function with a receiver (`c Config`), roughly a
// Python instance method where `c` is `self`.
func (c Config) IsDevelopment() bool {
	return c.Env == "development"
}

// Load reads configuration from the environment, applying defaults.
//
// Go functions commonly return (value, error). There are no exceptions:
// the caller must look at the error. errors.Join lets us report every bad
// variable at once instead of failing on the first.
func Load() (Config, error) {
	var errs []error

	// A tiny closure that records parse errors so each line below stays short.
	dur := func(key string, def time.Duration) time.Duration {
		d, err := getDuration(key, def)
		if err != nil {
			errs = append(errs, err)
		}
		return d
	}

	cfg := Config{
		Env:      getString("APP_ENV", "development"),
		LogLevel: getString("LOG_LEVEL", "info"),
		HTTP: HTTPConfig{
			Addr:            getString("HTTP_ADDR", ":8080"),
			ReadTimeout:     dur("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    dur("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     dur("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: dur("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			RequestTimeout:  dur("HTTP_REQUEST_TIMEOUT", 8*time.Second),
		},
	}

	switch cfg.Env {
	case "development", "production":
		// ok. Note: Go's switch does not fall through by default.
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be development or production, got %q", cfg.Env))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

func getString(key, def string) string {
	// os.LookupEnv returns two values: the value and whether it was set.
	// This "comma ok" idiom is everywhere in Go.
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v) // accepts "5s", "250ms", "1m30s"
	if err != nil {
		return def, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}
