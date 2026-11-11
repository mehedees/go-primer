// Command api is the entry point for the greeting HTTP service.
//
// Everything under cmd/<name>/ builds into a binary called <name>. main()
// should stay thin: load config, build dependencies, run, exit. All real
// logic lives in internal/ where it can be tested.
//
// Python analogy: this is `if __name__ == "__main__": main()`, except that
// `go build` turns it into a single static binary with no interpreter or
// virtualenv needed on the server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mehedees/go-primer/internal/config"
	"github.com/mehedees/go-primer/internal/greeting"
	"github.com/mehedees/go-primer/internal/logger"
	"github.com/mehedees/go-primer/internal/server"
)

// version is overwritten at build time:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/api
//
// The Makefile does this using `git describe`.
var version = "dev"

func main() {
	// The "run() error" pattern: main only translates an error into an exit
	// code. Deferred cleanups inside run() still execute, which would not
	// happen if we called os.Exit deep inside the program.
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel, cfg.IsDevelopment(), version)
	log.Info().Str("env", cfg.Env).Msg("starting")

	// ctx is cancelled when the process receives Ctrl+C (SIGINT) or SIGTERM
	// (what Docker and Kubernetes send on stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Dependency wiring ("composition root") -------------------------
	// Build objects bottom-up and hand each its dependencies. To swap the
	// in-memory store for Postgres later, only this line changes.
	repo := greeting.NewMemoryRepository()
	svc := greeting.NewService(repo)
	greetingHandler := greeting.NewHandler(svc)

	srv := server.New(server.Deps{
		Config:   cfg,
		Logger:   log,
		Version:  version,
		Greeting: greetingHandler,
	})

	if err := srv.Run(ctx); err != nil {
		return err
	}
	log.Info().Msg("bye")
	return nil
}
