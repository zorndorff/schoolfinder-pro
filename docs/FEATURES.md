# School Finder TUI - Feature Overview

## Current Features

### 🔍 Search & Discovery

**Multi-field Fuzzy Search**
- School name
- District name
- City
- Street address
- ZIP code

**State Filtering**
- Cycle through common states (CA, TX, NY, FL, IL, PA, GA, NJ, NC, OH)
- Press `Ctrl+S` to toggle
- Or search all states

**Interactive List View**
- Shows up to 100 results
- Display format: `[School Name]`
  - `City, ST | District | Students: 1234 | Teachers: 45.5 | NCESSCH_ID`
- Navigate with arrow keys
- Tab to switch focus between search and results

### 📊 Comprehensive School Details

When you select a school, you see **4 organized sections**:

#### 1. Basic Information
```
School Name:     [Name]
NCESSCH ID:      [12-digit ID]
District:        [District Name]
School Type:     Regular School / Charter / Alternative / etc.
Level:           Elementary / Middle / High / Other
Grade Range:     K - 5 / 9 - 12 / Pre-K - 8 / etc.
Charter School:  Yes / No
School Year:     2023-2024
```

#### 2. Location
```
Street Address:  [Full street address, multi-line if needed]
City:            [City]
State:           [State Name (ST)]
Zip Code:        [12345 or 12345-6789]
```

#### 3. Contact
```
Phone:           [(555) 555-5555]
Website:         [https://school.edu]
```

#### 4. Enrollment & Staffing
```
Total Enrollment:  [1,234 students]
Teachers (FTE):    [45.5 full-time equivalent]
Student/Teacher:   [27.1:1 ratio]
```

### ⚡ Performance

- **Search Speed**: < 100ms for most queries
- **Data Loading**: Instant (queries CSV files directly)
- **Memory Usage**: ~50MB binary + minimal runtime overhead
- **Records**: Searches across 100K+ schools

### 🎨 User Interface

**Keyboard Controls:**
- `Tab` - Switch between search input and results list
- `Enter` - Execute search / View selected school
- `Ctrl+S` - Cycle state filter
- `Ctrl+Y` - Copy NCESSCH ID to clipboard
- `Esc` - Return to search (from detail view)
- `Ctrl+C` - Quit application
- `↑↓` - Navigate results list

**Visual Design:**
- Color-coded sections with borders
- Bold labels for easy scanning
- Proper alignment and spacing
- Loading indicators
- Error messages displayed clearly

### 📁 Data Sources

**CCD (Common Core of Data) Files:**
1. `ccd_sch_029_2324_w_1a_073124.csv` - School Directory
   - Basic school information
   - Location data (address, city, state, zip)
   - Contact information (phone, website)
   - School characteristics (type, level, grades, charter status)

2. `ccd_sch_059_2324_l_1a_073124.csv` - Teacher Staffing
   - Full-time equivalent (FTE) teacher counts
   - Data quality flags

3. `ccd_sch_052_2324_l_1a_073124.csv` - Student Enrollment
   - Total student counts
   - Enrollment by various categories

**Data Quality:**
- Handles NULL/missing values gracefully
- Shows "N/A" for unavailable data
- Validates data types (e.g., charter status variations)

### 🔧 Technical Details

**Technology Stack:**
- **Language**: Go 1.25+
- **Database**: DuckDB (in-memory, zero-config)
- **TUI Framework**: Bubble Tea (Elm architecture)
- **UI Components**: Bubbles (list, text input)
- **Styling**: Lipgloss
- **Clipboard**: atotto/clipboard (cross-platform)

**Architecture:**
- Clean separation: DB layer (`db.go`) + UI layer (`main.go`)
- Type-safe NULL handling with `sql.NullString` / `sql.NullFloat64` / `sql.NullInt64`
- Helper methods for formatted output
- LEFT JOINs to combine data from multiple sources
- CSV files read directly (no import/export needed)

**Calculated Fields:**
- Student/Teacher Ratio: `enrollment / teachers`
- Grade Range: Converts CCD codes to readable format
- Charter Status: Normalizes various charter text values

### 🚀 Future Enhancement Ideas

**Potential Features:**
- Export search results to CSV/JSON
- Save favorite schools
- Compare multiple schools side-by-side
- View historical data (if older year files added)
- Advanced filters (enrollment range, school type, etc.)
- Demographic data display (race/ethnicity breakdown)
- Free/reduced lunch statistics
- View by-grade enrollment breakdown
- Map integration (if coordinate data available)
- Bulk operations (compare schools in a district)

**Performance Optimizations:**
- Cache frequently accessed schools
- Preload states list
- Index creation for faster queries
- Pagination for large result sets

**UX Improvements:**
- Sort options (name, enrollment, ratio, etc.)
- Regex search mode
- Bookmark/favorite schools
- Search history
- Custom state filter lists
- Theme customization
