# Makefile: the project's task runner (think `invoke`, `nox` or the
# "scripts" section of pyproject.toml). Run `make help` to list targets.
#
# Note: recipe lines MUST be indented with a TAB, not spaces.

BINARY   := api
PKG      := github.com/mehedees/go-primer
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
IMAGE    ?= go-primer-api

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally (loads .env if present)
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
		go run -ldflags "-X main.version=$(VERSION)" ./cmd/api

.PHONY: build
build: ## Build a static binary into ./bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/api

.PHONY: test
test: ## Run all tests with the race detector
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out

.PHONY: fmt
fmt: ## Format code (gofmt is non-negotiable in Go)
	gofmt -s -w .

.PHONY: vet
vet: ## Static analysis bundled with Go
	go vet ./...

.PHONY: lint
lint: vet ## Run golangci-lint if installed, otherwise just go vet
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; \
	else echo "golangci-lint not installed: brew install golangci-lint"; fi

.PHONY: tidy
tidy: ## Sync go.mod/go.sum with imports (like pip-compile)
	go mod tidy

.PHONY: check
check: fmt tidy lint test ## Everything you'd run before pushing

.PHONY: docker-build
docker-build: ## Build the container image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

.PHONY: docker-run
docker-run: ## Run the container on :8080 with production settings
	docker run --rm -p 8080:8080 -e APP_ENV=production $(IMAGE):latest

.PHONY: clean
clean: ## Remove build artefacts
	rm -rf bin coverage.out
