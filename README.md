# School Finder TUI

A terminal user interface (TUI) application for searching and exploring school data from the Common Core of Data (CCD) using DuckDB and Charmbracelet's Bubble Tea framework.

![Demo](./docs/demo.gif)

## Features

- **Fast Search**: Powered by DuckDB for instant queries across 100K+ schools
- **Interactive TUI**: Beautiful terminal interface built with Bubble Tea and Lipgloss
- **Fuzzy Filtering**: Search by school name, city, district, address, or zip code
- **State Filtering**: Filter results by state (Ctrl+S to cycle)
- **📊 Data Visualizations**: ASCII charts and graphs for enrollment, staffing, and ratios
- **📈 Summary Statistics**: Aggregate metrics displayed for search results
- **🤖 AI Website Scraper**: Extract additional details from school websites using Claude 3.5 Haiku
- **💾 Smart Caching**: 30-day cache for AI-extracted data
- **Detailed View**: View comprehensive school information including:
  - **Basic Information**: School name, ID (NCESSCH), district, type, level
  - **Academic Details**: Grade range (e.g., 9-12), charter status, school year
  - **Location**: Full street address, city, state, zip code
  - **Contact**: Phone number and website
  - **Enrollment & Staffing**: Total student enrollment, teacher count (FTE), student/teacher ratio
- **Clipboard Integration**: Copy school IDs to clipboard (Ctrl+Y)

## Building

```bash
go build -o schoolfinder
```

## Running

### Basic Usage
```bash
# Run from the data directory
./schoolfinder

# Or specify the data directory
./schoolfinder /path/to/csv/files
```

### With AI Scraper (Optional)
```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY='sk-ant-your-key-here'

# Run the application
./schoolfinder
```

The AI scraper will be automatically enabled when the API key is detected.

**Get an API key:** [console.anthropic.com](https://console.anthropic.com)

**See:** [AI_SCRAPER_GUIDE.md](AI_SCRAPER_GUIDE.md) for full documentation

## Usage

### Search View

- **Type to search**: Enter school name, city, or district
- **Tab**: Switch focus between search input and results list
- **Enter**:
  - When search input is focused: Execute search
  - When list is focused: View detailed information
- **Ctrl+S**: Cycle through state filters (All States, CA, TX, NY, FL, etc.)
- **Arrow keys**: Navigate results (when list is focused)
- **Esc/Ctrl+C**: Quit application

### Detail View

- **Ctrl+A**: AI Extract Website Data (if API key is set)
- **Ctrl+Y**: Copy NCESSCH ID to clipboard
- **Esc**: Return to search view
- **Ctrl+C**: Quit application

## Architecture

### Files

- `main.go`: Main TUI application, view rendering, and event handling
- `db.go`: DuckDB integration and data access layer
- `go.mod`: Go module dependencies

### Key Dependencies

- **github.com/marcboeker/go-duckdb**: DuckDB database driver (v2)
- **github.com/charmbracelet/bubbletea**: TUI framework (Elm architecture)
- **github.com/charmbracelet/bubbles**: Reusable TUI components
- **github.com/charmbracelet/lipgloss**: Style and layout library
- **github.com/atotto/clipboard**: Cross-platform clipboard access

### Data Files

The application expects these CSV files in the data directory:

- `ccd_sch_029_2324_w_1a_073124.csv`: School directory (2023-2024)
- `ccd_sch_059_2324_l_1a_073124.csv`: Teacher counts (2023-2024)
- `ccd_sch_052_2324_l_1a_073124.csv`: Student enrollment (2023-2024)

## Design Patterns

### MVC-like Architecture
- **Model**: `model` struct containing application state
- **View**: `View()`, `searchViewRender()`, `detailView()` methods
- **Controller**: `Update()` method handling messages and state transitions

### Async Operations
- Database queries run in goroutines and communicate via messages
- Non-blocking UI updates with loading indicators

### State Management
- Two main views: `searchView` and `detailView`
- State transitions handled via `currentView` field
- Bubble Tea Elm architecture for predictable state updates

## Example Queries

The DuckDB integration supports:

```sql
-- Search with filters
SELECT d.*, t.TEACHERS
FROM directory d
LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
WHERE LOWER(d.SCH_NAME) LIKE '%elementary%'
  AND d.ST = 'CA'
ORDER BY d.SCH_NAME
LIMIT 100
```

## Performance

- Initial load: Instant (in-memory DuckDB)
- Search queries: < 100ms for filtered results
- Memory usage: ~50MB binary + DuckDB overhead
- Data files: Read directly from CSV (no import needed)

## Shell Script Alternative

For a simpler command-line alternative, see `find-school.sh` which provides similar functionality using bash, DuckDB CLI, and fzf.
