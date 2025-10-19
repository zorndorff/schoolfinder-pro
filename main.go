package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxResults = 100
)

var logger *slog.Logger

// setupLogger creates and configures the application logger
func setupLogger(dataDir string) error {
	logPath := filepath.Join(dataDir, "err.log")

	// Create log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true, // Include file:line information
	})

	logger = slog.New(handler)
	logger.Info("Application started", "version", "1.0", "data_dir", dataDir)

	return nil
}

// renderMarkdown renders markdown content with glamour for beautiful display
func renderMarkdown(content string, width int) (string, error) {
	// Account for borders, padding, and glamour's internal gutter
	const glamourGutter = 2
	const borderWidth = 4 // 2 for border characters, 2 for padding

	renderWidth := width - borderWidth - glamourGutter
	if renderWidth < 40 {
		renderWidth = 40 // Minimum width for readable content
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return "", err
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return "", err
	}

	return rendered, nil
}

type view int

const (
	searchView view = iota
	detailView
	savePromptView
)

type model struct {
	db              *DB
	aiScraper       *AIScraperService
	naepClient      *NAEPClient
	currentView     view
	searchInput     textinput.Model
	saveInput       textinput.Model
	viewport        viewport.Model
	stateFilter     string
	schools         []School
	list            list.Model
	selectedItem    *School
	enhancedData    *EnhancedSchoolData
	naepData        *NAEPData
	width           int
	height          int
	err             error
	loading         bool
	scrapingAI      bool
	loadingNAEP     bool
	saveSuccess     string
	viewportReady   bool
	autoFetchNAEP   bool // Auto-fetch NAEP data when viewing details
}

type schoolItem struct {
	school School
}

func (i schoolItem) Title() string {
	return i.school.Name
}

func (i schoolItem) Description() string {
	teachers := i.school.TeachersString()
	enrollment := i.school.EnrollmentString()
	return fmt.Sprintf("%s, %s | %s | Students: %s | Teachers: %s | %s",
		i.school.City,
		i.school.State,
		i.school.District,
		enrollment,
		teachers,
		i.school.NCESSCH,
	)
}

func (i schoolItem) FilterValue() string {
	return i.school.Name + " " + i.school.City + " " + i.school.State + " " + i.school.District
}

type searchMsg struct {
	schools []School
	err     error
}

type aiScrapeMsg struct {
	data *EnhancedSchoolData
	err  error
}

type saveMsg struct {
	filename string
	err      error
}

type naepDataMsg struct {
	data *NAEPData
	err  error
}

func scrapeSchoolWebsite(scraper *AIScraperService, school *School) tea.Cmd {
	return func() tea.Msg {
		data, err := scraper.ScrapeSchoolWebsite(context.Background(), school)
		return aiScrapeMsg{data: data, err: err}
	}
}

func fetchNAEPData(client *NAEPClient, school *School) tea.Cmd {
	return func() tea.Msg {
		data, err := client.FetchNAEPData(school)
		return naepDataMsg{data: data, err: err}
	}
}

func openInEditor(data *EnhancedSchoolData, cacheDir string) tea.Cmd {
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Default editors
		if _, err := os.Stat("/usr/bin/nano"); err == nil {
			editor = "nano"
		} else if _, err := os.Stat("/usr/bin/vi"); err == nil {
			editor = "vi"
		} else {
			editor = "vim"
		}
	}

	// Get the file path
	filename := filepath.Join(cacheDir, data.NCESSCH+".json")
	ncessch := data.NCESSCH

	c := exec.Command(editor, filename)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return aiScrapeMsg{err: fmt.Errorf("editor error: %w", err)}
		}
		// Reload the data after editor closes
		reloaded, loadErr := loadEnhancedDataFromCache(ncessch, cacheDir)
		return aiScrapeMsg{data: reloaded, err: loadErr}
	})
}

func loadEnhancedDataFromCache(ncessch, cacheDir string) (*EnhancedSchoolData, error) {
	filename := filepath.Join(cacheDir, ncessch+".json")

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var enhanced EnhancedSchoolData
	if err := json.Unmarshal(data, &enhanced); err != nil {
		return nil, err
	}

	return &enhanced, nil
}

func saveSchoolData(school *School, enhanced *EnhancedSchoolData, naepData *NAEPData, filename string) tea.Cmd {
	return func() tea.Msg {
		// Create a combined data structure
		data := map[string]interface{}{
			"school": school,
		}

		if enhanced != nil {
			data["ai_extracted"] = enhanced
		}

		if naepData != nil {
			data["naep_data"] = naepData
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return saveMsg{err: fmt.Errorf("failed to marshal data: %w", err)}
		}

		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			return saveMsg{err: fmt.Errorf("failed to write file: %w", err)}
		}

		return saveMsg{filename: filename, err: nil}
	}
}

func searchSchools(db *DB, query, state string) tea.Cmd {
	return func() tea.Msg {
		schools, err := db.SearchSchools(query, state, maxResults)
		return searchMsg{schools: schools, err: err}
	}
}

func initialModel(db *DB, aiScraper *AIScraperService, naepClient *NAEPClient) model {
	ti := textinput.New()
	ti.Placeholder = "Search schools by name, city, district, address, or zip..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	si := textinput.New()
	si.Placeholder = "Enter filename (e.g., school_data.json)"
	si.CharLimit = 200
	si.Width = 60

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "School Finder"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle()

	// Check if auto-fetch NAEP is enabled (default: true)
	autoFetchNAEP := true
	if autoFetchEnv := os.Getenv("NAEP_AUTO_FETCH"); autoFetchEnv != "" {
		autoFetchNAEP = autoFetchEnv != "0" && autoFetchEnv != "false" && autoFetchEnv != "no"
	}

	return model{
		db:            db,
		aiScraper:     aiScraper,
		naepClient:    naepClient,
		currentView:   searchView,
		searchInput:   ti,
		saveInput:     si,
		viewport:      vp,
		list:          l,
		schools:       []School{},
		autoFetchNAEP: autoFetchNAEP,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-10)

		// Update viewport dimensions
		// Reserve 6 lines: 1 for newline, 1 for scroll indicator, up to 3 for status messages, 1 for help text
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
		m.viewportReady = true

		// Refresh viewport content if in detail view
		if m.currentView == detailView {
			m.updateDetailViewport()
		}

		return m, nil

	case tea.KeyMsg:
		if m.currentView == detailView {
			return m.handleDetailViewKeys(msg)
		} else if m.currentView == savePromptView {
			return m.handleSavePromptKeys(msg)
		}
		return m.handleSearchViewKeys(msg)

	case tea.MouseMsg:
		if m.currentView == detailView {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case searchMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			if logger != nil {
				logger.Error("School search failed", "error", msg.err, "query", m.searchInput.Value(), "state_filter", m.stateFilter)
			}
			return m, nil
		}

		m.schools = msg.schools
		items := make([]list.Item, len(msg.schools))
		for i, school := range msg.schools {
			items[i] = schoolItem{school: school}
		}
		m.list.SetItems(items)
		if logger != nil {
			logger.Info("Search completed", "results_count", len(msg.schools), "query", m.searchInput.Value())
		}
		return m, nil

	case aiScrapeMsg:
		m.scrapingAI = false
		if msg.err != nil {
			m.err = fmt.Errorf("AI scraping failed: %w", msg.err)
			if logger != nil && m.selectedItem != nil {
				logger.Error("AI scraping failed", "error", msg.err, "school_id", m.selectedItem.NCESSCH, "school_name", m.selectedItem.Name, "website", m.selectedItem.WebsiteString())
			}
			return m, nil
		}
		m.enhancedData = msg.data
		m.err = nil
		if m.currentView == detailView {
			m.updateDetailViewport()
		}
		if logger != nil && m.selectedItem != nil {
			logger.Info("AI scraping completed", "school_id", m.selectedItem.NCESSCH, "school_name", m.selectedItem.Name)
		}
		return m, nil

	case saveMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("save failed: %w", msg.err)
			m.currentView = detailView
			if logger != nil && m.selectedItem != nil {
				logger.Error("Failed to save school data", "error", msg.err, "school_id", m.selectedItem.NCESSCH, "filename", m.saveInput.Value())
			}
			return m, nil
		}
		m.saveSuccess = fmt.Sprintf("Saved to: %s", msg.filename)
		m.saveInput.SetValue("")
		m.currentView = detailView
		if logger != nil && m.selectedItem != nil {
			logger.Info("School data saved", "school_id", m.selectedItem.NCESSCH, "filename", msg.filename)
		}
		return m, nil

	case naepDataMsg:
		m.loadingNAEP = false
		if msg.err != nil {
			m.err = fmt.Errorf("NAEP fetch failed: %w", msg.err)
			if logger != nil && m.selectedItem != nil {
				logger.Error("NAEP data fetch failed", "error", msg.err, "school_id", m.selectedItem.NCESSCH, "school_name", m.selectedItem.Name, "state", m.selectedItem.State, "grade_low", m.selectedItem.GradeLow.String, "grade_high", m.selectedItem.GradeHigh.String)
			}
			return m, nil
		}
		m.naepData = msg.data
		m.err = nil
		if m.currentView == detailView {
			m.updateDetailViewport()
		}
		if logger != nil && m.selectedItem != nil {
			logger.Info("NAEP data fetched", "school_id", m.selectedItem.NCESSCH, "state", m.selectedItem.State, "state_scores", len(msg.data.StateScores), "district_scores", len(msg.data.DistrictScores))
		}
		return m, nil
	}

	if m.currentView == searchView {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)

		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		cmds = append(cmds, listCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleSearchViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyEnter:
		if m.searchInput.Focused() {
			// Perform search
			m.loading = true
			return m, searchSchools(m.db, m.searchInput.Value(), m.stateFilter)
		} else {
			// Select school from list
			if item, ok := m.list.SelectedItem().(schoolItem); ok {
				m.selectedItem = &item.school
				m.currentView = detailView
				m.viewport.GotoTop() // Reset scroll position
				m.updateDetailViewport() // Load content into viewport

				// Auto-fetch NAEP data if enabled
				if m.autoFetchNAEP && m.naepClient != nil && !m.loadingNAEP {
					m.loadingNAEP = true
					return m, fetchNAEPData(m.naepClient, m.selectedItem)
				}
			}
		}
		return m, nil

	case tea.KeyTab:
		if m.searchInput.Focused() {
			m.searchInput.Blur()
		} else {
			m.searchInput.Focus()
		}
		return m, textinput.Blink

	case tea.KeyCtrlS:
		// Cycle through states
		states := []string{"", "CA", "TX", "NY", "FL", "IL", "PA", "GA", "NJ", "NC", "OH"}
		found := false
		for i, s := range states {
			if s == m.stateFilter {
				m.stateFilter = states[(i+1)%len(states)]
				found = true
				break
			}
		}
		if !found {
			m.stateFilter = states[0]
		}
		if m.searchInput.Value() != "" {
			m.loading = true
			return m, searchSchools(m.db, m.searchInput.Value(), m.stateFilter)
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.searchInput.Focused() {
		m.searchInput, cmd = m.searchInput.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m model) handleDetailViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyEsc:
		if msg.Type == tea.KeyEsc {
			m.currentView = searchView
			m.selectedItem = nil
			m.enhancedData = nil
			m.naepData = nil
			m.err = nil
			m.saveSuccess = ""
			m.viewport.GotoTop()
			return m, nil
		}

	case tea.KeyCtrlC:
		m.currentView = searchView
		m.selectedItem = nil
		m.enhancedData = nil
		m.naepData = nil
		m.err = nil
		m.saveSuccess = ""
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyCtrlY:
		if m.selectedItem != nil {
			_ = clipboard.WriteAll(m.selectedItem.NCESSCH)
		}
		return m, nil

	case tea.KeyCtrlA:
		// AI scrape website
		if m.selectedItem != nil && !m.scrapingAI && m.aiScraper != nil {
			m.scrapingAI = true
			m.err = nil
			return m, scrapeSchoolWebsite(m.aiScraper, m.selectedItem)
		}
		return m, nil

	case tea.KeyCtrlE:
		// Open extracted data in editor
		if m.enhancedData != nil && m.aiScraper != nil {
			return m, openInEditor(m.enhancedData, m.aiScraper.cacheDir)
		}
		return m, nil

	case tea.KeyCtrlW:
		// Save school data to file
		if m.selectedItem != nil {
			m.currentView = savePromptView
			m.saveInput.Focus()
			m.err = nil
			m.saveSuccess = ""
			// Pre-fill with school name
			defaultName := strings.ReplaceAll(strings.ToLower(m.selectedItem.Name), " ", "_") + ".json"
			m.saveInput.SetValue(defaultName)
			return m, textinput.Blink
		}
		return m, nil

	case tea.KeyCtrlN:
		// Fetch NAEP data
		if m.selectedItem != nil && !m.loadingNAEP && m.naepClient != nil {
			m.loadingNAEP = true
			m.err = nil
			return m, fetchNAEPData(m.naepClient, m.selectedItem)
		}
		return m, nil

	// Scrolling keys
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleSavePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.currentView = detailView
		m.saveInput.SetValue("")
		return m, nil

	case tea.KeyEnter:
		filename := m.saveInput.Value()
		if filename == "" {
			m.err = fmt.Errorf("filename cannot be empty")
			return m, nil
		}
		return m, saveSchoolData(m.selectedItem, m.enhancedData, m.naepData, filename)
	}

	var cmd tea.Cmd
	m.saveInput, cmd = m.saveInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.currentView == detailView {
		return m.detailViewRender()
	} else if m.currentView == savePromptView {
		return m.savePromptView()
	}
	return m.searchViewRender()
}

func (m model) searchViewRender() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	b.WriteString(headerStyle.Render("🏫 School Finder"))
	b.WriteString("\n\n")

	// Search input
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	b.WriteString(inputStyle.Render(m.searchInput.View()))
	b.WriteString("\n")

	// State filter
	stateText := "All States"
	if m.stateFilter != "" {
		stateText = m.stateFilter
	}
	b.WriteString(fmt.Sprintf("State Filter: %s (Ctrl+S to cycle)", stateText))
	b.WriteString("\n\n")

	// Loading indicator
	if m.loading {
		b.WriteString("Loading...\n")
	}

	// Error display
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	// Results summary stats
	if len(m.schools) > 0 {
		// Calculate quick stats
		totalEnrollment := int64(0)
		totalTeachers := 0.0
		schoolsWithData := 0

		for _, school := range m.schools {
			if school.Enrollment.Valid {
				totalEnrollment += school.Enrollment.Int64
			}
			if school.Teachers.Valid {
				totalTeachers += school.Teachers.Float64
				schoolsWithData++
			}
		}

		avgEnrollment := 0.0
		if len(m.schools) > 0 {
			avgEnrollment = float64(totalEnrollment) / float64(len(m.schools))
		}

		avgTeachers := 0.0
		if schoolsWithData > 0 {
			avgTeachers = totalTeachers / float64(schoolsWithData)
		}

		// Stats display
		statsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

		stats := fmt.Sprintf("Results: %d schools | Avg Enrollment: %.0f | Avg Teachers: %.1f",
			len(m.schools), avgEnrollment, avgTeachers)
		b.WriteString(statsStyle.Render(stats))
		b.WriteString("\n")

		b.WriteString(m.list.View())
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	help := "\nTab: Switch focus | Enter: Search/Select | Ctrl+S: Filter by state | Esc/Ctrl+C: Quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) detailViewContent() string {
	if m.selectedItem == nil {
		return "No school selected"
	}

	s := m.selectedItem

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33")).
		Width(20)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	sectionStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		MarginBottom(1)

	// Header
	b.WriteString(titleStyle.Render("🏫 School Details"))
	b.WriteString("\n\n")

	// Basic Info Section
	var basicInfo strings.Builder
	basicInfo.WriteString(labelStyle.Render("School Name:") + " " + valueStyle.Render(s.Name) + "\n")
	basicInfo.WriteString(labelStyle.Render("NCESSCH ID:") + " " + valueStyle.Render(s.NCESSCH) + "\n")
	basicInfo.WriteString(labelStyle.Render("District:") + " " + valueStyle.Render(s.District) + "\n")
	basicInfo.WriteString(labelStyle.Render("School Type:") + " " + valueStyle.Render(s.SchoolTypeString()) + "\n")
	basicInfo.WriteString(labelStyle.Render("Level:") + " " + valueStyle.Render(s.LevelString()) + "\n")
	basicInfo.WriteString(labelStyle.Render("Grade Range:") + " " + valueStyle.Render(s.GradeRangeString()) + "\n")
	basicInfo.WriteString(labelStyle.Render("Charter School:") + " " + valueStyle.Render(s.CharterString()) + "\n")
	basicInfo.WriteString(labelStyle.Render("School Year:") + " " + valueStyle.Render(s.SchoolYear) + "\n")

	b.WriteString(sectionStyle.Render(basicInfo.String()))
	b.WriteString("\n")

	// Location Section
	var locationInfo strings.Builder
	locationInfo.WriteString(labelStyle.Render("Street Address:") + " " + valueStyle.Render(s.FullAddress()) + "\n")
	locationInfo.WriteString(labelStyle.Render("City:") + " " + valueStyle.Render(s.City) + "\n")
	locationInfo.WriteString(labelStyle.Render("State:") + " " + valueStyle.Render(fmt.Sprintf("%s (%s)", s.StateName, s.State)) + "\n")
	locationInfo.WriteString(labelStyle.Render("Zip Code:") + " " + valueStyle.Render(s.ZipString()) + "\n")

	b.WriteString(sectionStyle.Render(locationInfo.String()))
	b.WriteString("\n")

	// Contact Section
	var contactInfo strings.Builder
	contactInfo.WriteString(labelStyle.Render("Phone:") + " " + valueStyle.Render(s.PhoneString()) + "\n")
	contactInfo.WriteString(labelStyle.Render("Website:") + " " + valueStyle.Render(s.WebsiteString()) + "\n")

	b.WriteString(sectionStyle.Render(contactInfo.String()))
	b.WriteString("\n")

	// Enrollment & Staffing Section
	var statsInfo strings.Builder
	statsInfo.WriteString(labelStyle.Render("Total Enrollment:") + " " + valueStyle.Render(s.EnrollmentString()) + "\n")
	statsInfo.WriteString(labelStyle.Render("Teachers (FTE):") + " " + valueStyle.Render(s.TeachersString()) + "\n")
	statsInfo.WriteString(labelStyle.Render("Student/Teacher:") + " " + valueStyle.Render(s.StudentTeacherRatio()) + "\n")

	b.WriteString(sectionStyle.Render(statsInfo.String()))
	b.WriteString("\n")

	// Visualizations Section
	if s.Enrollment.Valid && s.Enrollment.Int64 > 0 {
		vizTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Render("📊 Metrics Visualization")

		b.WriteString(vizTitle)
		b.WriteString("\n\n")

		// Enrollment bar
		avgEnrollment := 500.0 // National average assumption
		enrollBar := BarChart("Enrollment     ", float64(s.Enrollment.Int64), avgEnrollment*2, 40, lipgloss.Color("33"))
		b.WriteString(enrollBar)
		b.WriteString("\n")

		// Teachers bar
		if s.Teachers.Valid && s.Teachers.Float64 > 0 {
			avgTeachers := 30.0
			teacherBar := BarChart("Teachers (FTE) ", s.Teachers.Float64, avgTeachers*2, 40, lipgloss.Color("201"))
			b.WriteString(teacherBar)
			b.WriteString("\n")
		}

		// Student/Teacher ratio indicator
		if s.Teachers.Valid && s.Teachers.Float64 > 0 {
			ratio := float64(s.Enrollment.Int64) / s.Teachers.Float64
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33")).Render("Student/Teacher Ratio Analysis:"))
			b.WriteString("\n")
			b.WriteString(RatioIndicator(ratio, 15.0, 25.0))
			b.WriteString(fmt.Sprintf("\nCurrent Ratio: %.1f:1", ratio))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// NAEP Data Section
	if m.naepData != nil {
		naepTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33")).
			Render("📊 Nation's Report Card (NAEP) Scores")

		b.WriteString(naepTitle)
		b.WriteString("\n\n")

		// State-level scores
		if len(m.naepData.StateScores) > 0 {
			stateHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("62")).
				Render(fmt.Sprintf("State: %s", m.naepData.State))
			b.WriteString(stateHeader)
			b.WriteString("\n\n")

			// Group scores by subject and grade
			subjects := []string{"mathematics", "reading", "science"}
			for _, subject := range subjects {
				hasData := false
				for _, score := range m.naepData.StateScores {
					if score.Subject == subject && score.MeanScore > 0 {
						hasData = true
						break
					}
				}
				if !hasData {
					continue
				}

				subjectTitle := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("226")).
					Render(strings.Title(subject))
				b.WriteString(subjectTitle)
				b.WriteString("\n")

				// Show scores for each grade
				grades := []int{4, 8, 12}
				for _, grade := range grades {
					current, previous, change := m.naepData.GetScoreTrend(subject, grade, false)
					if current == nil {
						continue
					}

					label := fmt.Sprintf("  Grade %d (%d)", grade, current.Year)
					scoreInfo := fmt.Sprintf("Score: %.0f", current.MeanScore)
					if previous != nil {
						scoreInfo += " " + NAEPTrendIndicator(change)
					}
					if current.AtProficient > 0 {
						scoreInfo += fmt.Sprintf(" | %.0f%% Proficient+", current.AtProficient)
					}

					b.WriteString(fmt.Sprintf("%-25s %s\n", label, scoreInfo))

					// Show score gauge
					if current.MeanScore > 0 {
						b.WriteString("    ")
						b.WriteString(NAEPScoreGauge(current.MeanScore, 50))
						b.WriteString("\n")
					}
				}
				b.WriteString("\n")
			}
		}

		// District-level scores (if available)
		if len(m.naepData.DistrictScores) > 0 {
			districtHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("201")).
				Render(fmt.Sprintf("District: %s", m.naepData.District))
			b.WriteString(districtHeader)
			b.WriteString("\n\n")

			subjects := []string{"mathematics", "reading", "science"}
			for _, subject := range subjects {
				hasData := false
				for _, score := range m.naepData.DistrictScores {
					if score.Subject == subject && score.MeanScore > 0 {
						hasData = true
						break
					}
				}
				if !hasData {
					continue
				}

				subjectTitle := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("226")).
					Render(strings.Title(subject))
				b.WriteString(subjectTitle)
				b.WriteString("\n")

				grades := []int{4, 8, 12}
				for _, grade := range grades {
					current, previous, change := m.naepData.GetScoreTrend(subject, grade, true)
					if current == nil {
						continue
					}

					label := fmt.Sprintf("  Grade %d (%d)", grade, current.Year)
					scoreInfo := fmt.Sprintf("Score: %.0f", current.MeanScore)
					if previous != nil {
						scoreInfo += " " + NAEPTrendIndicator(change)
					}
					if current.AtProficient > 0 {
						scoreInfo += fmt.Sprintf(" | %.0f%% Proficient+", current.AtProficient)
					}

					b.WriteString(fmt.Sprintf("%-25s %s\n", label, scoreInfo))

					// Show score gauge
					if current.MeanScore > 0 {
						b.WriteString("    ")
						b.WriteString(NAEPScoreGauge(current.MeanScore, 50))
						b.WriteString("\n")
					}
				}
				b.WriteString("\n")
			}
		}

		cacheNote := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf("Cached: %s (90-day TTL)", m.naepData.ExtractedAt.Format("2006-01-02")))
		b.WriteString(cacheNote)
		b.WriteString("\n\n")
	}

	// AI-Enhanced Data Section
	if m.enhancedData != nil {
		aiTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("201")).
			Render("🤖 AI-Extracted Information")

		b.WriteString(aiTitle)
		b.WriteString("\n\n")

		// If we have markdown content, render it with glamour
		if m.enhancedData.MarkdownContent != "" {
			rendered, err := renderMarkdown(m.enhancedData.MarkdownContent, m.width)
			if err != nil {
				// Fallback to plain markdown if rendering fails
				b.WriteString("Source: " + m.enhancedData.SourceURL + "\n")
				b.WriteString("Extracted: " + m.enhancedData.ExtractedAt.Format("2006-01-02 15:04") + "\n\n")
				b.WriteString(m.enhancedData.MarkdownContent)
			} else {
				// Use glamour-rendered content
				b.WriteString("Source: " + m.enhancedData.SourceURL + "\n")
				b.WriteString("Extracted: " + m.enhancedData.ExtractedAt.Format("2006-01-02 15:04") + "\n\n")
				b.WriteString(rendered)
			}
		} else {
			// Fall back to legacy structured format
			enhancedText := FormatEnhancedData(m.enhancedData)
			b.WriteString(enhancedText)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *model) updateDetailViewport() {
	if !m.viewportReady || m.selectedItem == nil {
		return
	}
	content := m.detailViewContent()
	m.viewport.SetContent(content)
}

func (m model) detailViewRender() string {
	if !m.viewportReady || m.selectedItem == nil {
		return "Loading..."
	}

	var b strings.Builder

	// Render viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Add scroll indicator if content is scrollable
	if m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPercent := int(m.viewport.ScrollPercent() * 100)
		scrollInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf("─── %d%% ───", scrollPercent))
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}

	// Status indicators (always visible)
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true)

	// NAEP loading status
	if m.loadingNAEP {
		b.WriteString(statusStyle.Render("⏳ Fetching NAEP data..."))
		b.WriteString("\n")
	}

	// AI scraping status
	if m.scrapingAI {
		b.WriteString(statusStyle.Render("⏳ Scraping website with AI..."))
		b.WriteString("\n")
	}

	// Save success message
	if m.saveSuccess != "" {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		b.WriteString(successStyle.Render("✓ " + m.saveSuccess))
		b.WriteString("\n")
	}

	// Error display
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help text (always visible at bottom)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	var help string
	s := m.selectedItem

	// Build NAEP shortcut text
	naepText := "Ctrl+N: NAEP"
	if m.autoFetchNAEP {
		naepText = "Ctrl+N: Refresh NAEP"
	}

	if m.enhancedData != nil {
		help = fmt.Sprintf("↑/↓/PgUp/PgDn: Scroll | Ctrl+W: Save | Ctrl+E: Edit | %s | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit", naepText)
	} else if m.aiScraper != nil && s.Website.Valid && s.Website.String != "" {
		help = fmt.Sprintf("↑/↓/PgUp/PgDn: Scroll | Ctrl+W: Save | Ctrl+A: AI Extract | %s | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit", naepText)
	} else {
		help = fmt.Sprintf("↑/↓/PgUp/PgDn: Scroll | Ctrl+W: Save | %s | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit", naepText)
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) savePromptView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("💾 Save School Data"))
	b.WriteString("\n\n")

	if m.selectedItem != nil {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
		b.WriteString(infoStyle.Render(fmt.Sprintf("Saving data for: %s", m.selectedItem.Name)))
		b.WriteString("\n\n")
	}

	// Input prompt
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	b.WriteString("Filename: ")
	b.WriteString(inputStyle.Render(m.saveInput.View()))
	b.WriteString("\n\n")

	// Info text
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	info := "The file will contain:\n"
	info += "  • School information (name, location, contact, enrollment, etc.)\n"
	if m.enhancedData != nil {
		info += "  • AI-extracted data (principal, programs, activities, etc.)\n"
	}
	info += "\nFormat: JSON"
	b.WriteString(infoStyle.Render(info))
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	help := "Enter: Save | Esc: Cancel | Ctrl+C: Quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func main() {
	dataDir := "tmpdata/"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	// Setup logger first
	if err := setupLogger(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		// Continue without logging rather than fail
	}

	// Check for required data files
	missing, err := CheckDataFiles(dataDir)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to check data files", "error", err, "data_dir", dataDir)
		}
		fmt.Fprintf(os.Stderr, "Error checking data files: %v\n", err)
		os.Exit(1)
	}

	// If files are missing, prompt user to download
	if len(missing) > 0 {
		if PromptUserForDownload(missing) {
			if err := DownloadAndExtractFiles(dataDir, missing); err != nil {
				if logger != nil {
					logger.Error("Failed to download data files", "error", err, "missing_files", missing)
				}
				fmt.Fprintf(os.Stderr, "Error downloading files: %v\n", err)
				os.Exit(1)
			}
		} else {
			if logger != nil {
				logger.Warn("User declined to download required data files", "missing_files", missing)
			}
			fmt.Println("\n❌ Cannot proceed without required data files.")
			fmt.Println("Please download the files manually or run the program again.")
			os.Exit(1)
		}
	}

	db, err := NewDB(dataDir)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to initialize database", "error", err, "data_dir", dataDir)
		}
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize AI scraper (optional - requires ANTHROPIC_API_KEY)
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	var aiScraper *AIScraperService
	if apiKey != "" {
		aiScraper, err = NewAIScraperService(apiKey, ".school_cache")
		if err != nil {
			if logger != nil {
				logger.Warn("AI scraper initialization failed", "error", err)
			}
			fmt.Fprintf(os.Stderr, "Warning: AI scraper initialization failed: %v\n", err)
			aiScraper = nil
		}
	}

	// Initialize NAEP client
	naepClient := NewNAEPClient(".naep_cache")

	// Print configuration info
	fmt.Println("\n📊 School Finder Configuration:")
	if os.Getenv("NAEP_AUTO_FETCH") == "" || (os.Getenv("NAEP_AUTO_FETCH") != "0" && os.Getenv("NAEP_AUTO_FETCH") != "false" && os.Getenv("NAEP_AUTO_FETCH") != "no") {
		fmt.Println("   • NAEP Auto-Fetch: ✓ Enabled (set NAEP_AUTO_FETCH=0 to disable)")
	} else {
		fmt.Println("   • NAEP Auto-Fetch: ✗ Disabled (unset NAEP_AUTO_FETCH to enable)")
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Println("   • AI Website Scraper: ✓ Available")
	} else {
		fmt.Println("   • AI Website Scraper: ✗ Not configured (set ANTHROPIC_API_KEY)")
	}
	fmt.Println()

	p := tea.NewProgram(
		initialModel(db, aiScraper, naepClient),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
