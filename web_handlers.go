package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"github.com/go-chi/chi/v5"
	"schoolfinder/internal/agent"
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
	BelowBasicPct   float64
	BasicPct        float64
	ProficientPct   float64
	AdvancedPct     float64
	NationalScore   *NAEPScoreView // Matching national score for comparison
	NationalCompare string         // "Above" or "Below"
}

// NAEPDataView wraps NAEPData with enriched scores for templating
type NAEPDataView struct {
	NCESSCH        string
	State          string
	District       string
	ExtractedAt    time.Time
	StateScores    []NAEPScoreView
	DistrictScores []NAEPScoreView
	NationalScores []NAEPScoreView
	UseDistrict    bool
	Grade4Scores   []NAEPScoreView
	Grade8Scores   []NAEPScoreView
	NationalByKey  map[string]*NAEPScoreView // key: "subject-grade"
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
				if err := json.Unmarshal(legacyData, enhancedData); err != nil {
					// Log error but continue - enhanced data is optional
					log.Printf("Warning: failed to unmarshal legacy data: %v", err)
				}
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
				if err := json.Unmarshal(stateScoresJSON, &naepData.StateScores); err != nil {
					log.Printf("Warning: failed to unmarshal state scores: %v", err)
				}
			}
			// Unmarshal district scores
			if len(districtScoresJSON) > 0 {
				if err := json.Unmarshal(districtScoresJSON, &naepData.DistrictScores); err != nil {
					log.Printf("Warning: failed to unmarshal district scores: %v", err)
				}
			}
			// Unmarshal national scores
			if len(nationalScoresJSON) > 0 {
				if err := json.Unmarshal(nationalScoresJSON, &naepData.NationalScores); err != nil {
					log.Printf("Warning: failed to unmarshal national scores: %v", err)
				}
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
			if _, err := w.Write([]byte(`<div class="naep-no-data">
				<p class="help-text error-message">
					<strong>No NAEP Data Available</strong><br>
					Assessment data is not available for this school.
				</p>
			</div>`)); err != nil {
				log.Printf("Warning: failed to write response: %v", err)
			}
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
		switch score.Grade {
		case 4:
			view.Grade4Scores = append(view.Grade4Scores, score)
		case 8:
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

// AgentPage renders the AI agent page
func (h *WebHandler) AgentPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":       "AI Agent",
		"Query":       r.URL.Query().Get("q"),
		"AIAvailable": h.AIScraper != nil,
	}

	if err := h.templates.ExecuteTemplate(w, "agent.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentQueryResponse holds the response data for agent queries
type AgentQueryResponse struct {
	Query        string
	ResponseText string
	Schools      []*School
	TotalCount   int
	Page         int
	PageSize     int
	TotalPages   int
	StartIndex   int
	EndIndex     int
	PrevPage     int
	NextPage     int
	SchoolIDs    string
	Error        string
}

// AgentQuery handles AI agent queries
func (h *WebHandler) AgentQuery(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	query := r.FormValue("query")
	if query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	// Check if AI scraper is available
	if h.AIScraper == nil {
		data := AgentQueryResponse{
			Query: query,
			Error: "AI Agent requires ANTHROPIC_API_KEY to be set",
		}
		if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Use Claude to interpret the query and generate a SQL search
	response, schoolIDs, err := h.queryWithAI(r.Context(), query)
	if err != nil {
		log.Printf("AI query error: %v", err)
		data := AgentQueryResponse{
			Query: query,
			Error: fmt.Sprintf("Failed to process query: %v", err),
		}
		if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the schools by IDs
	var schools []*School
	if len(schoolIDs) > 0 {
		schools, err = h.DB.GetSchoolsByIDs(schoolIDs)
		if err != nil {
			log.Printf("Database error fetching schools: %v", err)
			data := AgentQueryResponse{
				Query:        query,
				ResponseText: response,
				Error:        "Failed to fetch school details",
			}
			if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
				log.Printf("Template error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	// Paginate results
	pageSize := 20
	totalCount := len(schools)
	page := 1

	// Calculate pagination
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	startIdx := 0
	endIdx := totalCount
	if totalCount > pageSize {
		endIdx = pageSize
	}

	paginatedSchools := schools[startIdx:endIdx]

	// Convert school IDs to comma-separated string for pagination
	schoolIDsStr := strings.Join(schoolIDs, ",")

	data := AgentQueryResponse{
		Query:        query,
		ResponseText: response,
		Schools:      paginatedSchools,
		TotalCount:   totalCount,
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		StartIndex:   startIdx + 1,
		EndIndex:     endIdx,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		SchoolIDs:    schoolIDsStr,
	}

	if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentPaginate handles pagination for agent query results
func (h *WebHandler) AgentPaginate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	query := r.FormValue("query")
	pageStr := r.FormValue("page")
	schoolIDsStr := r.FormValue("school_ids")

	page := 1
	if pageStr != "" {
		if p, err := fmt.Sscanf(pageStr, "%d", &page); err != nil || p != 1 {
			page = 1
		}
	}

	// Parse school IDs
	var schoolIDs []string
	if schoolIDsStr != "" {
		schoolIDs = strings.Split(schoolIDsStr, ",")
	}

	// Fetch schools
	schools, err := h.DB.GetSchoolsByIDs(schoolIDs)
	if err != nil {
		log.Printf("Database error: %v", err)
		data := AgentQueryResponse{
			Query: query,
			Error: "Failed to fetch schools",
		}
		if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Paginate
	pageSize := 20
	totalCount := len(schools)
	totalPages := (totalCount + pageSize - 1) / pageSize

	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > totalCount {
		endIdx = totalCount
	}

	paginatedSchools := schools[startIdx:endIdx]

	data := AgentQueryResponse{
		Query:      query,
		Schools:    paginatedSchools,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		StartIndex: startIdx + 1,
		EndIndex:   endIdx,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		SchoolIDs:  schoolIDsStr,
	}

	if err := h.templates.ExecuteTemplate(w, "agent_response.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// queryWithAI uses Fantasy agent to interpret natural language queries and execute SQL
// The agent has built-in retry logic and will self-correct failed SQL queries
func (h *WebHandler) queryWithAI(ctx context.Context, query string) (string, []string, error) {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Create Anthropic provider for Fantasy
	provider, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create language model (use Haiku 4.5 for speed)
	model, err := provider.LanguageModel(ctx, "claude-haiku-4-5")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Define system prompt for data exploration
	systemPrompt := `You are a data analyst helping users explore a school database with 102,274 schools from the NCES Common Core of Data (CCD).

**Your Task:**
Answer the user's question by querying the database using the 'query' tool.

**Available Tools:**
- 'query': Execute SQL queries against the DuckDB database
- 'schema': Get database schema information

**Database Schema:**
- **directory**: School information (NCESSCH, SCH_NAME, ST, STATENAME, MCITY, LEA_NAME, SCH_TYPE_TEXT, LEVEL, GSLO, GSHI, CHARTER_TEXT, PHONE, WEBSITE, MSTREET1, MZIP, SCHOOL_YEAR)
- **enrollment**: Student counts (NCESSCH, STUDENT_COUNT, TOTAL_INDICATOR - use = 'Education Unit Total' for totals)
- **teachers**: Teacher FTE counts (NCESSCH, TEACHERS)

**Query Strategy:**
1. For SEARCH queries (finding specific schools): Return SQL that selects NCESSCH IDs
   Example: "SELECT d.NCESSCH FROM directory d WHERE d.ST = 'CA' AND d.LEVEL = 'High' LIMIT 200"

2. For ANALYSIS queries (statistics/aggregations): Return SQL with aggregated results
   Example: "SELECT d.ST, AVG(e.STUDENT_COUNT) as avg_enrollment FROM directory d LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH WHERE e.TOTAL_INDICATOR = 'Education Unit Total' GROUP BY d.ST"

**Important SQL Guidelines:**
- JOIN on NCESSCH
- For enrollment: Use "e.TOTAL_INDICATOR = 'Education Unit Total'" in JOIN condition
- Student-teacher ratio: CAST(e.STUDENT_COUNT AS FLOAT) / t.TEACHERS
- Use ST (2-letter code) not STATENAME for filtering
- Limit search results to 200
- Database is DuckDB (PostgreSQL-compatible)

**If SQL Fails:**
The query tool will return an error. Analyze the error, correct your SQL, and try again. Common issues:
- Column names are case-sensitive
- Missing JOIN conditions
- Incorrect aggregate usage

**Response Format:**
1. Execute the query using the 'query' tool
2. Summarize the results in natural language
3. If it's a search query, list the school IDs found
4. If it's an analysis, present the aggregated data clearly`

	// Create query tool for SQL execution
	queryTool := fantasy.NewAgentTool(
		"query",
		"Execute a SQL query against the DuckDB database and return results as JSON",
		func(ctx context.Context, input agent.QueryInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.SQL == "" {
				return fantasy.NewTextErrorResponse("sql parameter is required"), nil
			}

			// Execute the query using the DB
			rows, err := h.DB.ExecuteQuery(input.SQL)
			if err != nil {
				// Return the error so agent can retry with corrected SQL
				return fantasy.NewTextErrorResponse(fmt.Sprintf("SQL error: %v", err)), nil
			}

			// Convert to JSON
			jsonBytes, err := json.MarshalIndent(rows, "", "  ")
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to encode results: %v", err)), nil
			}

			return fantasy.NewTextResponse(string(jsonBytes)), nil
		},
	)

	// Create schema tool for database introspection
	schemaTool := fantasy.NewAgentTool(
		"schema",
		"Get database schema information for all tables",
		func(ctx context.Context, input agent.SchemaInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			tables := []string{"directory", "teachers", "enrollment"}
			type SchemaOutput struct {
				TableName string `json:"table_name"`
				Columns   []struct {
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"columns"`
			}
			schemas := make([]SchemaOutput, 0, len(tables))

			for _, tableName := range tables {
				pragmaQuery := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
				rows, err := h.DB.ExecuteQuery(pragmaQuery)
				if err != nil {
					continue
				}

				schema := SchemaOutput{
					TableName: tableName,
					Columns:   make([]struct{ Name string `json:"name"`; Type string `json:"type"` }, 0),
				}

				for _, row := range rows {
					name, _ := row["name"].(string)
					colType, _ := row["type"].(string)
					schema.Columns = append(schema.Columns, struct {
						Name string `json:"name"`
						Type string `json:"type"`
					}{Name: name, Type: colType})
				}

				schemas = append(schemas, schema)
			}

			jsonBytes, _ := json.MarshalIndent(schemas, "", "  ")
			return fantasy.NewTextResponse(string(jsonBytes)), nil
		},
	)

	// Create Fantasy agent with tools (Fantasy handles retries internally)
	fantasyAgent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(queryTool, schemaTool),
	)

	// Generate response using the agent
	result, err := fantasyAgent.Generate(ctx, fantasy.AgentCall{Prompt: query})
	if err != nil {
		return "", nil, fmt.Errorf("agent generation failed: %w", err)
	}

	// Extract response text
	responseText := result.Response.Content.Text()

	// Parse school IDs from the response if it's a search query
	// Look for NCESSCH values in the response
	var schoolIDs []string

	// Try to find JSON arrays in the response that might contain school IDs
	if strings.Contains(responseText, "NCESSCH") {
		// Extract school IDs from JSON-like patterns
		lines := strings.Split(responseText, "\n")
		for _, line := range lines {
			// Look for NCESSCH patterns like: "NCESSCH": "123456789012"
			if strings.Contains(line, `"NCESSCH"`) || strings.Contains(line, `"ncessch"`) {
				// Extract the ID value
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					idPart := strings.TrimSpace(parts[1])
					idPart = strings.Trim(idPart, `",`)
					if len(idPart) == 12 { // NCESSCH IDs are 12 digits
						schoolIDs = append(schoolIDs, idPart)
					}
				}
			}
		}
	}

	return responseText, schoolIDs, nil
}
