# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

School Finder TUI is a terminal user interface application for searching and exploring school data from the Common Core of Data (CCD). It uses DuckDB for fast queries and Charmbracelet's Bubble Tea framework for the interactive interface.

## Building and Running

### Build
```bash
go build -o schoolfinder
```

### Run
```bash
# Run from current directory (looks for data in tmpdata/)
./schoolfinder

# Or specify data directory
./schoolfinder /path/to/csv/files
```

### With AI Scraper (Optional)
```bash
# Set API key
export ANTHROPIC_API_KEY='your-key-here'

# Run
./schoolfinder
```

## Testing

There are currently no automated tests. To test manually:
1. Build the application
2. Run with sample data
3. Test search functionality, state filters, and detail views
4. Test AI scraper with valid API key

## Architecture

### File Structure
- **main.go** - TUI application, view rendering, event handling, and state management
- **db.go** - DuckDB integration and data access layer with table creation and indexing
- **data_downloader.go** - Automatic CSV download and extraction from NCES
- **ai_scraper.go** - Claude 4.5 Haiku with web search integration for extracting school staff contact information
- **charts.go** - ASCII chart and visualization utilities

### Data Flow
1. **Database Layer (db.go)**: DuckDB persistent database with indexed tables
   - On first run: Creates data.duckdb and imports CSV files into tables
   - Three tables: directory (102K schools), teachers (100K), enrollment (11M records)
   - B-tree indexes on NCESSCH for fast joins, ST for state filtering
   - FTS index for full-text search across: school name, district, city, street, zip
   - SQL queries with LEFT JOINs to combine data from tables
   - Uses BM25 relevance ranking for search result ordering
   - Returns School structs with nullable fields for missing data

2. **TUI Layer (main.go)**: Bubble Tea Elm architecture
   - Model: Application state (current view, search input, selected school)
   - Update: Event handling (key presses, async results via messages)
   - View: Renders search view, detail view, or save prompt view
   - Async operations use goroutines and communicate via tea.Msg

3. **AI Scraper (ai_scraper.go)**: Claude 4.5 Haiku with Web Search
   - Uses web search tool to find staff directories and contact pages
   - Extracts structured data including staff contacts (name, title, email, phone, department)
   - Also extracts school info (principal, programs, sports, facilities, etc.)
   - 30-day file-based cache in .school_cache/
   - Returns EnhancedSchoolData struct with StaffContact array

4. **Data Downloader (data_downloader.go)**: Automatic CSV acquisition
   - Checks for required CSV files on startup
   - Prompts user for confirmation before downloading
   - Downloads ZIP files from NCES with progress tracking
   - Extracts CSV files and cleans up temporary files
   - User-friendly with progress indicators (percentage, MB downloaded)

5. **Visualizations (charts.go)**: ASCII chart utilities
   - BarChart, PercentageBar, RatioIndicator, etc.
   - Used in detail view to show enrollment, teacher counts, and ratios

### Startup Sequence

1. **Data File Check**: CheckDataFiles() verifies CSV presence
2. **Auto-Download** (if missing): User prompted to download from NCES
3. **Database Init** (if needed): Creates data.duckdb and imports CSVs
4. **TUI Launch**: Bubble Tea starts interactive interface

### Key Design Patterns

**Bubble Tea Elm Architecture**:
- All state changes go through the Update() method
- Async operations send messages back to Update
- Views are pure functions of state

**View State Machine**:
- Three views: searchView, detailView, savePromptView
- Transitions via currentView field
- Each view has its own key handlers

**Nullable Database Fields**:
- sql.NullString, sql.NullFloat64, sql.NullInt64 for optional data
- Helper methods on School struct (e.g., TeachersString()) handle null values

**AI Caching Strategy**:
- Cache key: NCESSCH ID (unique school identifier)
- Cache TTL: 30 days
- Cache location: .school_cache/ directory
- Cached data includes extraction timestamp

### Common CSV Data Locations
The application expects CSV files in tmpdata/:
- ccd_sch_029_2324_w_1a_073124.csv (school directory)
- ccd_sch_059_2324_l_1a_073124.csv (teacher FTE counts)
- ccd_sch_052_2324_l_1a_073124.csv (student enrollment)

## Key Commands and Shortcuts

### Search View
- Type to search (name, city, district, address, zip)
- Tab: Switch focus between input and results
- Enter: Execute search or view details
- Ctrl+S: Cycle state filters
- Esc/Ctrl+C: Quit

### Detail View
- Ctrl+A: AI extract website data
- Ctrl+Y: Copy school ID to clipboard
- Ctrl+W: Save school data to JSON file
- Ctrl+E: Edit cached AI data in $EDITOR
- Esc: Return to search
- Ctrl+C: Quit

## Database Query Patterns

All queries use indexed DuckDB tables for fast performance:

```sql
SELECT d.*, t.TEACHERS, e.STUDENT_COUNT
FROM directory d
LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH
  AND e.TOTAL_INDICATOR = 'Education Unit Total'
WHERE LOWER(d.SCH_NAME) LIKE LOWER('%query%')
ORDER BY d.SCH_NAME
LIMIT 100
```

**Indexes**:
- `idx_directory_ncessch` on directory(NCESSCH)
- `idx_directory_state` on directory(ST)
- `idx_directory_name` on directory(SCH_NAME)
- `idx_teachers_ncessch` on teachers(NCESSCH)
- `idx_enrollment_ncessch` on enrollment(NCESSCH)

**Full-Text Search**:
-Uses DuckDB's FTS extension with BM25 ranking algorithm
- FTS index created on: SCH_NAME, LEA_NAME, MCITY, MSTREET1, MZIP
- Supports natural language queries ("lincoln high" finds all Lincoln High Schools)
- Results ordered by relevance score
- Query syntax: `fts_main_directory.match_bm25(NCESSCH, 'search query')`

**Database Initialization** (first run only):
- Creates tmpdata/data.duckdb (~323MB)
- Imports CSVs with `CREATE TABLE AS SELECT * FROM read_csv(...)`
- Creates B-tree indexes for fast joins
- Creates FTS index for relevance-ranked searches
- Takes ~13 seconds total (includes 1s for FTS index)

## Adding New Features

### Adding a New View
1. Add view constant to main.go (e.g., `myView view = iota`)
2. Create handler method `handleMyViewKeys(msg tea.KeyMsg)`
3. Create render method `myViewRender() string`
4. Update Update() method to route to new handlers
5. Update View() method to render new view

### Adding New School Data Fields
1. Add field to School struct in db.go
2. Update SQL queries in SearchSchools() and GetSchoolByID()
3. Add helper method for display (e.g., MyFieldString())
4. Update detail view rendering in main.go

### Adding AI Extraction Fields
1. Add field to EnhancedSchoolData struct in ai_scraper.go
2. Update extraction prompt to request new field
3. Update FormatEnhancedData() to display new field

### Adding New Visualizations
1. Add chart function to charts.go
2. Call from detail view in main.go
3. Use lipgloss for styling consistency

## Dependencies

Key packages:
- **github.com/marcboeker/go-duckdb** - DuckDB driver (v1.8.5)
- **github.com/charmbracelet/bubbletea** - TUI framework (v1.3.10)
- **github.com/charmbracelet/bubbles** - TUI components (v0.21.0)
- **github.com/charmbracelet/lipgloss** - Styling (v1.1.0)
- **github.com/anthropics/anthropic-sdk-go** - Claude API (v1.14.0)
- **github.com/atotto/clipboard** - Clipboard access (v0.1.4)

## Environment Variables

- **ANTHROPIC_API_KEY** - Required for AI scraper feature
- **EDITOR** or **VISUAL** - Used for Ctrl+E (editing cached AI data)

## Performance Characteristics

- **First-time setup**: ~13 seconds to create database from CSVs (includes FTS indexing)
- **Subsequent loads**: Instant (database already initialized)
- **Search queries**:
  - Full-text search with relevance ranking: <10ms
  - Complex queries with joins: <20ms
  - State filtering combined with FTS: <15ms
- **AI extraction**: 2-5 seconds per school (cached for 30 days)
- **Memory**: ~70MB binary + ~50MB DuckDB working memory
- **Disk usage**:
  - CSV files: 2.3GB (can be deleted after DB creation)
  - Database file: 323MB (14% of CSV size)
- **Dataset**: 102,274 schools, 11.2M enrollment records
- **FTS Performance**:
  - BM25 relevance ranking algorithm
  - Supports phrase queries and Boolean operations
  - Results returned in order of relevance (highest score first)
