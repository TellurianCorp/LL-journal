// LifeLogger LL-Journal Server
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/telluriancorp/ll-journal/internal/config"
	"github.com/telluriancorp/ll-journal/internal/git"
	"github.com/telluriancorp/ll-journal/internal/handlers"
	"github.com/telluriancorp/ll-journal/internal/journal"
	"github.com/telluriancorp/ll-journal/internal/migrations"
	"github.com/telluriancorp/ll-journal/internal/s3"
	"github.com/telluriancorp/ll-journal/internal/store"
)

const version = "0.1.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Failed to load configuration: %v. Using defaults.", err)
		cfg = config.Default()
	}

	envMode := strings.ToLower(os.Getenv("ENV"))
	if envMode == "" {
		envMode = strings.ToLower(os.Getenv("APP_ENV"))
	}
	if envMode == "" {
		envMode = "development"
	}

	// Initialize store
	var st *store.Store
	if cfg.DatabaseURL != "" {
		st, err = store.New(cfg.DatabaseURL)
		if err != nil {
			if envMode == "production" {
				log.Fatalf("Failed to connect to database in production: %v", err)
			}
			log.Printf("Warning: failed to connect to database (%v); service will not function properly", err)
		} else {
			log.Printf("Connected to database")
		}
	} else {
		if envMode == "production" {
			log.Fatalf("Production mode requires database connection; LL_JOURNAL_DATABASE_URL missing")
		}
		log.Printf("Warning: No database URL provided")
	}

	if st == nil {
		log.Fatalf("Database connection required")
	}

	// Run database migrations automatically
	log.Printf("Running database migrations...")
	if err := migrations.RunMigrations(st.DB()); err != nil {
		log.Printf("Warning: Failed to run migrations: %v. Continuing anyway...", err)
		log.Printf("You may need to run migrations manually if tables are missing")
	} else {
		log.Printf("Database migrations completed successfully")
	}

	// Initialize S3 client
	var s3Client *s3.Client
	if cfg.S3Endpoint != "" && cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		s3Client, err = s3.New(s3.Config{
			Endpoint:  cfg.S3Endpoint,
			Bucket:     cfg.S3Bucket,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Region:    "us-east-1",
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 client: %v", err)
		}
		log.Printf("S3 client initialized (bucket: %s)", cfg.S3Bucket)
	} else {
		if envMode == "production" {
			log.Fatalf("Production mode requires S3 configuration")
		}
		log.Printf("Warning: S3 not configured")
	}

	// Initialize Git client
	gitClient, err := git.New(cfg.GitRoot)
	if err != nil {
		log.Fatalf("Failed to initialize Git client: %v", err)
	}
	log.Printf("Git client initialized (root: %s)", cfg.GitRoot)

	// Initialize journal service
	journalService := journal.NewService(st, s3Client, gitClient)

	// Initialize handlers
	h := handlers.New(journalService)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check endpoint
	r.Get("/health", healthHandler)

	// API routes
	r.Route("/api/journals", func(r chi.Router) {
		r.Post("/", h.CreateJournal)
		r.Get("/", h.ListJournals)
		r.Get("/{id}", h.GetJournal)
		r.Put("/{id}", h.UpdateJournal)
		r.Delete("/{id}", h.DeleteJournal)

		// Entry routes
		r.Route("/{journalId}/entries", func(r chi.Router) {
			r.Post("/", h.CreateEntry)
			r.Get("/", h.ListEntries)
			r.Get("/{date}", h.GetEntry)
			r.Put("/{date}", h.UpdateEntry)
			r.Delete("/{date}", h.DeleteEntry)

			// Version routes
			r.Get("/{date}/versions", h.ListVersions)
			r.Get("/{date}/versions/{commit}", h.GetVersion)
		})
	})

	// Start server
	addr := cfg.SocketAddr()
	log.Printf("LL-Journal version: %s", version)
	log.Printf("Starting LL-Journal on %s", addr)
	log.Printf("Note: Authentication and routing handled by LL-proxy gateway")

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "ll-journal",
		"version": version,
	})
}
