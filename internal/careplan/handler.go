package careplan

import (
	"encoding/json"
	"net/http"

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
	r.Route("/CarePlan", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.search)
		r.Get("/{id}", h.getByID)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var cp CarePlan
	if err := json.NewDecoder(r.Body).Decode(&cp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}

	created, err := h.svc.Create(r.Context(), &cp)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	cp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "exception", err.Error())
		return
	}
	if cp == nil {
		writeError(w, http.StatusNotFound, "not-found", "CarePlan/"+id+" not found")
		return
	}

	writeJSON(w, http.StatusOK, cp)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]string)
	if v := r.URL.Query().Get("patient"); v != "" {
		params["patient"] = v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		params["status"] = v
	}

	results, err := h.svc.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "exception", err.Error())
		return
	}

	resources := make([]interface{}, len(results))
	for i := range results {
		resources[i] = results[i]
	}
	bundle := fhir.NewSearchBundle(resources, r.URL.Path)
	writeJSON(w, http.StatusOK, bundle)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var cp CarePlan
	if err := json.NewDecoder(r.Body).Decode(&cp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}

	updated, err := h.svc.Update(r.Context(), id, &cp)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid", err.Error())
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "not-found", "CarePlan/"+id+" not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not-found", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", code, message))
}
