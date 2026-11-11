# syntax=docker/dockerfile:1
#
# Multi-stage build. Stage 1 has the full Go toolchain (~800 MB); stage 2
# contains only the compiled binary on a minimal base (~10 MB total).
# Compare with a Python image, which must ship the interpreter plus every
# installed package.

# ---- build stage ----------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src

# Copy dependency manifests first so this layer is cached until they change
# (same trick as copying requirements.txt before the source).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
# CGO_ENABLED=0 -> pure Go, statically linked, runs on any Linux.
# -trimpath     -> strip local paths from the binary.
# -s -w         -> strip debug symbols (smaller binary).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/api ./cmd/api

# ---- runtime stage --------------------------------------------------------
# distroless/static: no shell, no package manager, just CA certs and tzdata.
# The :nonroot tag runs as UID 65532 instead of root.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api

ENV APP_ENV=production \
    HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/api"]
