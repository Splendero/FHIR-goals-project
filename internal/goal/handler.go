package goal

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/Goal", func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleSearch)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var g Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "error", "invalid", "invalid JSON: "+err.Error())
		return
	}

	created, err := h.svc.Create(r.Context(), &g)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	g, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]string)
	if v := r.URL.Query().Get("patient"); v != "" {
		params["patient"] = strings.TrimPrefix(v, "Patient/")
	}
	if v := r.URL.Query().Get("status"); v != "" {
		params["status"] = v
	}
	if v := r.URL.Query().Get("category"); v != "" {
		params["category"] = v
	}

	goals, err := h.svc.Search(r.Context(), params)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	entries := make([]fhir.BundleEntry, len(goals))
	for i, g := range goals {
		entries[i] = fhir.BundleEntry{
			FullURL:  "Goal/" + g.ID,
			Resource: g,
		}
	}
	bundle := fhir.Bundle{
		ResourceType: "Bundle",
		Type:         "searchset",
		Total:        len(goals),
		Entry:        entries,
	}

	writeJSON(w, http.StatusOK, bundle)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var g Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "error", "invalid", "invalid JSON: "+err.Error())
		return
	}

	updated, err := h.svc.Update(r.Context(), id, &g)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrValidation) {
		writeError(w, http.StatusBadRequest, "error", "required", err.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "error", "not-found", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "error", "exception", err.Error())
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, severity, code, message string) {
	writeJSON(w, status, fhir.NewErrorOutcome(severity, code, message))
}
