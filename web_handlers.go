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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/go-chi/chi/v5"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"schoolfinder/internal/agent"
)

// WebHandler handles HTMX HTML requests
type WebHandler struct {
	DB         *DB
	AIScraper  *AIScraperService
	NAEPClient *NAEPClient
	templates  *template.Template
}

// markdownToHTML converts markdown text to HTML
func markdownToHTML(md string) template.HTML {
	// Create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))

	// Create HTML renderer with options
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render markdown to HTML
	htmlBytes := markdown.Render(doc, renderer)

	// Return as template.HTML to prevent escaping
	return template.HTML(htmlBytes)
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
	Query         string
	ResponseText  string        // AI's text summary/answer (markdown)
	ResponseHTML  template.HTML // AI's response converted to HTML
	SQLQuery      string        // The SQL query executed
	TableData     []map[string]interface{} // Raw query results as table
	TableColumns  []string // Column names for table display
	Schools       []*School
	TotalCount    int
	Page          int
	PageSize      int
	TotalPages    int
	StartIndex    int
	EndIndex      int
	PrevPage      int
	NextPage      int
	SchoolIDs     string
	Error         string
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
	result, err := h.queryWithAI(r.Context(), query)
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

	// Fetch the schools by IDs (if the query returned school IDs)
	var schools []*School
	if len(result.SchoolIDs) > 0 {
		schools, err = h.DB.GetSchoolsByIDs(result.SchoolIDs)
		if err != nil {
			log.Printf("Database error fetching schools: %v", err)
			data := AgentQueryResponse{
				Query:        query,
				ResponseText: result.ResponseText,
				ResponseHTML: markdownToHTML(result.ResponseText),
				SQLQuery:     result.SQLQuery,
				TableData:    result.TableData,
				TableColumns: result.TableColumns,
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
	schoolIDsStr := strings.Join(result.SchoolIDs, ",")

	data := AgentQueryResponse{
		Query:        query,
		ResponseText: result.ResponseText,
		ResponseHTML: markdownToHTML(result.ResponseText),
		SQLQuery:     result.SQLQuery,
		TableData:    result.TableData,
		TableColumns: result.TableColumns,
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
func (h *WebHandler) queryWithAI(ctx context.Context, query string) (*AIQueryResult, error) {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Create Anthropic provider for Fantasy
	provider, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create language model (use Haiku 4.5 for speed)
	model, err := provider.LanguageModel(ctx, "claude-haiku-4-5")
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Define system prompt for data exploration
	systemPrompt := `You are a data analyst helping users explore a school database with 102,274 schools from the NCES Common Core of Data (CCD), as well as any user-imported custom datasets.

**Your Task:**
Answer the user's question by querying the database using the 'query' tool. The tool will return a SUMMARY of the query results. Use this summary to provide a clear, natural language answer to the user's question.

**Available Tools:**
- 'query': Execute SQL queries against the DuckDB database (returns a summary of results)
- 'schema': Get database schema information for ALL tables including user-imported data

**Core Database Schema:**
- **directory**: School information (NCESSCH, SCH_NAME, ST, STATENAME, MCITY, LEA_NAME, SCH_TYPE_TEXT, LEVEL, GSLO, GSHI, CHARTER_TEXT, PHONE, WEBSITE, MSTREET1, MZIP, SCHOOL_YEAR)
- **enrollment**: Student counts (NCESSCH, STUDENT_COUNT, TOTAL_INDICATOR - use = 'Education Unit Total' for totals)
- **teachers**: Teacher FTE counts (NCESSCH, TEACHERS)

**User-Imported Tables:**
- Users can import custom CSV datasets which appear as additional tables
- Use the 'schema' tool to discover all available tables and their columns
- Table and column comments provide important context about user-imported data

**Query Strategy:**
1. For SEARCH queries (finding specific schools): Return SQL that selects NCESSCH IDs and relevant school info
   Example: "SELECT d.NCESSCH, d.SCH_NAME, d.MCITY, d.ST FROM directory d WHERE d.ST = 'CA' AND d.LEVEL = 'High' LIMIT 200"

2. For ANALYSIS queries (statistics/aggregations): Return SQL with aggregated results
   Example: "SELECT d.ST, AVG(e.STUDENT_COUNT) as avg_enrollment FROM directory d LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH WHERE e.TOTAL_INDICATOR = 'Education Unit Total' GROUP BY d.ST ORDER BY avg_enrollment DESC"

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
2. Analyze the summary results returned by the tool
3. Provide a clear, natural language answer based on the summary
4. If it's a search query, mention how many schools were found
5. If it's an analysis, present key insights and aggregated data clearly`

	// Variables to capture SQL and full results (outside agent context)
	var capturedSQL string
	var capturedResults []map[string]interface{}
	var capturedColumns []string

	// Create query tool for SQL execution
	// This tool returns only a SUMMARY to the agent, but captures full results for display
	queryTool := fantasy.NewAgentTool(
		"query",
		"Execute a SQL query against the DuckDB database. Returns a summary of results to avoid context limits.",
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

			// Capture SQL and full results for later display
			capturedSQL = input.SQL
			capturedResults = rows

			// Extract column names from first row
			if len(rows) > 0 {
				for col := range rows[0] {
					capturedColumns = append(capturedColumns, col)
				}
				sort.Strings(capturedColumns)
			}

			// Create summary for agent context (first 10 rows only)
			summary := summarizeQueryResults(rows, 10)

			return fantasy.NewTextResponse(summary), nil
		},
	)

	// Create schema tool for database introspection
	schemaTool := fantasy.NewAgentTool(
		"schema",
		"Get database schema information for all tables including user-imported tables",
		func(ctx context.Context, input agent.SchemaInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// Get list of all tables using DuckDB's SHOW ALL TABLES
			tablesQuery := "SHOW ALL TABLES"
			tableRows, err := h.DB.ExecuteQuery(tablesQuery)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get table list: %v", err)), nil
			}

			// Extract table names from SHOW ALL TABLES result
			var tables []string
			for _, row := range tableRows {
				if tableName, ok := row["table_name"].(string); ok {
					// Skip internal/system tables
					if tableName != "fts_main_directory" && tableName != "fts_main_directory_docs" && tableName != "fts_main_directory_config" {
						tables = append(tables, tableName)
					}
				} else if name, ok := row["name"].(string); ok {
					// Some versions use 'name' instead of 'table_name'
					if name != "fts_main_directory" && name != "fts_main_directory_docs" && name != "fts_main_directory_config" {
						tables = append(tables, name)
					}
				}
			}

			type SchemaOutput struct {
				TableName   string `json:"table_name"`
				TableComment string `json:"table_comment,omitempty"`
				Columns   []struct {
					Name    string `json:"name"`
					Type    string `json:"type"`
					Comment string `json:"comment,omitempty"`
				} `json:"columns"`
			}
			schemas := make([]SchemaOutput, 0, len(tables))

			for _, tableName := range tables {
				// Get column info from duckdb_columns() which includes comments
				colQuery := fmt.Sprintf("SELECT column_name, data_type, comment FROM duckdb_columns() WHERE table_name = '%s' ORDER BY column_index", tableName)
				rows, err := h.DB.ExecuteQuery(colQuery)
				if err != nil {
					continue
				}

				schema := SchemaOutput{
					TableName: tableName,
					Columns:   make([]struct{ Name string `json:"name"`; Type string `json:"type"`; Comment string `json:"comment,omitempty"` }, 0),
				}

				// Get table comment from duckdb_tables()
				tableCommentQuery := fmt.Sprintf("SELECT comment FROM duckdb_tables() WHERE table_name = '%s'", tableName)
				if commentRows, err := h.DB.ExecuteQuery(tableCommentQuery); err == nil && len(commentRows) > 0 {
					if comment, ok := commentRows[0]["comment"].(string); ok && comment != "" {
						schema.TableComment = comment
					}
				}

				// Get column details and comments
				for _, row := range rows {
					name, _ := row["column_name"].(string)
					colType, _ := row["data_type"].(string)
					comment, _ := row["comment"].(string)

					col := struct {
						Name    string `json:"name"`
						Type    string `json:"type"`
						Comment string `json:"comment,omitempty"`
					}{Name: name, Type: colType}

					if comment != "" {
						col.Comment = comment
					}

					schema.Columns = append(schema.Columns, col)
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
		return nil, fmt.Errorf("agent generation failed: %w", err)
	}

	// Extract response text
	responseText := result.Response.Content.Text()

	// Parse school IDs from the captured results (if NCESSCH column exists)
	var schoolIDs []string
	for _, row := range capturedResults {
		if ncessch, ok := row["NCESSCH"]; ok {
			if ncesschStr, ok := ncessch.(string); ok && len(ncesschStr) == 12 {
				schoolIDs = append(schoolIDs, ncesschStr)
			}
		}
	}

	// Return the complete result
	return &AIQueryResult{
		ResponseText: responseText,
		SQLQuery:     capturedSQL,
		TableData:    capturedResults,
		TableColumns: capturedColumns,
		SchoolIDs:    schoolIDs,
	}, nil
}

// AIQueryResult holds the result of an AI-powered database query
type AIQueryResult struct {
	ResponseText string                   // AI's natural language summary
	SQLQuery     string                   // The SQL query that was executed
	TableData    []map[string]interface{} // Full query results
	TableColumns []string                 // Column names from the query
	SchoolIDs    []string                 // Extracted school IDs (if applicable)
}

// summarizeQueryResults creates a concise summary of query results for agent context
// This avoids filling the context window with large result sets
func summarizeQueryResults(rows []map[string]interface{}, maxRows int) string {
	if len(rows) == 0 {
		return "Query returned 0 rows."
	}

	// Get column names from first row
	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	// Build summary
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Query returned %d rows.\n\n", len(rows)))

	// Include first N rows
	displayRows := maxRows
	if len(rows) < displayRows {
		displayRows = len(rows)
	}

	summary.WriteString(fmt.Sprintf("First %d rows:\n", displayRows))
	summary.WriteString(strings.Join(columns, " | ") + "\n")
	summary.WriteString(strings.Repeat("-", len(columns)*20) + "\n")

	for i := 0; i < displayRows; i++ {
		row := rows[i]
		var values []string
		for _, col := range columns {
			val := row[col]
			if val == nil {
				values = append(values, "NULL")
			} else {
				values = append(values, fmt.Sprintf("%v", val))
			}
		}
		summary.WriteString(strings.Join(values, " | ") + "\n")
	}

	if len(rows) > displayRows {
		summary.WriteString(fmt.Sprintf("\n... and %d more rows not shown ...\n", len(rows)-displayRows))
	}

	return summary.String()
}

// ImportPage renders the data import page
func (h *WebHandler) ImportPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Import Data",
	}

	if err := h.templates.ExecuteTemplate(w, "import.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ImportResult holds the result of a CSV import operation
type ImportResult struct {
	TableName         string
	RowCount          int64
	ColumnCount       int
	FileSize          string
	DataMetrics       []ColumnMetric
	AIDescription     string
	ProcessingStages  []ProcessingStage
	Error             string
}

// ColumnMetric holds metrics about a column from SUMMARIZE
type ColumnMetric struct {
	ColumnName     string
	ColumnType     string
	Min            string
	Max            string
	Unique         string
	NullPercentage string
}

// ProcessingStage tracks each stage of the import process
type ProcessingStage struct {
	Stage    string
	Message  string
	Duration string
}

// ImportCSV handles CSV file upload and import
func (h *WebHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	result := &ImportResult{
		ProcessingStages: make([]ProcessingStage, 0),
	}

	// Stage 1: Parse multipart form
	stageStart := time.Now()
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		result.Error = fmt.Sprintf("Failed to parse form: %v", err)
		h.renderImportResult(w, result)
		return
	}
	result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
		Stage:    "Parse Form",
		Message:  "Form data parsed successfully",
		Duration: time.Since(stageStart).String(),
	})

	// Get form values
	tableName := r.FormValue("table_name")
	description := r.FormValue("description")

	if tableName == "" || description == "" {
		result.Error = "Table name and description are required"
		h.renderImportResult(w, result)
		return
	}

	result.TableName = tableName

	// Stage 2: Get uploaded file
	stageStart = time.Now()
	file, header, err := r.FormFile("csv_file")
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get uploaded file: %v", err)
		h.renderImportResult(w, result)
		return
	}
	defer file.Close()

	result.FileSize = formatFileSize(header.Size)
	result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
		Stage:    "File Upload",
		Message:  fmt.Sprintf("Received file: %s (%s)", header.Filename, result.FileSize),
		Duration: time.Since(stageStart).String(),
	})

	// Stage 3: Create user_data directory and save file
	stageStart = time.Now()
	userDataDir := filepath.Join(h.DB.dataDir, "user_data")
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		result.Error = fmt.Sprintf("Failed to create user_data directory: %v", err)
		h.renderImportResult(w, result)
		return
	}

	// Save file with table name
	filePath := filepath.Join(userDataDir, fmt.Sprintf("%s.csv", tableName))
	outFile, err := os.Create(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create file: %v", err)
		h.renderImportResult(w, result)
		return
	}
	defer outFile.Close()

	if _, err := outFile.ReadFrom(file); err != nil {
		result.Error = fmt.Sprintf("Failed to save file: %v", err)
		h.renderImportResult(w, result)
		return
	}

	result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
		Stage:    "Save File",
		Message:  fmt.Sprintf("File saved to %s", filePath),
		Duration: time.Since(stageStart).String(),
	})

	// Stage 4: Run SUMMARIZE to analyze the data
	stageStart = time.Now()
	summarizeQuery := fmt.Sprintf("SUMMARIZE SELECT * FROM read_csv('%s', auto_detect=true)", filePath)
	summaryRows, err := h.DB.ExecuteQuery(summarizeQuery)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to analyze CSV: %v", err)
		h.renderImportResult(w, result)
		return
	}

	// Parse summary results into metrics
	result.DataMetrics = parseSummaryToMetrics(summaryRows)
	result.ColumnCount = len(result.DataMetrics)
	result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
		Stage:    "Analyze Data",
		Message:  fmt.Sprintf("Analyzed %d columns", result.ColumnCount),
		Duration: time.Since(stageStart).String(),
	})

	// Stage 5: Import data as new table
	stageStart = time.Now()
	createTableQuery := fmt.Sprintf(`
		CREATE TABLE %s AS
		SELECT * FROM read_csv('%s', auto_detect=true)
	`, tableName, filePath)

	if _, err := h.DB.ExecuteQuery(createTableQuery); err != nil {
		result.Error = fmt.Sprintf("Failed to create table: %v", err)
		h.renderImportResult(w, result)
		return
	}

	// Get row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
	countRows, err := h.DB.ExecuteQuery(countQuery)
	if err == nil && len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			result.RowCount = count
		}
	}

	result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
		Stage:    "Create Table",
		Message:  fmt.Sprintf("Table '%s' created with %d rows", tableName, result.RowCount),
		Duration: time.Since(stageStart).String(),
	})

	// Stage 6: Use AI to generate table and column descriptions
	stageStart = time.Now()
	aiDescription, columnComments, err := h.generateAIDescriptions(r.Context(), tableName, description, result.DataMetrics)
	if err != nil {
		log.Printf("Warning: Failed to generate AI descriptions: %v", err)
		result.AIDescription = "AI description generation failed"
	} else {
		result.AIDescription = aiDescription
		result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
			Stage:    "Generate Descriptions",
			Message:  "AI-generated table and column descriptions",
			Duration: time.Since(stageStart).String(),
		})

		// Stage 7: Add comments to table and columns
		stageStart = time.Now()
		if err := h.addTableComments(tableName, aiDescription, columnComments); err != nil {
			log.Printf("Warning: Failed to add table comments: %v", err)
		} else {
			result.ProcessingStages = append(result.ProcessingStages, ProcessingStage{
				Stage:    "Add Comments",
				Message:  fmt.Sprintf("Added comments to table and %d columns", len(columnComments)),
				Duration: time.Since(stageStart).String(),
			})
		}
	}

	h.renderImportResult(w, result)
}

// renderImportResult renders the import result partial
func (h *WebHandler) renderImportResult(w http.ResponseWriter, result *ImportResult) {
	if err := h.templates.ExecuteTemplate(w, "import_result.html", result); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// parseSummaryToMetrics converts SUMMARIZE output to ColumnMetric structs
func parseSummaryToMetrics(summaryRows []map[string]interface{}) []ColumnMetric {
	metrics := make([]ColumnMetric, 0, len(summaryRows))

	for _, row := range summaryRows {
		metric := ColumnMetric{
			ColumnName: fmt.Sprintf("%v", row["column_name"]),
			ColumnType: fmt.Sprintf("%v", row["column_type"]),
			Min:        formatValue(row["min"]),
			Max:        formatValue(row["max"]),
			Unique:     formatValue(row["approx_unique"]),
		}

		// Calculate null percentage if available
		if nullCount, ok := row["null_percentage"].(float64); ok {
			metric.NullPercentage = fmt.Sprintf("%.1f", nullCount)
		} else {
			metric.NullPercentage = "0"
		}

		metrics = append(metrics, metric)
	}

	return metrics
}

// formatValue safely formats interface{} values for display
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", val)
}

// formatFileSize converts bytes to human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// generateAIDescriptions uses Claude to generate descriptions for table and columns
func (h *WebHandler) generateAIDescriptions(ctx context.Context, tableName string, userDescription string, metrics []ColumnMetric) (string, map[string]string, error) {
	// Check if AI is available
	if h.AIScraper == nil {
		return "", nil, fmt.Errorf("AI not available")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Create Anthropic client
	client := anthropicsdk.NewClient(option.WithAPIKey(apiKey))

	// Build prompt with metrics
	metricsJSON, _ := json.MarshalIndent(metrics, "", "  ")
	prompt := fmt.Sprintf(`You are analyzing a newly imported CSV dataset. Based on the user's description and the data metrics, generate:
1. A concise table description (1-2 sentences) explaining what this table contains and how it should be used in queries
2. A brief comment for each column explaining what it contains

User's description: %s

Table name: %s

Column metrics:
%s

Respond in JSON format:
{
  "table_description": "...",
  "column_comments": {
    "column_name": "comment",
    ...
  }
}`, userDescription, tableName, string(metricsJSON))

	// Create the message parameters
	params := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.ModelClaudeHaiku4_5_20251001,
		MaxTokens: anthropicsdk.Int(2000),
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(prompt)),
		},
	}

	// Call the Messages API
	message, err := client.Messages.New(ctx, params)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract response text from content blocks
	if len(message.Content) == 0 {
		return "", nil, fmt.Errorf("no content in response")
	}

	var responseText string
	for _, block := range message.Content {
		if block.Type == anthropicsdk.ContentBlockTypeText {
			responseText += block.Text
		}
	}

	if responseText == "" {
		return "", nil, fmt.Errorf("no text content in response")
	}

	// Extract JSON from response (may be wrapped in markdown code blocks)
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return "", nil, fmt.Errorf("no JSON found in response")
	}
	jsonStr := responseText[jsonStart : jsonEnd+1]

	var aiResponse struct {
		TableDescription string            `json:"table_description"`
		ColumnComments   map[string]string `json:"column_comments"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &aiResponse); err != nil {
		return "", nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return aiResponse.TableDescription, aiResponse.ColumnComments, nil
}

// addTableComments adds COMMENT ON statements for table and columns
func (h *WebHandler) addTableComments(tableName string, tableComment string, columnComments map[string]string) error {
	// Add table comment
	tableCommentQuery := fmt.Sprintf("COMMENT ON TABLE %s IS '%s'",
		tableName,
		strings.ReplaceAll(tableComment, "'", "''"))

	if _, err := h.DB.ExecuteQuery(tableCommentQuery); err != nil {
		return fmt.Errorf("failed to add table comment: %w", err)
	}

	// Add column comments
	for col, comment := range columnComments {
		colCommentQuery := fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s'",
			tableName,
			col,
			strings.ReplaceAll(comment, "'", "''"))

		if _, err := h.DB.ExecuteQuery(colCommentQuery); err != nil {
			log.Printf("Warning: Failed to add comment for column %s: %v", col, err)
		}
	}

	return nil
}
