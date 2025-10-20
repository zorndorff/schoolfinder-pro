package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// APIHandler handles JSON API requests
type APIHandler struct {
	DB        *DB
	AIScraper *AIScraperService
}

// Search handles API search requests
func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	state := r.URL.Query().Get("state")

	schools, err := h.DB.SearchSchools(query, state, maxResults)
	if err != nil {
		log.Printf("Search error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Search failed",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schools": schools,
		"count":   len(schools),
		"query":   query,
		"state":   state,
	})
}

// GetSchool handles API requests for a single school
func (h *APIHandler) GetSchool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	school, err := h.DB.GetSchoolByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error": "School not found",
			})
			return
		}
		log.Printf("Database error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
		return
	}

	// Check for cached AI data (requires AI scraper)
	var enhancedData *EnhancedSchoolData
	if h.AIScraper != nil {
		// Try to load from cache with a reasonable TTL
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"school":       school,
		"enhancedData": enhancedData,
	})
}

// ExtractAI handles API requests for AI extraction
func (h *APIHandler) ExtractAI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	school, err := h.DB.GetSchoolByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error": "School not found",
			})
			return
		}
		log.Printf("Database error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
		return
	}

	// Check if AI scraper is available
	if h.AIScraper == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "AI extraction not available: ANTHROPIC_API_KEY not set",
		})
		return
	}

	enhancedData, err := h.AIScraper.ExtractSchoolDataWithWebSearch(r.Context(), school)
	if err != nil {
		log.Printf("AI extraction error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "AI extraction failed: " + err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"school":       school,
		"enhancedData": enhancedData,
	})
}

// respondJSON is a helper function to send JSON responses
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}
