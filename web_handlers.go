package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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

// NAEPScoreView extends NAEPScore with pre-calculated achievement levels for templating
type NAEPScoreView struct {
	NAEPScore
	BelowBasicPct    float64
	BasicPct         float64
	ProficientPct    float64
	AdvancedPct      float64
	NationalScore    *NAEPScoreView // Matching national score for comparison
	NationalCompare  string         // "Above" or "Below"
}

// NAEPDataView wraps NAEPData with enriched scores for templating
type NAEPDataView struct {
	NCESSCH         string
	State           string
	District        string
	ExtractedAt     time.Time
	StateScores     []NAEPScoreView
	DistrictScores  []NAEPScoreView
	NationalScores  []NAEPScoreView
	UseDistrict     bool
	Grade4Scores    []NAEPScoreView
	Grade8Scores    []NAEPScoreView
	NationalByKey   map[string]*NAEPScoreView // key: "subject-grade"
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
	var naepView *NAEPDataView
	if h.NAEPClient != nil && h.DB != nil {
		// Try to load from cache with 90-day TTL
		state, district, stateScoresJSON, districtScoresJSON, nationalScoresJSON, extractedAt, err := h.DB.LoadNAEPCache(school.NCESSCH, 90*24*time.Hour)
		if err == nil && len(stateScoresJSON) > 0 {
			// Parse the cached data
			naepData := &NAEPData{
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
			// Enrich the cached data for template
			naepView = h.enrichNAEPData(naepData)
		}
	}

	data := map[string]interface{}{
		"Title":        school.Name,
		"School":       school,
		"EnhancedData": enhancedData,
		"NAEPData":     naepView,
		"AIAvailable":  h.AIScraper != nil,
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

	// Extract AI data using ScrapeSchoolWebsite which properly fills metadata and caches
	enhancedData, err := h.AIScraper.ScrapeSchoolWebsite(r.Context(), school)
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

		// Check if this is a "no data available" error vs a real server error
		errMsg := err.Error()
		if strings.Contains(errMsg, "no NAEP data available") ||
		   strings.Contains(errMsg, "no NAEP grades applicable") {
			// Return 200 with content indicating no data available
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<div class="naep-no-data">
				<p class="help-text error-message">
					<strong>No NAEP Data Available</strong><br>
					Assessment data is not available for this school.
				</p>
			</div>`))
			return
		}

		// Real server error
		http.Error(w, "NAEP data fetch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to view model with pre-calculated data
	naepView := h.enrichNAEPData(naepData)

	data := map[string]interface{}{
		"NAEPData": naepView,
		"School":   school,
	}

	if err := h.templates.ExecuteTemplate(w, "naep_data.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// enrichNAEPData converts NAEPData to NAEPDataView with pre-calculated achievement levels
func (h *WebHandler) enrichNAEPData(data *NAEPData) *NAEPDataView {
	useDistrict := len(data.DistrictScores) > 0

	view := &NAEPDataView{
		NCESSCH:       data.NCESSCH,
		State:         data.State,
		District:      data.District,
		ExtractedAt:   data.ExtractedAt,
		UseDistrict:   useDistrict,
		NationalByKey: make(map[string]*NAEPScoreView),
	}

	// First, enrich national scores and create lookup map
	for _, score := range data.NationalScores {
		enrichedScore := h.enrichScore(score, data, false)
		view.NationalScores = append(view.NationalScores, enrichedScore)
		key := score.Subject + "-" + fmt.Sprintf("%d", score.Grade)
		view.NationalByKey[key] = &enrichedScore
	}

	// Enrich state scores with national comparison
	for _, score := range data.StateScores {
		enrichedScore := h.enrichScore(score, data, false)
		h.addNationalComparison(&enrichedScore, view.NationalByKey)
		view.StateScores = append(view.StateScores, enrichedScore)
	}

	// Enrich district scores with national comparison
	for _, score := range data.DistrictScores {
		enrichedScore := h.enrichScore(score, data, true)
		h.addNationalComparison(&enrichedScore, view.NationalByKey)
		view.DistrictScores = append(view.DistrictScores, enrichedScore)
	}

	// Group scores by grade for easier template iteration
	primaryScores := view.StateScores
	if useDistrict {
		primaryScores = view.DistrictScores
	}

	for _, score := range primaryScores {
		if score.Grade == 4 {
			view.Grade4Scores = append(view.Grade4Scores, score)
		} else if score.Grade == 8 {
			view.Grade8Scores = append(view.Grade8Scores, score)
		}
	}

	return view
}

// addNationalComparison adds national comparison data to a score
func (h *WebHandler) addNationalComparison(score *NAEPScoreView, nationalByKey map[string]*NAEPScoreView) {
	key := score.Subject + "-" + fmt.Sprintf("%d", score.Grade)
	if national, ok := nationalByKey[key]; ok {
		score.NationalScore = national
		if score.AtProficient >= national.AtProficient {
			score.NationalCompare = "Above"
		} else {
			score.NationalCompare = "Below"
		}
	}
}

// enrichScore adds achievement level percentages to a score
func (h *WebHandler) enrichScore(score NAEPScore, data *NAEPData, useDistrict bool) NAEPScoreView {
	belowBasic, basic, proficient, advanced := data.GetAchievementLevels(score.Subject, score.Grade, useDistrict)

	return NAEPScoreView{
		NAEPScore:     score,
		BelowBasicPct: belowBasic,
		BasicPct:      basic,
		ProficientPct: proficient,
		AdvancedPct:   advanced,
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
