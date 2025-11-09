# Testing Guide for School Finder TUI

This document describes the testing strategy and how to run tests for the School Finder TUI application.

## Overview

The testing strategy uses **mock CSV data** to avoid downloading the full 2.3GB dataset during tests. Tests are organized into three main categories:

1. **Database Layer Tests** (`db_test.go`) - Tests DuckDB integration and data access
2. **TUI Model Tests** (`tui_test.go`) - Tests Bubble Tea model logic and state management
3. **NAEP Client Tests** (`naep_client_test.go`) - Tests NAEP data processing (pre-existing)

## Quick Start

### Run All Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific test file
go test -v -run TestSearchSchools

# Run with race detector
go test -v -race ./...
```

### Run Specific Test Categories

```bash
# Database tests only
go test -v -run "TestNewDB|TestSearchSchools|TestGetSchoolByID"

# TUI tests only
go test -v -run "TestInitialModel|TestSearchView|TestDetailView"

# NAEP tests only
go test -v -run "TestDetermineGrades|TestMatchDistrict|TestSortNAEPScores"
```

## Mock Data

### Test Data Location

Mock CSV files are located in `testdata/`:
- `ccd_sch_029_2324_w_1a_073124.csv` - 5 mock schools (directory data)
- `ccd_sch_059_2324_l_1a_073124.csv` - 5 mock teacher counts
- `ccd_sch_052_2324_l_1a_073124.csv` - 7 mock enrollment records

### Mock Schools

The test dataset includes:
1. **Lincoln Elementary School** (CA) - Elementary, 500 students, 25.5 teachers
2. **Washington High School** (CA) - High school, 850 students, 45 teachers
3. **Jefferson Middle School** (TX) - Middle school, 620 students, 30.2 teachers
4. **Roosevelt Charter School** (NY) - Charter high school, 725 students, 38.5 teachers
5. **Madison K-8 School** (FL) - K-8 school, 680 students, 35 teachers

### Test Database Setup

Tests automatically:
1. Create a temporary directory
2. Copy mock CSV files
3. Initialize a DuckDB database
4. Clean up after test completion

Example test helper usage:

```go
func TestMyFeature(t *testing.T) {
    db, cleanup := SetupTestDB(t)
    defer cleanup()

    // Your test code here
    schools, err := db.SearchSchools("Lincoln", "", 100)
    // ...
}
```

## Test Coverage

### Database Layer Tests (`db_test.go`)

**✅ TestNewDB**
- Verifies database initialization with mock data
- Ensures connection is established
- Tests FTS index creation

**✅ TestSearchSchools**
- Search by school name
- Search by city
- Search with state filter
- Search in specific state
- No results handling
- Empty query with state filter

**✅ TestGetSchoolByID**
- Valid school ID retrieval
- Multiple valid IDs
- Non-existent school handling

**✅ TestSchoolFields**
- Basic field population (NCESSCH, City, etc.)
- Nullable field handling (Teachers, Enrollment)
- Contact information (Phone, Website)

**✅ TestSchoolHelperMethods**
- TeachersString()
- EnrollmentString()
- StudentTeacherRatio()
- GradeRangeString()
- CharterString()

**✅ TestSearchLimit**
- Respects limit parameter

**✅ TestDatabasePersistence**
- Data consistency across queries

### TUI Model Tests (`tui_test.go`)

**✅ TestInitialModel**
- Initial view state (searchView)
- Input focus state
- Empty schools list
- No selected item
- Loading and error states

**✅ TestSearchViewKeyHandling**
- Tab switches focus between input and results
- Ctrl+S cycles state filter

**✅ TestSearchMessageHandling**
- Successful search results
- List items population
- Loading state cleared
- Error handling

**✅ TestWindowSizeHandling**
- Width and height updates
- Viewport ready state
- List dimensions adjustment

**✅ TestDetailViewTransition**
- Enter key selects school
- View changes to detailView
- Selected item is set

**✅ TestDetailViewBackToSearch**
- Esc key returns to search
- Selected item cleared
- Enhanced data cleared

**✅ TestSavePromptTransition**
- Ctrl+W opens save prompt
- Save input focused
- Default filename generated

**✅ TestSavePromptCancel**
- Esc cancels save
- Returns to detail view
- Input cleared

**✅ TestSearchViewRender**
- Contains "School Finder" header
- Shows search placeholder
- Displays state filter

**✅ TestDetailViewRender**
- Contains "School Details" header
- Shows help text
- Includes scroll indicator

**✅ TestDetailViewContent**
- School name displayed
- NCESSCH ID shown
- City and state visible
- Teacher count shown
- Enrollment displayed

**✅ TestStateFilterCycling**
- State filter changes on Ctrl+S
- Cycles through states

**✅ TestSchoolItemInterface**
- Title() returns school name
- Description() contains city, state, district
- FilterValue() contains searchable text

## Test Architecture

### Helper Functions (`test_helpers.go`)

**SetupTestDB(t *testing.T) (*DB, func())**
- Creates temporary test database
- Copies mock CSV files
- Returns cleanup function

**MockSchool(...) *School**
- Creates mock School struct
- Useful for unit testing individual functions

**MockNAEPData(...) *NAEPData**
- Creates mock NAEP data
- Supports district scores and multi-year data

### Test Patterns

#### Database Tests
```go
func TestFeature(t *testing.T) {
    db, cleanup := SetupTestDB(t)
    defer cleanup()

    // Test database functionality
    result, err := db.SomeMethod()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    // Assertions
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

#### TUI Model Tests
```go
func TestTUIFeature(t *testing.T) {
    db, cleanup := SetupTestDB(t)
    defer cleanup()

    m := initialModel(db, nil, nil)
    m.width = 80
    m.height = 24

    // Simulate user interaction
    msg := tea.KeyMsg{Type: tea.KeyEnter}
    newModel, _ := m.Update(msg)
    m = newModel.(model)

    // Assertions
    if m.currentView != expectedView {
        t.Errorf("Expected view %v, got %v", expectedView, m.currentView)
    }
}
```

## Continuous Integration

Tests are designed to run in CI environments. Recommended GitHub Actions workflow:

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run tests
        run: go test -v -race -cover ./...

      - name: Run tests with coverage
        run: go test -v -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Best Practices

### 1. Use Table-Driven Tests
```go
testCases := []struct {
    name     string
    input    string
    expected int
}{
    {"case1", "input1", 1},
    {"case2", "input2", 2},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // Test logic
    })
}
```

### 2. Always Clean Up Resources
```go
db, cleanup := SetupTestDB(t)
defer cleanup()  // Ensures cleanup even if test fails
```

### 3. Test Error Cases
```go
// Test both success and failure paths
result, err := db.GetSchoolByID("invalid-id")
if err == nil {
    t.Error("Expected error for invalid ID")
}
```

### 4. Use Descriptive Test Names
```go
// Good
func TestSearchSchoolsByName(t *testing.T) {}

// Better
func TestSearchSchools_WithValidName_ReturnsResults(t *testing.T) {}
```

## Debugging Tests

### Run Specific Test
```bash
go test -v -run TestSearchSchools
```

### Print Debug Information
```go
t.Logf("Debug info: %+v", data)  // Only prints on failure
fmt.Printf("Always prints: %+v", data)  // Always prints
```

### Run with Verbose Output
```bash
go test -v  # Shows all test output
```

### Check Test Coverage
```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Adding New Tests

### 1. Database Tests
Add to `db_test.go`:
```go
func TestNewDatabaseFeature(t *testing.T) {
    db, cleanup := SetupTestDB(t)
    defer cleanup()

    // Test new database functionality
}
```

### 2. TUI Tests
Add to `tui_test.go`:
```go
func TestNewTUIFeature(t *testing.T) {
    db, cleanup := SetupTestDB(t)
    defer cleanup()

    m := initialModel(db, nil, nil)
    // Test new TUI functionality
}
```

### 3. Mock Data
Add to `testdata/` CSV files or update test helpers in `test_helpers.go`

## Known Limitations

1. **AI Scraper Tests**: Tests don't cover AI scraper functionality to avoid API calls
2. **NAEP API Tests**: Tests use mock data instead of real NAEP API calls
3. **File I/O**: Save operations tested without actual file writes in some cases
4. **Network Operations**: No network-dependent tests to ensure offline test runs

## Troubleshooting

### "cannot find package"
```bash
go mod download
go mod tidy
```

### "database locked"
Ensure cleanup functions are called:
```go
defer cleanup()
```

### "permission denied"
Check testdata directory permissions:
```bash
chmod -R 755 testdata/
```

## Performance

Tests are designed to be fast:
- Mock data is minimal (5 schools vs 102K)
- In-memory DuckDB database
- No network calls
- Parallel test execution supported

Expected test execution time:
- Database tests: ~2-3 seconds
- TUI tests: ~1-2 seconds
- NAEP tests: <1 second
- **Total: ~5 seconds**

## Future Improvements

- [ ] Add integration tests with real CSV data (optional, slow tests)
- [ ] Add screenshot/golden file tests for TUI rendering
- [ ] Add benchmarking tests for search performance
- [ ] Add property-based testing for edge cases
- [ ] Add tests for concurrent database access
- [ ] Add E2E tests using teatest library
