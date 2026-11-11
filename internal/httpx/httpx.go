// Package httpx holds small HTTP helpers shared by all handlers: writing JSON
// responses, a consistent error envelope, and decoding request bodies.
//
// In Python frameworks (FastAPI, DRF) serialization is done for you. In Go
// you call encoding/json yourself, so a few helpers keep handlers tidy.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mehedees/go-primer/internal/logger"
)

// maxBodyBytes caps request bodies at 1 MiB to protect the server.
const maxBodyBytes = 1 << 20

// ErrorResponse is the JSON shape of every error the API returns.
//
// The `json:"..."` strings are *struct tags*: metadata read by encoding/json
// via reflection, similar to pydantic's Field(alias=...).
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	// omitempty drops the field from the JSON when it is the zero value.
	RequestID string `json:"request_id,omitempty"`
}

// JSON writes v as a JSON response with the given status code.
//
// `v any` accepts a value of any type (`any` is an alias for `interface{}`),
// much like a `typing.Any` parameter.
func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are already sent, so all we can do is log it.
		logger.FromContext(r.Context()).Error().Err(err).Msg("encode response")
	}
}

// Error writes a JSON error envelope.
func Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	JSON(w, r, status, ErrorResponse{
		Error:     http.StatusText(status),
		Message:   message,
		RequestID: w.Header().Get("X-Request-Id"),
	})
}

// DecodeJSON reads the request body into a new value of type T.
//
// This is a *generic* function: `[T any]` is a type parameter, like
// `def decode(cls: type[T]) -> T` with a TypeVar in Python. Call it as
//
//	req, err := httpx.DecodeJSON[CreateGreetingRequest](r)
//
// See docs/09-generics.md.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T // the zero value of T: an empty struct, 0, "", nil, ...

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject {"nmae": "typo"} instead of ignoring it

	if err := dec.Decode(&v); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return v, fmt.Errorf("request body must not exceed %d bytes", maxErr.Limit)
		case errors.Is(err, io.EOF):
			return v, errors.New("request body must not be empty")
		default:
			return v, fmt.Errorf("invalid JSON: %w", err)
		}
	}
	// Ensure there is exactly one JSON value in the body.
	if dec.More() {
		return v, errors.New("request body must contain a single JSON object")
	}
	return v, nil
}
