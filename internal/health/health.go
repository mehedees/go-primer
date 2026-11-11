// Package health exposes liveness and readiness probes.
//
// Kubernetes (and most load balancers) call these:
//   - /healthz  "is the process alive?"   -> restart the pod if not
//   - /readyz   "can it take traffic?"    -> stop routing to it if not
//
// During graceful shutdown we flip readiness to false first, so the load
// balancer drains traffic before the server stops.
package health

import (
	"net/http"
	"sync/atomic"

	"github.com/mehedees/go-primer/internal/httpx"
)

// Handler tracks readiness with an atomic.Bool, which is safe to read and
// write from many goroutines without a mutex.
type Handler struct {
	ready   atomic.Bool
	version string
}

func NewHandler(version string) *Handler {
	h := &Handler{version: version}
	h.ready.Store(true)
	return h
}

// SetReady toggles the readiness probe.
func (h *Handler) SetReady(ready bool) { h.ready.Store(ready) }

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, r, http.StatusOK, map[string]string{"status": "ok", "version": h.version})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		httpx.JSON(w, r, http.StatusServiceUnavailable, map[string]string{"status": "shutting down"})
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]string{"status": "ready"})
}
