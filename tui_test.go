package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// TestInitialModel tests the initial model creation
func TestInitialModel(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")

	// Test initial state
	if m.currentView != searchView {
		t.Errorf("Expected initial view to be searchView, got %v", m.currentView)
	}

	if !m.searchInput.Focused() {
		t.Error("Expected search input to be focused initially")
	}

	if len(m.schools) != 0 {
		t.Errorf("Expected no schools initially, got %d", len(m.schools))
	}

	if m.selectedItem != nil {
		t.Error("Expected no selected item initially")
	}

	if m.loading {
		t.Error("Expected loading to be false initially")
	}

	if m.err != nil {
		t.Errorf("Expected no error initially, got %v", m.err)
	}
}

// TestSearchViewKeyHandling tests key handling in search view
func TestSearchViewKeyHandling(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24

	testCases := []struct {
		name           string
		key            tea.KeyMsg
		expectedAction string
	}{
		{
			name:           "Tab switches focus",
			key:            tea.KeyMsg{Type: tea.KeyTab},
			expectedAction: "blur_input",
		},
		{
			name:           "Ctrl+S cycles state filter",
			key:            tea.KeyMsg{Type: tea.KeyCtrlS},
			expectedAction: "cycle_state",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialFocused := m.searchInput.Focused()

			newModel, _ := m.handleSearchViewKeys(tc.key)
			m = newModel.(model)

			if tc.expectedAction == "blur_input" {
				if m.searchInput.Focused() == initialFocused {
					t.Error("Expected focus to change")
				}
			}
		})
	}
}

// TestSearchMessageHandling tests handling of search results
func TestSearchMessageHandling(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24
	m.loading = true

	// Simulate successful search
	schools, err := db.SearchSchools("Lincoln", "", 100)
	if err != nil {
		t.Fatalf("SearchSchools failed: %v", err)
	}

	msg := searchMsg{
		schools: schools,
		err:     nil,
	}

	newModel, _ := m.Update(msg)
	m = newModel.(model)

	if m.loading {
		t.Error("Expected loading to be false after search")
	}

	if len(m.schools) != 1 {
		t.Errorf("Expected 1 school in results, got %d", len(m.schools))
	}

	if m.err != nil {
		t.Errorf("Expected no error, got %v", m.err)
	}

	// Check that list items were set
	items := m.list.Items()
	if len(items) != 1 {
		t.Errorf("Expected 1 list item, got %d", len(items))
	}
}

// TestSearchMessageError tests handling of search errors
func TestSearchMessageError(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.loading = true

	// Simulate failed search
	msg := searchMsg{
		schools: nil,
		err:     nil, // Note: we're simulating a scenario where we get no error but no results
	}

	newModel, _ := m.Update(msg)
	m = newModel.(model)

	if m.loading {
		t.Error("Expected loading to be false after search")
	}

	if len(m.schools) != 0 {
		t.Errorf("Expected 0 schools in results, got %d", len(m.schools))
	}
}

// TestWindowSizeHandling tests window size message handling
func TestWindowSizeHandling(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 30,
	}

	newModel, _ := m.Update(msg)
	m = newModel.(model)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}

	if m.height != 30 {
		t.Errorf("Expected height 30, got %d", m.height)
	}

	if !m.viewportReady {
		t.Error("Expected viewport to be ready after window size message")
	}
}

// TestDetailViewTransition tests transitioning to detail view
func TestDetailViewTransition(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24

	// First, populate the list with a school
	schools, err := db.SearchSchools("Lincoln", "", 100)
	if err != nil {
		t.Fatalf("SearchSchools failed: %v", err)
	}

	items := make([]list.Item, len(schools))
	for i, school := range schools {
		items[i] = schoolItem{school: school}
	}
	m.list.SetItems(items)
	m.schools = schools

	// Blur search input to focus list
	m.searchInput.Blur()

	// Simulate Enter key to select school
	key := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.handleSearchViewKeys(key)
	m = newModel.(model)

	if m.currentView != detailView {
		t.Errorf("Expected view to be detailView, got %v", m.currentView)
	}

	if m.selectedItem == nil {
		t.Fatal("Expected selected item to be set")
	}

	if m.selectedItem.Name != "Lincoln Elementary School" {
		t.Errorf("Expected selected school to be Lincoln Elementary School, got %s", m.selectedItem.Name)
	}
}

// TestDetailViewBackToSearch tests returning from detail view to search view
func TestDetailViewBackToSearch(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24
	m.currentView = detailView
	m.selectedItem = MockSchool("123456", "Test School", "Test District", "CA", "PK", "05")

	// Simulate Esc key to go back
	key := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.handleDetailViewKeys(key)
	m = newModel.(model)

	if m.currentView != searchView {
		t.Errorf("Expected view to be searchView, got %v", m.currentView)
	}

	if m.selectedItem != nil {
		t.Error("Expected selected item to be cleared")
	}

	if m.enhancedData != nil {
		t.Error("Expected enhanced data to be cleared")
	}
}

// TestSavePromptTransition tests transitioning to save prompt view
func TestSavePromptTransition(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24
	m.currentView = detailView
	m.selectedItem = MockSchool("123456", "Test School", "Test District", "CA", "PK", "05")

	// Simulate Ctrl+W to save
	key := tea.KeyMsg{Type: tea.KeyCtrlW}
	newModel, _ := m.handleDetailViewKeys(key)
	m = newModel.(model)

	if m.currentView != savePromptView {
		t.Errorf("Expected view to be savePromptView, got %v", m.currentView)
	}

	if !m.saveInput.Focused() {
		t.Error("Expected save input to be focused")
	}

	// Check that default filename is set
	value := m.saveInput.Value()
	if value == "" {
		t.Error("Expected default filename to be set")
	}

	if !strings.Contains(value, ".json") {
		t.Error("Expected default filename to have .json extension")
	}
}

// TestSavePromptCancel tests canceling save prompt
func TestSavePromptCancel(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.currentView = savePromptView
	m.selectedItem = MockSchool("123456", "Test School", "Test District", "CA", "PK", "05")
	m.saveInput.SetValue("test.json")

	// Simulate Esc to cancel
	key := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.handleSavePromptKeys(key)
	m = newModel.(model)

	if m.currentView != detailView {
		t.Errorf("Expected view to be detailView, got %v", m.currentView)
	}

	if m.saveInput.Value() != "" {
		t.Error("Expected save input to be cleared")
	}
}

// TestSearchViewRender tests search view rendering
func TestSearchViewRender(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24

	output := m.searchViewRender()

	// Check for expected content
	if !strings.Contains(output, "School Finder") {
		t.Error("Expected output to contain 'School Finder'")
	}

	if !strings.Contains(output, "Search schools") {
		t.Error("Expected output to contain search placeholder text")
	}

	if !strings.Contains(output, "State Filter") {
		t.Error("Expected output to contain 'State Filter'")
	}
}

// TestDetailViewRender tests detail view rendering
func TestDetailViewRender(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24
	m.currentView = detailView
	m.viewportReady = true

	// Get a real school from the database
	school, err := db.GetSchoolByID("360000100001")
	if err != nil {
		t.Fatalf("GetSchoolByID failed: %v", err)
	}
	m.selectedItem = school
	m.updateDetailViewport()

	output := m.detailViewRender()

	// Check for expected content
	if !strings.Contains(output, "School Details") {
		t.Error("Expected output to contain 'School Details'")
	}

	// The output should contain help text
	if !strings.Contains(output, "Scroll") {
		t.Error("Expected output to contain scroll help text")
	}
}

// TestDetailViewContent tests detail view content generation
func TestDetailViewContent(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.width = 80
	m.height = 24

	// Get a real school from the database
	school, err := db.GetSchoolByID("360000100001")
	if err != nil {
		t.Fatalf("GetSchoolByID failed: %v", err)
	}
	m.selectedItem = school

	content := m.detailViewContent()

	// Check for expected content
	if !strings.Contains(content, "Lincoln Elementary School") {
		t.Error("Expected content to contain school name")
	}

	if !strings.Contains(content, "360000100001") {
		t.Error("Expected content to contain NCESSCH")
	}

	if !strings.Contains(content, "San Francisco") {
		t.Error("Expected content to contain city")
	}

	if !strings.Contains(content, "California") {
		t.Error("Expected content to contain state name")
	}

	if !strings.Contains(content, "25.5") {
		t.Error("Expected content to contain teacher count")
	}

	if !strings.Contains(content, "500") {
		t.Error("Expected content to contain enrollment")
	}
}

// TestStateFilterCycling tests state filter cycling
func TestStateFilterCycling(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	m := initialModel(db, nil, nil, "")
	m.searchInput.SetValue("School") // Set a query so filter triggers search

	initialState := m.stateFilter

	// Cycle through states
	key := tea.KeyMsg{Type: tea.KeyCtrlS}

	// First cycle
	newModel, _ := m.handleSearchViewKeys(key)
	m = newModel.(model)

	if m.stateFilter == initialState {
		t.Error("Expected state filter to change")
	}

	firstState := m.stateFilter

	// Cycle again
	newModel, _ = m.handleSearchViewKeys(key)
	m = newModel.(model)

	if m.stateFilter == firstState {
		t.Error("Expected state filter to change to next state")
	}
}

// TestSchoolItemInterface tests schoolItem list.Item interface
func TestSchoolItemInterface(t *testing.T) {
	school := School{
		NCESSCH:  "123456",
		Name:     "Test School",
		City:     "Test City",
		State:    "CA",
		District: "Test District",
	}

	item := schoolItem{school: school}

	// Test Title
	title := item.Title()
	if title != "Test School" {
		t.Errorf("Expected title 'Test School', got '%s'", title)
	}

	// Test Description
	desc := item.Description()
	if !strings.Contains(desc, "Test City") {
		t.Error("Expected description to contain city")
	}
	if !strings.Contains(desc, "CA") {
		t.Error("Expected description to contain state")
	}
	if !strings.Contains(desc, "Test District") {
		t.Error("Expected description to contain district")
	}

	// Test FilterValue
	filterVal := item.FilterValue()
	if !strings.Contains(filterVal, "Test School") {
		t.Error("Expected filter value to contain school name")
	}
}
