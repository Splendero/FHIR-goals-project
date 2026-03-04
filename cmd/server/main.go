package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/spencerosborn/fhir-goals-engine/internal/careplan"
	"github.com/spencerosborn/fhir-goals-engine/internal/config"
	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
	"github.com/spencerosborn/fhir-goals-engine/internal/observation"
	"github.com/spencerosborn/fhir-goals-engine/internal/patient"
	"github.com/spencerosborn/fhir-goals-engine/internal/suggestion"
	"github.com/spencerosborn/fhir-goals-engine/internal/websocket"
)

func main() {
	cfg := config.Load()

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	runMigrations(db)

	hub := websocket.NewHub()
	go hub.Run()

	patientRepo := patient.NewRepository(db)
	goalRepo := goal.NewRepository(db)
	carePlanRepo := careplan.NewRepository(db)
	obsRepo := observation.NewRepository(db)

	patientSvc := patient.NewService(patientRepo)
	goalSvc := goal.NewService(goalRepo)
	carePlanSvc := careplan.NewService(carePlanRepo)

	evaluator := goal.NewEvaluator(goalRepo)
	obsSvc := observation.NewService(obsRepo, evaluator, hub)

	suggestionSvc := suggestion.NewService(cfg.OpenAIKey, obsRepo, patientRepo)

	patientHandler := patient.NewHandler(patientSvc)
	goalHandler := goal.NewHandler(goalSvc)
	carePlanHandler := careplan.NewHandler(carePlanSvc)
	obsHandler := observation.NewHandler(obsSvc)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/ws", hub.HandleWebSocket)

	r.Route("/", func(r chi.Router) {
		patientHandler.RegisterRoutes(r)
		goalHandler.RegisterRoutes(r)
		carePlanHandler.RegisterRoutes(r)
		obsHandler.RegisterRoutes(r)
	})

	r.Post("/Goal/$suggest", suggestHandler(suggestionSvc))

	r.Get("/metadata", capabilityHandler)

	staticDir := http.Dir("./static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticDir)))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("FHIR Goals Engine started on :%s", cfg.Port)
		log.Printf("Dashboard: http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func runMigrations(db *sqlx.DB) {
	migration, err := os.ReadFile("./migrations/001_create_tables.up.sql")
	if err != nil {
		log.Printf("No migration file found, skipping: %v", err)
		return
	}
	if _, err := db.Exec(string(migration)); err != nil {
		log.Printf("Migration note (tables may already exist): %v", err)
	}
}

type suggestRequest struct {
	PatientID string `json:"patientId"`
}

func suggestHandler(svc *suggestion.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req suggestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/fhir+json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", "invalid", "Invalid JSON"))
			return
		}

		if req.PatientID == "" {
			w.Header().Set("Content-Type", "application/fhir+json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", "required", "patientId is required"))
			return
		}

		goals, err := svc.SuggestGoals(r.Context(), req.PatientID)
		if err != nil {
			w.Header().Set("Content-Type", "application/fhir+json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(fhir.NewErrorOutcome("error", "exception", err.Error()))
			return
		}

		entries := make([]fhir.BundleEntry, len(goals))
		for i, g := range goals {
			entries[i] = fhir.BundleEntry{Resource: g}
		}
		bundle := fhir.Bundle{
			ResourceType: "Bundle",
			Type:         "collection",
			Total:        len(goals),
			Entry:        entries,
		}

		w.Header().Set("Content-Type", "application/fhir+json")
		json.NewEncoder(w).Encode(bundle)
	}
}

func capabilityHandler(w http.ResponseWriter, r *http.Request) {
	cap := map[string]interface{}{
		"resourceType": "CapabilityStatement",
		"status":       "active",
		"kind":         "instance",
		"fhirVersion":  "4.0.1",
		"format":       []string{"json"},
		"rest": []map[string]interface{}{
			{
				"mode": "server",
				"resource": []map[string]interface{}{
					{"type": "Patient", "interaction": []map[string]string{{"code": "read"}, {"code": "search-type"}, {"code": "create"}, {"code": "update"}, {"code": "delete"}}},
					{"type": "Goal", "interaction": []map[string]string{{"code": "read"}, {"code": "search-type"}, {"code": "create"}, {"code": "update"}, {"code": "delete"}}},
					{"type": "CarePlan", "interaction": []map[string]string{{"code": "read"}, {"code": "search-type"}, {"code": "create"}, {"code": "update"}, {"code": "delete"}}},
					{"type": "Observation", "interaction": []map[string]string{{"code": "read"}, {"code": "search-type"}, {"code": "create"}, {"code": "update"}, {"code": "delete"}}},
				},
				"operation": []map[string]string{
					{"name": "suggest", "definition": "Goal/$suggest"},
				},
			},
		},
		"description": fmt.Sprintf("FHIR Goals Engine - Health Goals Tracking and AI-Powered Suggestion Platform"),
	}

	w.Header().Set("Content-Type", "application/fhir+json")
	json.NewEncoder(w).Encode(cap)
}
