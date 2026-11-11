package greeting

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mehedees/go-primer/internal/httpx"
	"github.com/mehedees/go-primer/internal/logger"
)

// Handler adapts HTTP requests to Service calls.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns a sub-router to be mounted by the server, e.g. at
// /api/v1/greetings. Similar to a FastAPI APIRouter or a Flask Blueprint.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get) // {id} is a URL parameter, read with chi.URLParam
	return r
}

// Request/response DTOs are kept separate from the domain model so the API
// contract can evolve independently (like pydantic request schemas).

type createRequest struct {
	Name     string   `json:"name"`
	Language Language `json:"language"`
}

type helloResponse struct {
	Message string `json:"message"`
}

type listResponse struct {
	Items []Greeting `json:"items"`
	Count int        `json:"count"`
}

// Hello handles GET /api/v1/hello?name=Ada&lang=fr
//
// An HTTP handler in Go is any func(w http.ResponseWriter, r *http.Request).
// You write the response into w; nothing is returned.
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")
	if name == "" {
		name = "World"
	}

	msg, err := h.svc.Hello(r.Context(), name, Language(q.Get("lang")))
	if err != nil {
		h.writeError(w, r, err)
		return // easy to forget! without it the handler keeps going
	}
	httpx.JSON(w, r, http.StatusOK, helloResponse{Message: msg})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createRequest](w, r)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	g, err := h.svc.Create(r.Context(), req.Name, req.Language)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	logger.FromContext(r.Context()).Info().
		Str("greeting_id", g.ID).
		Msg("greeting created")

	w.Header().Set("Location", "/api/v1/greetings/"+g.ID)
	httpx.JSON(w, r, http.StatusCreated, g)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	g, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, g)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, listResponse{Items: items, Count: len(items)})
}

// writeError maps domain errors to HTTP status codes in ONE place.
// This is like a FastAPI exception_handler.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var vErr *ValidationError
	switch {
	case errors.As(err, &vErr):
		httpx.Error(w, r, http.StatusUnprocessableEntity, vErr.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, r, http.StatusNotFound, "greeting not found")
	default:
		// Unexpected: log the real error, hide details from the client.
		logger.FromContext(r.Context()).Error().Err(err).Msg("unhandled error")
		httpx.Error(w, r, http.StatusInternalServerError, "internal server error")
	}
}
