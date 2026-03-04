package observation

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

const fhirContentType = "application/fhir+json"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/Observation", func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleSearch)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var o Observation
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "Invalid JSON: "+err.Error())
		return
	}

	created, err := h.svc.Create(r.Context(), &o)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrCodeRequired) {
			writeError(w, http.StatusBadRequest, "required", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to create observation")
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	w.Header().Set("Location", "/Observation/"+created.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	o, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Observation not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(o)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{}
	if patient := r.URL.Query().Get("patient"); patient != "" {
		params["patient"] = patient
	}
	if code := r.URL.Query().Get("code"); code != "" {
		params["code"] = code
	}
	if date := r.URL.Query().Get("date"); date != "" {
		params["date"] = date
	}

	observations, err := h.svc.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "exception", "Search failed")
		return
	}

	resources := make([]interface{}, len(observations))
	for i := range observations {
		resources[i] = observations[i]
	}

	bundle := fhir.NewSearchBundle(resources, r.URL.Path)
	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(bundle)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var o Observation
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "Invalid JSON: "+err.Error())
		return
	}

	updated, err := h.svc.Update(r.Context(), id, &o)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrCodeRequired) {
			writeError(w, http.StatusBadRequest, "required", err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Observation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to update observation")
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(updated)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Observation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to delete observation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", fhirContentType)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", code, message))
}
