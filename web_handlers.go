package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

// WebHandler handles HTMX HTML requests
type WebHandler struct {
	DB         *DB
	AIScraper  *AIScraperService
	NAEPClient *NAEPClient
	templates  *template.Template
}

// NewWebHandler creates a new WebHandler with parsed templates
func NewWebHandler(db *DB, aiScraper *AIScraperService, naepClient *NAEPClient) *WebHandler {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	template.Must(tmpl.ParseGlob("templates/partials/*.html"))
	return &WebHandler{
		DB:         db,
		AIScraper:  aiScraper,
		NAEPClient: naepClient,
		templates:  tmpl,
	}
}

// SearchPage renders the main search page
func (h *WebHandler) SearchPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "School Finder",
		"Query": r.URL.Query().Get("q"),
		"State": r.URL.Query().Get("state"),
	}

	if err := h.templates.ExecuteTemplate(w, "search.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SearchResults handles search requests and returns results partial
func (h *WebHandler) SearchResults(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	query := r.FormValue("query")
	state := r.FormValue("state")

	schools, err := h.DB.SearchSchools(query, state, maxResults)
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Schools": schools,
		"Query":   query,
		"State":   state,
		"Count":   len(schools),
	}

	if err := h.templates.ExecuteTemplate(w, "results.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SchoolDetail renders the school detail page
func (h *WebHandler) SchoolDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	school, err := h.DB.GetSchoolByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if we have cached AI data (requires AI scraper)
	var enhancedData *EnhancedSchoolData
	if h.AIScraper != nil {
		// Try to load from cache using the AI scraper's private method
		// We'll check the database cache directly with a reasonable TTL
		_, sourceURL, markdownContent, legacyData, extractedAt, err := h.DB.LoadAIScraperCache(school.NCESSCH, 30*24*time.Hour)
		if err == nil && sourceURL != "" {
			// Parse the cached data
			enhancedData = &EnhancedSchoolData{
				NCESSCH:         school.NCESSCH,
				SchoolName:      school.Name,
				SourceURL:       sourceURL,
				MarkdownContent: markdownContent,
				ExtractedAt:     extractedAt,
			}
			// Parse legacy data if available
			if len(legacyData) > 0 {
				json.Unmarshal(legacyData, enhancedData)
			}
		}
	}

	// Check if we have cached NAEP data
	var naepData *NAEPData
	if h.NAEPClient != nil && h.DB != nil {
		// Try to load from cache with 90-day TTL
		state, district, stateScoresJSON, districtScoresJSON, nationalScoresJSON, extractedAt, err := h.DB.LoadNAEPCache(school.NCESSCH, 90*24*time.Hour)
		if err == nil && len(stateScoresJSON) > 0 {
			// Parse the cached data
			naepData = &NAEPData{
				NCESSCH:     school.NCESSCH,
				State:       state,
				District:    district,
				ExtractedAt: extractedAt,
			}
			// Unmarshal state scores
			if len(stateScoresJSON) > 0 {
				json.Unmarshal(stateScoresJSON, &naepData.StateScores)
			}
			// Unmarshal district scores
			if len(districtScoresJSON) > 0 {
				json.Unmarshal(districtScoresJSON, &naepData.DistrictScores)
			}
			// Unmarshal national scores
			if len(nationalScoresJSON) > 0 {
				json.Unmarshal(nationalScoresJSON, &naepData.NationalScores)
			}
		}
	}

	data := map[string]interface{}{
		"Title":        school.Name,
		"School":       school,
		"EnhancedData": enhancedData,
		"NAEPData":     naepData,
	}

	if err := h.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ExtractAI handles AI extraction requests and returns AI data partial
func (h *WebHandler) ExtractAI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	school, err := h.DB.GetSchoolByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if AI scraper is available
	if h.AIScraper == nil {
		http.Error(w, "AI extraction not available: ANTHROPIC_API_KEY not set", http.StatusServiceUnavailable)
		return
	}

	// Extract AI data
	enhancedData, err := h.AIScraper.ExtractSchoolDataWithWebSearch(r.Context(), school)
	if err != nil {
		log.Printf("AI extraction error: %v", err)
		http.Error(w, "AI extraction failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"EnhancedData": enhancedData,
		"School":       school,
	}

	if err := h.templates.ExecuteTemplate(w, "ai_data.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// FetchNAEP handles NAEP data fetching requests and returns NAEP data partial
func (h *WebHandler) FetchNAEP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	school, err := h.DB.GetSchoolByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if NAEP client is available
	if h.NAEPClient == nil {
		http.Error(w, "NAEP data not available", http.StatusServiceUnavailable)
		return
	}

	// Fetch NAEP data
	naepData, err := h.NAEPClient.FetchNAEPData(school)
	if err != nil {
		log.Printf("NAEP fetch error: %v", err)
		http.Error(w, "NAEP data fetch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"NAEPData": naepData,
		"School":   school,
	}

	if err := h.templates.ExecuteTemplate(w, "naep_data.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Helper function to parse templates
func parseTemplates() (*template.Template, error) {
	tmpl := template.New("")

	// Parse all templates
	layouts, err := filepath.Glob("templates/*.html")
	if err != nil {
		return nil, err
	}

	partials, err := filepath.Glob("templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	allTemplates := append(layouts, partials...)
	for _, file := range allTemplates {
		_, err := tmpl.ParseFiles(file)
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}
