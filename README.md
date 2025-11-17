# School Finder Pro

A comprehensive school data exploration platform with CLI, TUI, and web interfaces. Search and analyze data from 102,000+ U.S. schools using the Common Core of Data (CCD), enhanced with AI-powered insights and NAEP performance metrics.

![Demo](./docs/demo.gif)

## Features

### 🔍 **Three Modes of Operation**
- **CLI Mode**: Fast command-line queries with JSON output for scripting
- **TUI Mode**: Interactive terminal interface with keyboard shortcuts
- **Web Mode**: Modern browser interface with HTMX for dynamic updates

### ⚡ **Core Capabilities**
- **Lightning-Fast Search**: DuckDB with full-text search (BM25) across 102K+ schools, <10ms queries
- **Smart Data Integration**: Automatic CSV download from NCES (2.3GB → 323MB optimized database)
- **AI-Powered Data Agent**: Natural language queries using Claude 3.5 Haiku ("Show me top 10 schools in CA by enrollment")
- **Website Intelligence**: Extract staff contacts, programs, and facilities from school websites
- **Academic Performance**: NAEP test score integration for reading and math proficiency
- **Custom Data Import**: Upload and analyze your own school datasets (CSV/Excel)
- **Rich Visualizations**: ASCII charts for terminal, styled tables for web

### 📊 **Data Insights**
- **School Information**: Name, district, type, level, charter status, magnet programs
- **Demographics**: Student enrollment by grade, race/ethnicity, gender
- **Staffing**: Teacher counts (FTE), student-teacher ratios, administrative personnel
- **Performance**: NAEP reading/math scores at district level
- **Contact Details**: Phone, website, full mailing address
- **AI-Enhanced**: Principal info, programs, sports teams, facilities (via web scraping)

## Quick Start

### Installation

```bash
# Build the application
go build -o schoolfinder

# First run will prompt to download data (2.3GB) from NCES
# Creates optimized DuckDB database (323MB) automatically
./schoolfinder
```

### Enable AI Features (Optional)

```bash
# Set your Anthropic API key for AI agent and web scraper
export ANTHROPIC_API_KEY='sk-ant-your-key-here'

# Run with AI capabilities enabled
./schoolfinder
```

**Get an API key:** [console.anthropic.com](https://console.anthropic.com)

## Usage by Mode

### 1. TUI Mode (Default)

Interactive terminal interface with keyboard shortcuts:

```bash
# Launch TUI (default when no subcommand provided)
./schoolfinder

# Or explicitly specify data directory
./schoolfinder --data-dir /path/to/data
```

**Keyboard Shortcuts:**
- **Search View**: Type to search, Tab to switch focus, Ctrl+S for state filter, Enter to view details
- **Detail View**: Ctrl+A for AI extract, Ctrl+N for NAEP data, Ctrl+Y to copy ID, Ctrl+W to save JSON
- **Data Agent**: Ctrl+D to open AI agent, ask questions in natural language
- **Global**: Esc to go back, Ctrl+C to quit

### 2. CLI Mode

Fast command-line operations with JSON output:

```bash
# Search schools
./schoolfinder search "Lincoln High" --state CA --limit 10

# Get school details by ID
./schoolfinder details 062961004587

# Run SQL query directly
./schoolfinder query "SELECT * FROM directory WHERE ST='NY' LIMIT 5"

# Ask AI agent a question
./schoolfinder ask "What are the top 10 largest schools in Texas?"

# Scrape website for additional data
./schoolfinder scrape 062961004587

# Show database schema
./schoolfinder schema

# Summarize search results
./schoolfinder summarize --state CA --type "Regular school"
```

**Output:** All CLI commands return structured JSON for easy parsing and automation.

### 3. Web Mode

Browser-based interface with modern UI:

```bash
# Start web server (default port 3000)
./schoolfinder serve

# Custom port
./schoolfinder serve --port 8080
```

**Open in browser:** `http://localhost:3000`

**Features:**
- 🔍 Real-time search with HTMX updates
- 📊 Interactive charts and visualizations
- 🤖 AI data agent with chat interface
- 📥 Import custom datasets (CSV/Excel)
- 📈 NAEP performance data display
- 🌐 One-click website data extraction

## Architecture

### Project Structure

```
schoolfinder-pro/
├── cmd/                      # CLI subcommands (Cobra)
│   ├── root.go              # Main command and TUI launcher
│   ├── serve.go             # Web server command
│   ├── search.go            # Search command (JSON output)
│   ├── query.go             # SQL query command
│   ├── ask.go               # AI agent command
│   ├── scrape.go            # Website scraper command
│   ├── details.go           # School details command
│   ├── schema.go            # Database schema command
│   └── summarize.go         # Summary statistics command
├── internal/
│   └── agent/               # AI data agent implementation
├── templates/               # HTML templates (Go templates)
│   ├── layout.html          # Base layout with HTMX
│   ├── search.html          # Search page
│   ├── detail.html          # School detail page
│   ├── agent.html           # AI agent chat interface
│   ├── import.html          # Data import page
│   └── partials/            # HTMX partial responses
├── static/                  # CSS, JS, and assets
│   ├── style.css            # Tailwind-based styles
│   └── favicon.ico          # App icon
├── main.go                  # TUI application (Bubble Tea)
├── server.go                # HTTP server setup (Chi router)
├── web_handlers.go          # Web route handlers
├── api_handlers.go          # API endpoints
├── db.go                    # DuckDB database layer
├── ai_scraper.go            # Claude-powered web scraper
├── naep_client.go           # NAEP API integration
├── data_downloader.go       # Automatic CSV download
├── charts.go                # ASCII visualizations
└── tmpdata/                 # Data directory (gitignored)
    ├── data.duckdb          # Optimized database (323MB)
    └── *.csv                # Source files (2.3GB, optional after import)
```

### Technology Stack

**Core:**
- **Go 1.22+**: Performance and concurrency
- **DuckDB**: Embedded analytical database with full-text search
- **Cobra**: CLI framework with subcommands

**TUI:**
- **Bubble Tea**: Elm architecture for terminal UIs
- **Bubbles**: Reusable components (textinput, list, viewport)
- **Lipgloss**: Styling and layouts
- **Glamour**: Markdown rendering

**Web:**
- **Chi**: Lightweight HTTP router
- **HTMX**: Dynamic HTML updates without JavaScript
- **Go Templates**: Server-side rendering
- **Tailwind CSS**: Utility-first styling

**AI:**
- **Anthropic SDK**: Claude 3.5 Haiku integration
- **Web Search Tool**: For website data extraction

### Data Flow

1. **Startup Sequence**
   - Check for CSV files in data directory
   - If missing, prompt to download from NCES (automatic)
   - Create DuckDB database if needed (one-time, ~13s)
   - Build indexes and FTS for fast queries
   - Launch selected mode (TUI, CLI, or Web)

2. **Database Layer** (`db.go`)
   - **Three tables**: `directory` (102K schools), `teachers` (100K), `enrollment` (11M records)
   - **Indexes**: B-tree on NCESSCH (joins), ST (state filter), SCH_NAME (sorting)
   - **Full-Text Search**: BM25 ranking on name, district, city, address, zip
   - **Queries**: LEFT JOIN pattern for nullable teacher/enrollment data
   - Returns: `School` structs with `sql.Null*` types for missing values

3. **TUI Layer** (`main.go`)
   - **Bubble Tea Elm architecture**: Model → Update → View cycle
   - **Views**: searchView, detailView, agentView, savePromptView
   - **Async operations**: Goroutines send tea.Msg back to Update()
   - **State management**: Single source of truth in `model` struct

4. **Web Layer** (`server.go`, `web_handlers.go`)
   - **Chi router**: RESTful routes and middleware
   - **HTMX patterns**: Partial HTML responses, out-of-band swaps
   - **Streaming responses**: Server-sent events for AI agent
   - **File uploads**: Multipart form data for CSV/Excel import

5. **AI Services**
   - **Data Agent** (`internal/agent/`): Converts natural language to SQL
   - **Web Scraper** (`ai_scraper.go`): Extracts structured data from websites
   - **Caching**: 30-day file-based cache (`.school_cache/`)
   - **Model**: Claude 3.5 Haiku for speed and cost-efficiency

6. **NAEP Integration** (`naep_client.go`)
   - Fetches reading/math scores from NAEP API
   - District-level aggregation (school-level not available)
   - Grade determination based on school level
   - Cached responses to minimize API calls

### Design Patterns

**Database Access:**
- Repository pattern with `DB` struct
- Prepared statements for performance
- Nullable fields with helper methods (`TeachersString()`, `EnrollmentString()`)

**Concurrency:**
- Goroutines for I/O operations (DB, HTTP, AI)
- Channels for result communication
- Context-based timeouts and cancellation

**Error Handling:**
- Structured logging with `slog` (JSON format to `err.log`)
- Graceful degradation (missing AI key, network errors)
- User-friendly error messages in UI

**Caching Strategy:**
- AI data: 30 days, file-based (JSON)
- NAEP data: In-memory for session
- Database: Persistent on disk

### Example Database Queries

**Full-text search with BM25 ranking:**
```sql
SELECT d.*,
       t.TEACHERS,
       e.STUDENT_COUNT,
       fts_main_directory.match_bm25(d.NCESSCH, ?) as score
FROM directory d
LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH
  AND e.TOTAL_INDICATOR = 'Education Unit Total'
WHERE fts_main_directory.match_bm25(d.NCESSCH, ?) IS NOT NULL
ORDER BY score DESC
LIMIT 100
```

**State filtering with indexed lookup:**
```sql
SELECT d.*, t.TEACHERS, e.STUDENT_COUNT
FROM directory d
LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH
WHERE d.ST = ?  -- Uses idx_directory_state
ORDER BY d.SCH_NAME
LIMIT 100
```

## Performance

### Database
- **First-time setup**: ~13 seconds (CSV import + indexing + FTS)
- **Subsequent loads**: <100ms (database already exists)
- **Search queries**:
  - Full-text search (BM25): <10ms
  - Filtered queries with joins: <20ms
  - State filtering: <15ms (indexed)
- **Database size**: 323MB (vs 2.3GB CSV source = 14% compression)

### Memory Usage
- **Binary**: ~70MB compiled
- **Runtime**: ~50MB + DuckDB working memory
- **Dataset**: 102,274 schools, 11.2M enrollment records

### AI Operations
- **Data agent query**: 2-10 seconds (depends on complexity)
- **Website scraping**: 3-7 seconds per school
- **Caching**: 30-day TTL, instant retrieval on cache hit
- **NAEP API**: 1-3 seconds per district (cached for session)

### Network
- **Initial download**: 2.3GB over HTTP (with progress tracking)
- **Web server**: <50ms page load (HTMX partial updates)
- **API responses**: <100ms for JSON endpoints

## Testing

Comprehensive test coverage using **mock data** (no need to download full dataset).

### Run Tests

```bash
# All tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific test suites
go test -v ./...              # All tests
go test -v -run TestDB        # Database tests only
go test -v -run TestTUI       # TUI tests only
go test -v -run TestNAEP      # NAEP client tests

# With race detector
go test -v -race ./...
```

### Test Coverage

| Component | File | Coverage |
|-----------|------|----------|
| Database Layer | `db_test.go` | Search, joins, FTS, null handling |
| TUI Application | `tui_test.go` | State management, views, key handlers |
| Web Handlers | `web_handlers_test.go` | HTTP routes, templates, HTMX |
| NAEP Client | `naep_client_test.go` | API calls, grade logic, caching |

### Mock Data

Tests use 5 mock schools in `testdata/`:
- Lincoln Elementary (CA), Washington High (CA), Jefferson Middle (TX)
- Roosevelt Charter (NY), Madison K-8 (FL)

**Performance**: All tests complete in ~5 seconds with in-memory databases.

**See:** [TESTING.md](TESTING.md) for detailed testing guide and [E2E_TEST_SPEC.md](E2E_TEST_SPEC.md) for end-to-end testing.

## Configuration

### Environment Variables

```bash
# Required for AI features
export ANTHROPIC_API_KEY='sk-ant-...'

# Optional: Custom data directory
export DATA_DIR='/path/to/data'

# Optional: Editor for Ctrl+E (edit cached AI data)
export EDITOR='vim'  # or nano, emacs, code, etc.
```

### Data Directory Structure

```
tmpdata/
├── data.duckdb              # Main database (323MB)
├── err.log                  # Application logs (JSON format)
├── .school_cache/           # AI scraper cache (30-day TTL)
│   └── {NCESSCH}.json       # Cached school data
└── *.csv                    # Optional: Original CSV files (can delete after import)
```

### Configuration Files

- **No config files needed**: All settings via CLI flags or environment variables
- **Cache management**: Automatic cleanup of expired cache entries
- **Logging**: Structured JSON logs to `tmpdata/err.log`

## Development

### Adding Features

**New CLI Command:**
1. Create `cmd/mycommand.go` with Cobra command
2. Register in `cmd/root.go` init()
3. Implement logic using shared database interface
4. Return JSON for consistency

**New TUI View:**
1. Add view constant in `main.go`
2. Create `myViewRender()` method
3. Add key handler `handleMyViewKeys()`
4. Update `Update()` and `View()` methods

**New Web Route:**
1. Add route in `server.go` router setup
2. Create handler in `web_handlers.go`
3. Add template in `templates/`
4. Use HTMX for dynamic updates

**New Database Field:**
1. Update `School` struct in `db.go`
2. Modify SQL queries to include field
3. Add helper method (e.g., `MyFieldString()`)
4. Update views/templates to display field

### Key Dependencies

```go
// Core
github.com/marcboeker/go-duckdb       // DuckDB driver
github.com/spf13/cobra                // CLI framework

// TUI
github.com/charmbracelet/bubbletea    // TUI framework
github.com/charmbracelet/bubbles      // TUI components
github.com/charmbracelet/lipgloss     // Styling
github.com/charmbracelet/glamour      // Markdown rendering

// Web
github.com/go-chi/chi/v5              // HTTP router
html/template                         // Go templates
github.com/xuri/excelize/v2           // Excel import

// AI
github.com/anthropics/anthropic-sdk-go  // Claude API
```

### Debugging

**TUI Mode:**
- Logs written to `tmpdata/err.log` (JSON format)
- Use `tail -f tmpdata/err.log | jq` for real-time logs

**Web Mode:**
- Chi middleware logs all HTTP requests
- Browser DevTools Network tab for HTMX debugging
- Check console for JavaScript errors

**Database:**
- Use `./schoolfinder query "SQL"` to test queries
- Use `./schoolfinder schema` to inspect structure
- Open `tmpdata/data.duckdb` with DuckDB CLI for manual inspection

## Documentation

- **[CLAUDE.md](CLAUDE.md)**: Detailed architecture and development guide
- **[TESTING.md](TESTING.md)**: Comprehensive testing documentation
- **[E2E_TEST_SPEC.md](E2E_TEST_SPEC.md)**: End-to-end testing specifications
- **[AI_SCRAPER_GUIDE.md](AI_SCRAPER_GUIDE.md)**: AI scraper usage and examples (if exists)

## License

MIT License - See LICENSE file for details
