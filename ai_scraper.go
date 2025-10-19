package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
	NCESSCH          string    `json:"ncessch"`
	SchoolName       string    `json:"school_name"`
	ExtractedAt      time.Time `json:"extracted_at"`
	SourceURL        string    `json:"source_url"`

	// Markdown content from AI extraction
	MarkdownContent  string    `json:"markdown_content"`

	// Legacy structured fields (kept for backward compatibility with cached data)
	Principal        string   `json:"principal,omitempty"`
	VicePrincipals   []string `json:"vice_principals,omitempty"`
	Mascot           string   `json:"mascot,omitempty"`
	SchoolColors     []string `json:"school_colors,omitempty"`
	Founded          string   `json:"founded,omitempty"`

	// Staff Contact Information
	StaffContacts    []StaffContact `json:"staff_contacts,omitempty"`
	MainOfficeEmail  string         `json:"main_office_email,omitempty"`
	MainOfficePhone  string         `json:"main_office_phone,omitempty"`

	// Academic Programs
	APCourses        []string `json:"ap_courses,omitempty"`
	Honors           []string `json:"honors,omitempty"`
	SpecialPrograms  []string `json:"special_programs,omitempty"`
	Languages        []string `json:"languages,omitempty"`

	// Activities & Sports
	Sports           []string `json:"sports,omitempty"`
	Clubs            []string `json:"clubs,omitempty"`
	Arts             []string `json:"arts,omitempty"`

	// Facilities
	Facilities       []string `json:"facilities,omitempty"`

	// Schedule & Calendar
	BellSchedule     string   `json:"bell_schedule,omitempty"`
	SchoolHours      string   `json:"school_hours,omitempty"`

	// Achievements
	Achievements     []string `json:"achievements,omitempty"`
	Accreditations   []string `json:"accreditations,omitempty"`

	// Additional Info
	Mission          string   `json:"mission,omitempty"`
	Notes            string   `json:"notes,omitempty"`
}

// AIScraperService handles website scraping with Claude
type AIScraperService struct {
	client      *anthropic.Client
	cacheDir    string
	httpClient  *http.Client
}

// NewAIScraperService creates a new AI scraper service
func NewAIScraperService(apiKey, cacheDir string) (*AIScraperService, error) {
	if apiKey == "" {
		if logger != nil {
			logger.Error("AI scraper initialization failed: missing API key")
		}
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	if cacheDir == "" {
		cacheDir = ".school_cache"
	}

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		if logger != nil {
			logger.Error("Failed to create AI scraper cache directory", "error", err, "cache_dir", cacheDir)
		}
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if logger != nil {
		logger.Info("AI scraper service initialized", "cache_dir", cacheDir)
	}

	return &AIScraperService{
		client:   &client,
		cacheDir: cacheDir,
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
	defer resp.Body.Close()

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

	// Check cache first
	cached, err := s.loadFromCache(school.NCESSCH)
	if err == nil && cached != nil {
		// Return cached data if less than 30 days old
		if time.Since(cached.ExtractedAt) < 30*24*time.Hour {
			if logger != nil {
				logger.Info("Returning cached school data", "school_name", school.Name, "ncessch", school.NCESSCH, "cache_age_days", int(time.Since(cached.ExtractedAt).Hours()/24))
			}
			return cached, nil
		}
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

	// Save to cache
	if err := s.saveToCache(data); err != nil {
		// Don't fail if cache save fails, just log
		if logger != nil {
			logger.Warn("Failed to save school data to cache", "error", err, "school_name", school.Name, "ncessch", school.NCESSCH)
		}
	}

	return data, nil
}

// loadFromCache loads cached data for a school
func (s *AIScraperService) loadFromCache(ncessch string) (*EnhancedSchoolData, error) {
	filename := filepath.Join(s.cacheDir, ncessch+".json")

	data, err := os.ReadFile(filename)
	if err != nil {
		// Don't log "file not found" errors as they're expected for uncached schools
		if !os.IsNotExist(err) && logger != nil {
			logger.Warn("Failed to read cache file", "error", err, "ncessch", ncessch, "filename", filename)
		}
		return nil, err
	}

	var enhanced EnhancedSchoolData
	if err := json.Unmarshal(data, &enhanced); err != nil {
		if logger != nil {
			logger.Error("Failed to unmarshal cached school data", "error", err, "ncessch", ncessch, "filename", filename)
		}
		return nil, err
	}

	return &enhanced, nil
}

// saveToCache saves data to cache
func (s *AIScraperService) saveToCache(data *EnhancedSchoolData) error {
	filename := filepath.Join(s.cacheDir, data.NCESSCH+".json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		if logger != nil {
			logger.Error("Failed to marshal school data for caching", "error", err, "ncessch", data.NCESSCH, "school_name", data.SchoolName)
		}
		return err
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		if logger != nil {
			logger.Error("Failed to write cache file", "error", err, "ncessch", data.NCESSCH, "filename", filename)
		}
		return err
	}

	if logger != nil {
		logger.Info("Successfully cached school data", "ncessch", data.NCESSCH, "school_name", data.SchoolName, "filename", filename)
	}

	return nil
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
