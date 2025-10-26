package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ServerConfig holds configuration for the web server
type ServerConfig struct {
	Port       int
	DB         *DB
	AIScraper  *AIScraperService
	NAEPClient *NAEPClient
	DataPath   string
}

// StartServer initializes and starts the HTTP server
func StartServer(config ServerConfig) error {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Favicon route - serve from project root
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./static/favicon.ico")
	})

	// Web handlers (HTMX HTML responses)
	webHandler := NewWebHandler(config.DB, config.AIScraper, config.NAEPClient)
	r.Get("/", webHandler.SearchPage)
	r.Post("/search", webHandler.SearchResults)
	r.Get("/schools/{id}", webHandler.SchoolDetail)
	r.Post("/schools/{id}/ai", webHandler.ExtractAI)
	r.Post("/schools/{id}/naep", webHandler.FetchNAEP)

	// API handlers (JSON responses)
	apiHandler := &APIHandler{DB: config.DB, AIScraper: config.AIScraper}
	r.Route("/api", func(r chi.Router) {
		r.Get("/search", apiHandler.Search)
		r.Get("/schools/{id}", apiHandler.GetSchool)
		r.Post("/schools/{id}/ai", apiHandler.ExtractAI)
	})

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Starting server on http://localhost%s", addr)
	return http.ListenAndServe(addr, r)
}
