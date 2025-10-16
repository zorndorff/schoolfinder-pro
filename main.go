package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxResults = 100
)

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
	db            *DB
	aiScraper     *AIScraperService
	currentView   view
	searchInput   textinput.Model
	saveInput     textinput.Model
	stateFilter   string
	schools       []School
	list          list.Model
	selectedItem  *School
	enhancedData  *EnhancedSchoolData
	width         int
	height        int
	err           error
	loading       bool
	scrapingAI    bool
	saveSuccess   string
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

func scrapeSchoolWebsite(scraper *AIScraperService, school *School) tea.Cmd {
	return func() tea.Msg {
		data, err := scraper.ScrapeSchoolWebsite(context.Background(), school)
		return aiScrapeMsg{data: data, err: err}
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

func saveSchoolData(school *School, enhanced *EnhancedSchoolData, filename string) tea.Cmd {
	return func() tea.Msg {
		// Create a combined data structure
		data := map[string]interface{}{
			"school": school,
		}

		if enhanced != nil {
			data["ai_extracted"] = enhanced
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

func initialModel(db *DB, aiScraper *AIScraperService) model {
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

	return model{
		db:          db,
		aiScraper:   aiScraper,
		currentView: searchView,
		searchInput: ti,
		saveInput:   si,
		list:        l,
		schools:     []School{},
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
		return m, nil

	case tea.KeyMsg:
		if m.currentView == detailView {
			return m.handleDetailViewKeys(msg)
		} else if m.currentView == savePromptView {
			return m.handleSavePromptKeys(msg)
		}
		return m.handleSearchViewKeys(msg)

	case searchMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.schools = msg.schools
		items := make([]list.Item, len(msg.schools))
		for i, school := range msg.schools {
			items[i] = schoolItem{school: school}
		}
		m.list.SetItems(items)
		return m, nil

	case aiScrapeMsg:
		m.scrapingAI = false
		if msg.err != nil {
			m.err = fmt.Errorf("AI scraping failed: %w", msg.err)
			return m, nil
		}
		m.enhancedData = msg.data
		m.err = nil
		return m, nil

	case saveMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("save failed: %w", msg.err)
			m.currentView = detailView
			return m, nil
		}
		m.saveSuccess = fmt.Sprintf("Saved to: %s", msg.filename)
		m.saveInput.SetValue("")
		m.currentView = detailView
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
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.currentView = searchView
		m.selectedItem = nil
		m.enhancedData = nil
		m.err = nil
		m.saveSuccess = ""
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
		return m, saveSchoolData(m.selectedItem, m.enhancedData, filename)
	}

	var cmd tea.Cmd
	m.saveInput, cmd = m.saveInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.currentView == detailView {
		return m.detailView()
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

func (m model) detailView() string {
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

	// Scraping status
	if m.scrapingAI {
		scrapingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)
		b.WriteString(scrapingStyle.Render("⏳ Scraping website with AI..."))
		b.WriteString("\n\n")
	}

	// Save success message
	if m.saveSuccess != "" {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		b.WriteString(successStyle.Render("✓ " + m.saveSuccess))
		b.WriteString("\n\n")
	}

	// Error display
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	var help string
	if m.enhancedData != nil {
		help = "Ctrl+W: Save | Ctrl+E: Edit | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit"
	} else if m.aiScraper != nil && s.Website.Valid && s.Website.String != "" {
		help = "Ctrl+W: Save | Ctrl+A: AI Extract | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit"
	} else {
		help = "Ctrl+W: Save | Ctrl+Y: Copy ID | Esc: Back | Ctrl+C: Quit"
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

	// Check for required data files
	missing, err := CheckDataFiles(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking data files: %v\n", err)
		os.Exit(1)
	}

	// If files are missing, prompt user to download
	if len(missing) > 0 {
		if PromptUserForDownload(missing) {
			if err := DownloadAndExtractFiles(dataDir, missing); err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading files: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("\n❌ Cannot proceed without required data files.")
			fmt.Println("Please download the files manually or run the program again.")
			os.Exit(1)
		}
	}

	db, err := NewDB(dataDir)
	if err != nil {
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
			fmt.Fprintf(os.Stderr, "Warning: AI scraper initialization failed: %v\n", err)
			aiScraper = nil
		}
	}

	p := tea.NewProgram(
		initialModel(db, aiScraper),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
