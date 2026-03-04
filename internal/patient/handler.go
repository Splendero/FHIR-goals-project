package patient

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
	r.Route("/Patient", func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleSearch)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p Patient
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "Invalid JSON: "+err.Error())
		return
	}

	created, err := h.svc.Create(r.Context(), &p)
	if err != nil {
		if errors.Is(err, ErrNameRequired) {
			writeError(w, http.StatusBadRequest, "required", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to create patient")
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	w.Header().Set("Location", "/Patient/"+created.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Patient not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(p)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{}
	if name := r.URL.Query().Get("name"); name != "" {
		params["name"] = name
	}
	if gender := r.URL.Query().Get("gender"); gender != "" {
		params["gender"] = gender
	}

	patients, err := h.svc.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "exception", "Search failed")
		return
	}

	resources := make([]interface{}, len(patients))
	for i := range patients {
		resources[i] = patients[i]
	}

	bundle := fhir.NewSearchBundle(resources, r.URL.Path)
	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(bundle)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var p Patient
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "Invalid JSON: "+err.Error())
		return
	}

	updated, err := h.svc.Update(r.Context(), id, &p)
	if err != nil {
		if errors.Is(err, ErrNameRequired) {
			writeError(w, http.StatusBadRequest, "required", err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Patient not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to update patient")
		return
	}

	w.Header().Set("Content-Type", fhirContentType)
	json.NewEncoder(w).Encode(updated)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not-found", "Patient not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "exception", "Failed to delete patient")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", fhirContentType)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", code, message))
}
