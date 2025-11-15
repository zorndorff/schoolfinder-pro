package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// StaffContact represents contact information for a staff member
type StaffContact struct {
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Department string `json:"department,omitempty"`
}

// EnhancedSchoolData represents structured data extracted from school website
type EnhancedSchoolData struct {
	NCESSCH     string    `json:"ncessch"`
	SchoolName  string    `json:"school_name"`
	ExtractedAt time.Time `json:"extracted_at"`
	SourceURL   string    `json:"source_url"`

	// Markdown content from AI extraction
	MarkdownContent string `json:"markdown_content"`

	// Legacy structured fields (kept for backward compatibility with cached data)
	Principal      string   `json:"principal,omitempty"`
	VicePrincipals []string `json:"vice_principals,omitempty"`
	Mascot         string   `json:"mascot,omitempty"`
	SchoolColors   []string `json:"school_colors,omitempty"`
	Founded        string   `json:"founded,omitempty"`

	// Staff Contact Information
	StaffContacts   []StaffContact `json:"staff_contacts,omitempty"`
	MainOfficeEmail string         `json:"main_office_email,omitempty"`
	MainOfficePhone string         `json:"main_office_phone,omitempty"`

	// Academic Programs
	APCourses       []string `json:"ap_courses,omitempty"`
	Honors          []string `json:"honors,omitempty"`
	SpecialPrograms []string `json:"special_programs,omitempty"`
	Languages       []string `json:"languages,omitempty"`

	// Activities & Sports
	Sports []string `json:"sports,omitempty"`
	Clubs  []string `json:"clubs,omitempty"`
	Arts   []string `json:"arts,omitempty"`

	// Facilities
	Facilities []string `json:"facilities,omitempty"`

	// Schedule & Calendar
	BellSchedule string `json:"bell_schedule,omitempty"`
	SchoolHours  string `json:"school_hours,omitempty"`

	// Achievements
	Achievements   []string `json:"achievements,omitempty"`
	Accreditations []string `json:"accreditations,omitempty"`

	// Additional Info
	Mission string `json:"mission,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// AIScraperService handles website scraping with Claude
type AIScraperService struct {
	client         *anthropic.Client
	db             *DB
	cacheTTL       time.Duration
	httpClient     *http.Client
	maxSQLRetries  int // Maximum attempts to correct failed SQL queries
}

// NewAIScraperService creates a new AI scraper service
func NewAIScraperService(apiKey string, db *DB) (*AIScraperService, error) {
	if apiKey == "" {
		if logger != nil {
			logger.Error("AI scraper initialization failed: missing API key")
		}
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// Get max retries from environment or use default
	maxRetries := 3
	if retryStr := os.Getenv("AI_SQL_MAX_RETRIES"); retryStr != "" {
		if r, err := fmt.Sscanf(retryStr, "%d", &maxRetries); err == nil && r == 1 {
			if maxRetries < 0 {
				maxRetries = 0
			} else if maxRetries > 5 {
				maxRetries = 5 // Cap at 5 to avoid excessive API calls
			}
		}
	}

	if logger != nil {
		logger.Info("AI scraper service initialized with database caching", "cache_ttl_days", 30, "max_sql_retries", maxRetries)
	}

	return &AIScraperService{
		client:        &client,
		db:            db,
		cacheTTL:      30 * 24 * time.Hour, // 30 days
		maxSQLRetries: maxRetries,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// FetchWebsiteContent fetches the HTML content from a URL
func (s *AIScraperService) FetchWebsiteContent(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to create HTTP request for website fetch", "error", err, "url", url)
		}
		return "", err
	}

	req.Header.Set("User-Agent", "SchoolFinder/2.0 (Educational Research Tool; Contact Info Collector; +https://github.com/anthropics/claude-code)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if logger != nil {
			logger.Error("HTTP request failed for website fetch", "error", err, "url", url)
		}
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if logger != nil {
			logger.Error("HTTP request returned non-OK status", "status_code", resp.StatusCode, "url", url)
		}
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to read response body", "error", err, "url", url)
		}
		return "", err
	}

	// Limit content size (Claude has token limits)
	content := string(body)
	if len(content) > 100000 {
		content = content[:100000]
	}

	return content, nil
}

// ExtractSchoolDataWithWebSearch uses Claude 4.5 Haiku with web search to find staff contact information
func (s *AIScraperService) ExtractSchoolDataWithWebSearch(ctx context.Context, school *School) (*EnhancedSchoolData, error) {
	// Build context about the school
	address := ""
	if school.Street1.Valid && school.Street1.String != "" {
		address = school.Street1.String
	}
	if address == "" {
		address = fmt.Sprintf("%s, %s %s", school.City, school.State, school.ZipString())
	} else {
		address = fmt.Sprintf("%s, %s, %s %s", address, school.City, school.State, school.ZipString())
	}

	websiteURL := school.Website.String
	if !strings.HasPrefix(websiteURL, "http://") && !strings.HasPrefix(websiteURL, "https://") {
		websiteURL = "https://" + websiteURL
	}

	// Construct the user message
	content := fmt.Sprintf(`Your PRIMARY OBJECTIVE is to find administrative staff contact information for %s, located at %s. Website: %s.

**CRITICAL REQUIREMENT: You must locate and extract staff contact information with emails and phone numbers.**

Use web search to thoroughly investigate this school. Search multiple sources including:
- The school's official website staff directory or staff pages
- "About" or "Administration" pages
- Contact pages
- School district directory pages
- Any other relevant sources

**Staff Contact Information (PRIORITY #1):**
You MUST find and include:
- Principal: Full name, email address, phone number
- Vice Principals: Names, email addresses, phone numbers
- Key administrative staff: Counselors, department heads, secretaries
  - Include: Name, title, email, phone, department for each person
- Main office: Phone number and email address

Continue searching until you have located administrative staff contact information. Check multiple pages and sources.
If the main website doesn't have a staff directory, search for "[school name] staff directory" or "[school name] principal contact".

**Additional School Information (Secondary):**
Once you have the staff contacts, also gather:
- Mascot and school colors
- Academic programs (AP courses, honors, special programs, languages)
- Activities (sports, clubs, arts)
- Facilities
- School hours and schedule
- Mission statement
- Achievements and accreditations

**Output Format:**
Present your findings in clean, well-formatted markdown with:
- Clear section headers
- Bullet points or tables for staff listings
- Email addresses and phone numbers prominently displayed

If you cannot find staff contact information after thorough searching, explicitly state what you searched and why the information may not be publicly available.`,
		school.Name, address, websiteURL)

	webSearchTool := &anthropic.WebSearchTool20250305Param{}

	tools := []anthropic.ToolUnionParam{{
		OfWebSearchTool20250305: webSearchTool,
	}}

	// Create the message parameters
	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5_20251001,
		MaxTokens: 8000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(content)),
		},
		Tools: tools,
	}

	// Call the Messages API
	message, err := s.client.Messages.New(ctx, params)
	if err != nil {
		if logger != nil {
			logger.Error("Claude API call failed", "error", err, "school_name", school.Name, "ncessch", school.NCESSCH, "model", "haiku-4.5")
		}
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// Extract the response text from all content blocks
	if len(message.Content) == 0 {
		if logger != nil {
			logger.Error("Empty response from Claude API", "school_name", school.Name, "ncessch", school.NCESSCH)
		}
		return nil, fmt.Errorf("empty response from Claude")
	}

	responseText := ""
	for _, block := range message.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			responseText += textBlock.Text
		}
	}

	if responseText == "" {
		if logger != nil {
			logger.Error("No text content in Claude API response", "school_name", school.Name, "ncessch", school.NCESSCH, "content_blocks", len(message.Content))
		}
		return nil, fmt.Errorf("no text response from Claude")
	}

	if logger != nil {
		logger.Info("Successfully extracted school data with Claude", "school_name", school.Name, "ncessch", school.NCESSCH, slog.Int("response_length", len(responseText)))
	}

	// Store the markdown content directly
	data := &EnhancedSchoolData{
		MarkdownContent: responseText,
	}

	return data, nil
}

// ScrapeSchoolWebsite performs the full scraping workflow using web search
func (s *AIScraperService) ScrapeSchoolWebsite(ctx context.Context, school *School) (*EnhancedSchoolData, error) {
	// Check if website is available
	if !school.Website.Valid || school.Website.String == "" {
		if logger != nil {
			logger.Warn("Cannot scrape school: no website available", "school_name", school.Name, "ncessch", school.NCESSCH)
		}
		return nil, fmt.Errorf("no website available for this school")
	}

	websiteURL := school.Website.String
	if !strings.HasPrefix(websiteURL, "http://") && !strings.HasPrefix(websiteURL, "https://") {
		websiteURL = "https://" + websiteURL
	}

	// Check database cache first
	cached, err := s.loadFromCache(school.NCESSCH)
	if err == nil && cached != nil {
		if logger != nil {
			logger.Info("Returning cached school data from database", "school_name", school.Name, "ncessch", school.NCESSCH, "cache_age_days", int(time.Since(cached.ExtractedAt).Hours()/24))
		}
		return cached, nil
	}

	if logger != nil {
		logger.Info("Scraping school website", "school_name", school.Name, "ncessch", school.NCESSCH, "website", websiteURL)
	}

	// Extract data using Claude 4.5 Haiku with web search
	data, err := s.ExtractSchoolDataWithWebSearch(ctx, school)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to extract school data", "error", err, "school_name", school.Name, "ncessch", school.NCESSCH, "website", websiteURL)
		}
		return nil, err
	}

	// Fill in metadata
	data.NCESSCH = school.NCESSCH
	data.SchoolName = school.Name
	data.ExtractedAt = time.Now()
	data.SourceURL = websiteURL

	// Save to database cache
	if err := s.saveToCache(data); err != nil {
		// Don't fail if cache save fails, just log
		if logger != nil {
			logger.Warn("Failed to save school data to database cache", "error", err, "school_name", school.Name, "ncessch", school.NCESSCH)
		}
	}

	return data, nil
}

// loadFromCache loads cached data from the database
func (s *AIScraperService) loadFromCache(ncessch string) (*EnhancedSchoolData, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	schoolName, sourceURL, markdownContent, legacyData, extractedAt, err := s.db.LoadAIScraperCache(ncessch, s.cacheTTL)
	if err != nil {
		return nil, err
	}

	data := &EnhancedSchoolData{
		NCESSCH:         ncessch,
		SchoolName:      schoolName,
		SourceURL:       sourceURL,
		MarkdownContent: markdownContent,
		ExtractedAt:     extractedAt,
	}

	// Unmarshal legacy data if present
	if len(legacyData) > 0 {
		var legacy EnhancedSchoolData
		if err := json.Unmarshal(legacyData, &legacy); err == nil {
			// Copy legacy fields
			data.Principal = legacy.Principal
			data.VicePrincipals = legacy.VicePrincipals
			data.Mascot = legacy.Mascot
			data.SchoolColors = legacy.SchoolColors
			data.Founded = legacy.Founded
			data.StaffContacts = legacy.StaffContacts
			data.MainOfficeEmail = legacy.MainOfficeEmail
			data.MainOfficePhone = legacy.MainOfficePhone
			data.APCourses = legacy.APCourses
			data.Honors = legacy.Honors
			data.SpecialPrograms = legacy.SpecialPrograms
			data.Languages = legacy.Languages
			data.Sports = legacy.Sports
			data.Clubs = legacy.Clubs
			data.Arts = legacy.Arts
			data.Facilities = legacy.Facilities
			data.BellSchedule = legacy.BellSchedule
			data.SchoolHours = legacy.SchoolHours
			data.Achievements = legacy.Achievements
			data.Accreditations = legacy.Accreditations
			data.Mission = legacy.Mission
			data.Notes = legacy.Notes
		}
	}

	return data, nil
}

// saveToCache saves data to the database cache
func (s *AIScraperService) saveToCache(data *EnhancedSchoolData) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// Marshal legacy data (all structured fields) as JSON
	legacyData := map[string]interface{}{
		"principal":         data.Principal,
		"vice_principals":   data.VicePrincipals,
		"mascot":            data.Mascot,
		"school_colors":     data.SchoolColors,
		"founded":           data.Founded,
		"staff_contacts":    data.StaffContacts,
		"main_office_email": data.MainOfficeEmail,
		"main_office_phone": data.MainOfficePhone,
		"ap_courses":        data.APCourses,
		"honors":            data.Honors,
		"special_programs":  data.SpecialPrograms,
		"languages":         data.Languages,
		"sports":            data.Sports,
		"clubs":             data.Clubs,
		"arts":              data.Arts,
		"facilities":        data.Facilities,
		"bell_schedule":     data.BellSchedule,
		"school_hours":      data.SchoolHours,
		"achievements":      data.Achievements,
		"accreditations":    data.Accreditations,
		"mission":           data.Mission,
		"notes":             data.Notes,
	}

	legacyJSON, err := json.Marshal(legacyData)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to marshal legacy data", "error", err, "ncessch", data.NCESSCH)
		}
		legacyJSON = nil // Continue without legacy data
	}

	return s.db.SaveAIScraperCache(
		data.NCESSCH,
		data.SchoolName,
		data.SourceURL,
		data.MarkdownContent,
		legacyJSON,
		data.ExtractedAt,
	)
}

// FormatEnhancedData formats the enhanced data for display
func FormatEnhancedData(data *EnhancedSchoolData) string {
	var b strings.Builder

	b.WriteString("📚 AI-Extracted School Information\n")
	b.WriteString(fmt.Sprintf("Source: %s\n", data.SourceURL))
	b.WriteString(fmt.Sprintf("Extracted: %s\n", data.ExtractedAt.Format("2006-01-02 15:04")))
	b.WriteString("\n")

	// If we have markdown content, display it
	if data.MarkdownContent != "" {
		b.WriteString(data.MarkdownContent)
		return b.String()
	}

	// Otherwise, fall back to legacy structured format (for cached data)
	// Staff Contact Information Section
	if len(data.StaffContacts) > 0 || data.MainOfficeEmail != "" || data.MainOfficePhone != "" {
		b.WriteString("📞 Contact Information:\n")

		if data.MainOfficePhone != "" {
			b.WriteString(fmt.Sprintf("  Main Office: %s\n", data.MainOfficePhone))
		}
		if data.MainOfficeEmail != "" {
			b.WriteString(fmt.Sprintf("  Email: %s\n", data.MainOfficeEmail))
		}

		if len(data.StaffContacts) > 0 {
			b.WriteString("\n  Staff Directory:\n")
			for _, contact := range data.StaffContacts {
				b.WriteString(fmt.Sprintf("    • %s", contact.Name))
				if contact.Title != "" {
					b.WriteString(fmt.Sprintf(" - %s", contact.Title))
				}
				if contact.Department != "" {
					b.WriteString(fmt.Sprintf(" (%s)", contact.Department))
				}
				b.WriteString("\n")
				if contact.Email != "" {
					b.WriteString(fmt.Sprintf("      Email: %s\n", contact.Email))
				}
				if contact.Phone != "" {
					b.WriteString(fmt.Sprintf("      Phone: %s\n", contact.Phone))
				}
			}
		}
		b.WriteString("\n")
	}

	if data.Principal != "" {
		b.WriteString(fmt.Sprintf("Principal: %s\n", data.Principal))
	}

	if len(data.VicePrincipals) > 0 {
		b.WriteString(fmt.Sprintf("Vice Principals: %s\n", strings.Join(data.VicePrincipals, ", ")))
	}

	if data.Mascot != "" || len(data.SchoolColors) > 0 {
		b.WriteString("\nSchool Identity:\n")
		if data.Mascot != "" {
			b.WriteString(fmt.Sprintf("  Mascot: %s\n", data.Mascot))
		}
		if len(data.SchoolColors) > 0 {
			b.WriteString(fmt.Sprintf("  Colors: %s\n", strings.Join(data.SchoolColors, ", ")))
		}
	}

	if len(data.APCourses) > 0 {
		b.WriteString("\nAP Courses:\n")
		for _, course := range data.APCourses {
			b.WriteString(fmt.Sprintf("  • %s\n", course))
		}
	}

	if len(data.SpecialPrograms) > 0 {
		b.WriteString("\nSpecial Programs:\n")
		for _, prog := range data.SpecialPrograms {
			b.WriteString(fmt.Sprintf("  • %s\n", prog))
		}
	}

	if len(data.Sports) > 0 {
		b.WriteString(fmt.Sprintf("\nSports: %s\n", strings.Join(data.Sports, ", ")))
	}

	if len(data.Clubs) > 0 && len(data.Clubs) <= 10 {
		b.WriteString(fmt.Sprintf("\nClubs: %s\n", strings.Join(data.Clubs, ", ")))
	} else if len(data.Clubs) > 10 {
		b.WriteString(fmt.Sprintf("\nClubs: %d clubs available\n", len(data.Clubs)))
	}

	if data.SchoolHours != "" {
		b.WriteString(fmt.Sprintf("\nSchool Hours: %s\n", data.SchoolHours))
	}

	if data.Mission != "" {
		b.WriteString(fmt.Sprintf("\nMission: %s\n", data.Mission))
	}

	if len(data.Achievements) > 0 {
		b.WriteString("\nAchievements:\n")
		for _, ach := range data.Achievements {
			b.WriteString(fmt.Sprintf("  • %s\n", ach))
		}
	}

	return b.String()
}

// sqlQueryResult holds the parsed result from Claude's SQL generation
type sqlQueryResult struct {
	QueryType   string `json:"query_type"`   // "search" or "analysis"
	Explanation string `json:"explanation"`
	SQLQuery    string `json:"sql_query"`    // Full SQL query
	Analysis    string `json:"analysis"`     // Additional analysis text (optional)
}

// generateSQLFromClaude calls Claude to generate SQL based on user query and optional error context
func (s *AIScraperService) generateSQLFromClaude(ctx context.Context, query string, previousSQL string, sqlError string, attempt int) (*sqlQueryResult, error) {
	// Build the base prompt
	promptBase := `You are an AI data analyst helping users explore and analyze a database of 102,274 schools from the NCES Common Core of Data (CCD).

**Database Schema:**

Tables:
1. **directory** - Main school information (102K rows)
   - NCESSCH (Primary Key), SCH_NAME, ST (state code), STATENAME, MCITY (city)
   - LEA_NAME (district), LEAID, SCH_TYPE_TEXT, LEVEL
   - GSLO, GSHI (grade range), CHARTER_TEXT, PHONE, WEBSITE
   - MSTREET1, MZIP, SCHOOL_YEAR

2. **enrollment** - Student counts (11M rows)
   - NCESSCH (FK), STUDENT_COUNT
   - TOTAL_INDICATOR (use = 'Education Unit Total' for totals)

3. **teachers** - Teacher FTE counts (100K rows)
   - NCESSCH (FK), TEACHERS (float)

**User Query:** "%s"

**Task:** Analyze the query type and generate appropriate SQL.

**Query Types:**
1. **search** - Find specific schools → return NCESSCH column
2. **analysis** - Statistics/aggregations → return analysis columns

**Response Format (JSON only):**

For SEARCH (returns school list):
{
  "query_type": "search",
  "explanation": "What you're searching for",
  "sql_query": "SELECT d.NCESSCH FROM directory d LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total' LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH WHERE [conditions] LIMIT 200"
}

For ANALYSIS (returns aggregated data):
{
  "query_type": "analysis",
  "explanation": "What analysis you're performing",
  "sql_query": "SELECT [columns], COUNT(*), AVG() FROM directory d [JOINS] WHERE [conditions] GROUP BY [columns] ORDER BY [columns]"
}

**SQL Guidelines:**
- JOIN on NCESSCH
- For enrollment: LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total'
- Student-teacher ratio: CAST(e.STUDENT_COUNT AS FLOAT) / t.TEACHERS
- Use ST (not STATENAME) for state filtering
- LIMIT 200 for searches
- Use proper aggregation functions (COUNT, AVG, SUM, MIN, MAX)
- Database engine is DuckDB (PostgreSQL-compatible syntax)

**Examples:**

"Find charter high schools in California"
→ {"query_type": "search", "explanation": "Searching for charter high schools in CA", "sql_query": "SELECT d.NCESSCH FROM directory d WHERE d.ST = 'CA' AND d.LEVEL = 'High' AND d.CHARTER_TEXT LIKE '%%Yes%%' LIMIT 200"}

"Average enrollment by state"
→ {"query_type": "analysis", "explanation": "Computing average enrollment grouped by state", "sql_query": "SELECT d.ST, d.STATENAME, AVG(e.STUDENT_COUNT) as avg_enrollment, COUNT(DISTINCT d.NCESSCH) as school_count FROM directory d LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total' GROUP BY d.ST, d.STATENAME ORDER BY avg_enrollment DESC"}

"Top 10 schools by student-teacher ratio"
→ {"query_type": "search", "explanation": "Finding schools with best student-teacher ratios", "sql_query": "SELECT d.NCESSCH FROM directory d INNER JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total' INNER JOIN teachers t ON d.NCESSCH = t.NCESSCH WHERE t.TEACHERS > 0 ORDER BY CAST(e.STUDENT_COUNT AS FLOAT) / t.TEACHERS LIMIT 10"}
`

	// Add error correction context if this is a retry
	var prompt string
	if sqlError != "" && previousSQL != "" {
		prompt = fmt.Sprintf(`%s

**IMPORTANT - SQL ERROR CORRECTION (Attempt %d):**

Your previous SQL query failed with an error. Please analyze the error and generate a corrected query.

Previous SQL Query:
%s

Error Message:
%s

Please fix the SQL query to resolve this error. Common issues:
- Column names must match the schema exactly (check capitalization)
- Ensure proper JOIN conditions
- Verify aggregate functions are used correctly with GROUP BY
- Check for syntax errors

Return ONLY the corrected JSON with the fixed sql_query field.`, promptBase, attempt, previousSQL, sqlError)
	} else {
		prompt = promptBase + "\n\nReturn ONLY JSON, no other text."
	}

	prompt = fmt.Sprintf(prompt, query)

	// Call Claude API
	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5_20251001,
		MaxTokens: 4000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}

	message, err := s.client.Messages.New(ctx, params)
	if err != nil {
		if logger != nil {
			logger.Error("Claude API call failed for SQL generation", "error", err, "query", query, "attempt", attempt)
		}
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// Extract response
	if len(message.Content) == 0 {
		if logger != nil {
			logger.Error("Empty response from Claude for SQL generation", "query", query, "attempt", attempt)
		}
		return nil, fmt.Errorf("empty response from Claude")
	}

	responseText := ""
	for _, block := range message.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			responseText += textBlock.Text
		}
	}

	if responseText == "" {
		if logger != nil {
			logger.Error("No text content in Claude response for SQL generation", "query", query, "attempt", attempt)
		}
		return nil, fmt.Errorf("no text response from Claude")
	}

	// Parse JSON response
	var result sqlQueryResult

	// Try to extract JSON from response (it might be wrapped in markdown)
	jsonStr := responseText
	if strings.Contains(responseText, "```json") {
		start := strings.Index(responseText, "```json") + 7
		end := strings.Index(responseText[start:], "```")
		if end > 0 {
			jsonStr = responseText[start : start+end]
		}
	} else if strings.Contains(responseText, "```") {
		start := strings.Index(responseText, "```") + 3
		end := strings.Index(responseText[start:], "```")
		if end > 0 {
			jsonStr = responseText[start : start+end]
		}
	}

	jsonStr = strings.TrimSpace(jsonStr)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		if logger != nil {
			logger.Error("Failed to parse Claude response as JSON for SQL generation",
				"error", err,
				"response_preview", truncateString(responseText, 200),
				"attempt", attempt)
		}
		return nil, fmt.Errorf("failed to parse SQL response as JSON: %w", err)
	}

	if result.SQLQuery == "" {
		if logger != nil {
			logger.Warn("Claude generated empty SQL query", "query", query, "attempt", attempt)
		}
		return nil, fmt.Errorf("Claude generated empty SQL query")
	}

	if logger != nil {
		logger.Info("Successfully generated SQL from Claude",
			"query", query,
			"query_type", result.QueryType,
			"attempt", attempt)
	}

	return &result, nil
}

// QuerySchoolDatabase uses Claude to generate and execute SQL queries against the school database
// It implements retry logic with self-correction for failed SQL queries
func (s *AIScraperService) QuerySchoolDatabase(ctx context.Context, db *DB, query string) (string, []string, error) {
	if db == nil {
		return "", nil, fmt.Errorf("database not available")
	}

	var lastError error
	var previousSQL string

	// Retry loop with self-correction
	for attempt := 1; attempt <= s.maxSQLRetries; attempt++ {
		// Generate SQL using Claude
		var sqlResult *sqlQueryResult
		var err error

		if attempt == 1 {
			// First attempt: generate SQL from scratch
			if logger != nil {
				logger.Info("Generating SQL for database query", "query", query, "attempt", attempt)
			}
			sqlResult, err = s.generateSQLFromClaude(ctx, query, "", "", attempt)
		} else {
			// Retry attempt: include previous SQL and error for correction
			if logger != nil {
				logger.Info("Retrying SQL generation with error correction",
					"query", query,
					"attempt", attempt,
					"previous_error", lastError.Error())
			}
			sqlResult, err = s.generateSQLFromClaude(ctx, query, previousSQL, lastError.Error(), attempt)
		}

		if err != nil {
			// If we can't even generate SQL, no point retrying
			if logger != nil {
				logger.Error("Failed to generate SQL from Claude", "error", err, "query", query, "attempt", attempt)
			}
			return "", nil, fmt.Errorf("SQL generation failed: %w", err)
		}

		previousSQL = sqlResult.SQLQuery

		// Log the generated SQL for debugging
		if logger != nil {
			logger.Info("Executing AI-generated SQL",
				"query_type", sqlResult.QueryType,
				"sql_preview", truncateString(sqlResult.SQLQuery, 150),
				"attempt", attempt)
		}

		// Execute the SQL query
		rows, err := db.conn.Query(sqlResult.SQLQuery)
		if err != nil {
			lastError = err
			if logger != nil {
				logger.Warn("SQL execution failed, will retry if attempts remain",
					"error", err,
					"sql", sqlResult.SQLQuery,
					"attempt", attempt,
					"max_retries", s.maxSQLRetries)
			}

			// If this was the last attempt, return error
			if attempt >= s.maxSQLRetries {
				return "", nil, fmt.Errorf("SQL execution failed after %d attempts: %w\n\nLast SQL:\n%s",
					attempt, err, sqlResult.SQLQuery)
			}

			// Continue to next retry attempt
			continue
		}
		defer func() { _ = rows.Close() }()

		// Success! Process results based on query type
		var schoolIDs []string
		fullResponse := sqlResult.Explanation

		if sqlResult.QueryType == "search" {
			// For search queries, extract NCESSCH IDs
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					if logger != nil {
						logger.Warn("Failed to scan school ID from result", "error", err)
					}
					continue
				}
				schoolIDs = append(schoolIDs, id)
			}

			fullResponse = fmt.Sprintf("%s\n\nFound %d schools.", sqlResult.Explanation, len(schoolIDs))

			// Add retry note if we needed more than one attempt
			if attempt > 1 {
				fullResponse += fmt.Sprintf("\n\n*(Query succeeded on attempt %d after SQL self-correction)*", attempt)
			}

		} else if sqlResult.QueryType == "analysis" {
			// For analysis queries, format the results as a table/text
			columns, err := rows.Columns()
			if err != nil {
				if logger != nil {
					logger.Error("Failed to get result columns", "error", err)
				}
				return fmt.Sprintf("%s\n\nError reading results: %v", sqlResult.Explanation, err), []string{}, nil
			}

			// Build result table
			var analysisResults strings.Builder
			analysisResults.WriteString("\n\n**Results:**\n\n")

			// Read all rows
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			rowCount := 0
			for rows.Next() {
				if err := rows.Scan(valuePtrs...); err != nil {
					if logger != nil {
						logger.Warn("Failed to scan result row", "error", err, "row", rowCount)
					}
					continue
				}

				if rowCount == 0 {
					// Header
					analysisResults.WriteString("| ")
					for _, col := range columns {
						analysisResults.WriteString(fmt.Sprintf("%s | ", col))
					}
					analysisResults.WriteString("\n|")
					for range columns {
						analysisResults.WriteString("---|")
					}
					analysisResults.WriteString("\n")
				}

				// Data row
				analysisResults.WriteString("| ")
				for _, val := range values {
					if val == nil {
						analysisResults.WriteString("NULL | ")
					} else {
						// Format based on type
						switch v := val.(type) {
						case float64:
							analysisResults.WriteString(fmt.Sprintf("%.2f | ", v))
						case int64:
							analysisResults.WriteString(fmt.Sprintf("%d | ", v))
						default:
							analysisResults.WriteString(fmt.Sprintf("%v | ", v))
						}
					}
				}
				analysisResults.WriteString("\n")

				rowCount++
				if rowCount >= 50 { // Limit to 50 rows for display
					break
				}
			}

			if rowCount == 0 {
				analysisResults.WriteString("No results found.\n")
			} else if rowCount == 50 {
				analysisResults.WriteString("\n*(Showing first 50 results)*\n")
			}

			fullResponse = sqlResult.Explanation + analysisResults.String()

			// Add retry note if we needed more than one attempt
			if attempt > 1 {
				fullResponse += fmt.Sprintf("\n\n*(Query succeeded on attempt %d after SQL self-correction)*", attempt)
			}
		}

		if logger != nil {
			logger.Info("Successfully processed AI database query",
				"query", query,
				"query_type", sqlResult.QueryType,
				"school_results", len(schoolIDs),
				"attempt", attempt)
		}

		return fullResponse, schoolIDs, nil
	}

	// Should never reach here, but just in case
	return "", nil, fmt.Errorf("SQL query failed after %d attempts: %w", s.maxSQLRetries, lastError)
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
