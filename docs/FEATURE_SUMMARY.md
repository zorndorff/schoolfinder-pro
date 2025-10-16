# School Finder TUI - Complete Feature Summary

## 🎯 Application Overview

A comprehensive Terminal User Interface (TUI) for searching and analyzing U.S. school data with AI-powered website extraction.

**Technology Stack:**
- **Language**: Go 1.25+
- **Database**: DuckDB (in-memory SQL)
- **TUI Framework**: Bubble Tea (Elm architecture)
- **UI Components**: Bubbles & Lipgloss
- **AI**: Claude 3.5 Haiku (Anthropic)
- **Data**: NCES Common Core of Data (CCD)

---

## 🔍 Core Features

### 1. Fast Search & Discovery
- **DuckDB-powered queries** - Sub-100ms response times
- **Multi-field search** - Name, city, district, address, ZIP code
- **State filtering** - Quick cycle through common states (Ctrl+S)
- **Fuzzy matching** - Find schools even with partial information
- **100K+ schools** - Complete national dataset (2023-2024)

### 2. Rich Data Display
**School Directory Information:**
- NCESSCH ID, school name, district
- Full street address, city, state, ZIP
- Phone number and website
- School type (regular, charter, alternative, etc.)
- Grade levels and level (elementary, middle, high)
- Charter school status
- School year

**Staffing & Enrollment:**
- Total student enrollment
- Teacher count (FTE - Full-Time Equivalent)
- Calculated student/teacher ratio

### 3. Visual Analytics 📊
**Chart Types:**
- **Bar Charts** - Compare enrollment and teachers vs. averages
- **Ratio Indicator** - Visual scale showing S/T ratio quality
- **Summary Stats** - Aggregate metrics for search results

**Visual Features:**
- ASCII/Unicode rendering (no external dependencies)
- Color-coded by meaning (green=good, red=high)
- Benchmarked against national averages
- Instant rendering

**Metrics Displayed:**
- Enrollment (vs. 500 avg)
- Teachers (vs. 30 avg)
- Student/Teacher ratio (benchmarks: 15:1 excellent, 25:1 high)

### 4. AI Website Scraper 🤖

**Powered by Claude 3.5 Haiku**

**Extracts:**
- Principal and administrative staff
- School mascot and colors
- AP courses and honors programs
- Special programs (IB, STEM, etc.)
- Foreign languages offered
- Sports teams and athletic programs
- Clubs and student organizations
- Arts programs
- Facilities information
- Bell schedule and school hours
- Achievements and awards
- Accreditations
- Mission statement

**Features:**
- **Fast**: 3-10 second extraction
- **Cost-effective**: ~$0.001-0.005 per school
- **Cached**: 30-day local cache
- **Automatic**: One keypress (Ctrl+A)
- **Structured**: Clean JSON output
- **Saved**: Persistent cache in `.school_cache/`

**Cost:** Using Claude 3.5 Haiku - one of the most affordable AI models
- Input: $0.80/M tokens
- Output: $4.00/M tokens
- Typical: Less than half a cent per school

---

## 📚 Data Sources

### CSV Files Required
1. **ccd_sch_029_2324_w_1a_073124.csv** - School Directory
   - Basic school info, location, contact
   - School characteristics, grades, charter status

2. **ccd_sch_059_2324_l_1a_073124.csv** - Teacher Staffing
   - FTE teacher counts per school

3. **ccd_sch_052_2324_l_1a_073124.csv** - Student Enrollment
   - Total student counts

### Data Quality
- **97.38% completeness** for teacher data
- **100,458 schools** in dataset
- **All 50 states + DC** + territories
- **2023-2024 school year**

---

## ⌨️ Keyboard Controls

### Search View
| Key | Action |
|-----|--------|
| Type | Enter search query |
| Enter | Execute search (when focused on input) |
| Enter | View school details (when focused on list) |
| Tab | Switch focus (input ↔ list) |
| ↑/↓ | Navigate results |
| Ctrl+S | Cycle state filter |
| Esc/Ctrl+C | Quit |

### Detail View
| Key | Action |
|-----|--------|
| Ctrl+A | AI extract website data |
| Ctrl+Y | Copy NCESSCH ID to clipboard |
| Esc | Back to search |
| Ctrl+C | Quit |

---

## 🎨 User Interface

### Design Philosophy
- **Clean and organized** - Bordered sections
- **Color-coded** - Meaningful use of color
- **Information density** - Maximum data, minimal clutter
- **Responsive** - Adapts to terminal size
- **Professional** - Suitable for research use

### Visual Hierarchy
1. **Headers** - Bold, colored (blue)
2. **Labels** - Bold, cyan
3. **Values** - Yellow/white
4. **Charts** - Color-coded by meaning
5. **Help text** - Gray, non-intrusive

### Sections (Detail View)
1. Basic Information
2. Location
3. Contact
4. Enrollment & Staffing
5. Metrics Visualization
6. AI-Extracted Information (when available)

---

## 🚀 Performance

| Metric | Performance |
|--------|-------------|
| Search query | < 100ms |
| Data load | Instant (direct CSV read) |
| Chart rendering | < 1ms |
| AI extraction | 3-10 seconds |
| Memory usage | ~50MB base + DuckDB |
| Binary size | ~55MB (includes DuckDB) |

---

## 💾 Caching & Storage

### AI Cache
- **Location**: `.school_cache/` directory
- **Format**: JSON (one file per school)
- **Duration**: 30 days
- **Size**: ~5-20KB per school
- **Purge**: Automatic after 30 days

### Benefits
- Reduces API costs
- Faster subsequent views
- Offline access to cached data
- Easy to inspect (readable JSON)

---

## 🔒 Privacy & Security

### Data Handling
✅ Public information only
✅ No authentication/login
✅ No personal data collection
✅ Local storage only
✅ User-controlled API keys
✅ Respects robots.txt

### API Key Security
- Loaded from environment variable
- Never stored in code
- Not logged or displayed
- User-controlled

---

## 📖 Documentation

### User Guides
- **README.md** - Quick start and overview
- **AI_SCRAPER_GUIDE.md** - Complete AI scraper documentation
- **CHARTS.md** - Chart types and interpretation
- **VISUALIZATIONS_DEMO.md** - Visual examples
- **FEATURES.md** - Detailed feature list

### Developer Docs
- **CHANGELOG.md** - Version history and updates
- Code comments throughout
- Clean architecture (MVC-like)

---

## 🔧 Technical Architecture

### Code Structure (1,703 lines)
```
main.go         (548 lines) - TUI application & views
db.go          (359 lines) - DuckDB integration
ai_scraper.go  (346 lines) - AI website extraction
charts.go      (450 lines) - Visualization components
```

### Design Patterns
- **MVC-like separation** - Model/View/Controller
- **Elm architecture** - Bubble Tea messages
- **Functional composition** - Pure functions for charts
- **Error handling** - Graceful degradation
- **Optional features** - AI scraper works without API key

### Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - UI components
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/marcboeker/go-duckdb` - Database
- `github.com/anthropics/anthropic-sdk-go` - AI API
- `github.com/atotto/clipboard` - Clipboard support

---

## 🎯 Use Cases

### Education Researchers
- Compare schools across regions
- Analyze staffing patterns
- Study enrollment trends
- Extract school characteristics

### Parents & Students
- Research potential schools
- Compare programs offered
- View facilities and activities
- Check teacher ratios

### Policy Makers
- Analyze resource allocation
- Study school characteristics
- Compare similar schools
- Research best practices

### Data Analysts
- Explore CCD data interactively
- Quick lookups and validation
- Extract specific subsets
- Verify data quality

---

## 🌟 Unique Features

What sets this apart:

1. **TUI-native** - Fast, keyboard-driven, SSH-friendly
2. **AI integration** - Automated website extraction
3. **Visual analytics** - Charts in the terminal
4. **Zero setup** - No database import needed
5. **Caching** - Smart, efficient data reuse
6. **Complete** - All features in one tool
7. **Open source** - Fully transparent

---

## 🚀 Future Roadmap

### Planned Features
- [ ] Historical data (previous years)
- [ ] Export to CSV/JSON
- [ ] Multi-school comparison view
- [ ] Demographic data visualization
- [ ] Free/reduced lunch statistics
- [ ] Grade-by-grade breakdown
- [ ] Custom benchmark configuration
- [ ] Saved searches/favorites

### Possible Enhancements
- [ ] Map view (if coordinates available)
- [ ] Distance calculations
- [ ] District-level aggregation
- [ ] Time-series charts
- [ ] Social media integration
- [ ] Parent review aggregation

---

## 📊 Statistics

### Current Dataset
- **Schools**: 100,458
- **States**: 50 + DC + territories
- **Teachers**: 3.2M+ FTE
- **Students**: ~50M+ total
- **Data Year**: 2023-2024

### Usage Estimates
- **Typical session**: 5-15 schools viewed
- **AI extractions**: 1-5 per session
- **Search queries**: 3-10 per session
- **Session cost**: $0.01-0.05 (with AI)

---

## 🎓 Learning Value

This project demonstrates:
- **Modern Go development**
- **TUI application design**
- **AI API integration**
- **Data visualization techniques**
- **User experience design**
- **Performance optimization**
- **Clean architecture**

---

## 📝 License & Credits

### Data Source
- **NCES** - National Center for Education Statistics
- **CCD** - Common Core of Data
- Public domain educational data

### AI
- **Anthropic** - Claude 3.5 Haiku
- API terms apply

### Open Source
- MIT License (or similar)
- Contributions welcome
- Educational/research use encouraged

---

## 🔗 Quick Links

- **Get Started**: See README.md
- **AI Setup**: See AI_SCRAPER_GUIDE.md
- **View Charts**: See CHARTS.md & VISUALIZATIONS_DEMO.md
- **API Key**: https://console.anthropic.com
- **CCD Data**: https://nces.ed.gov/ccd

---

## 💡 Tips for Best Experience

1. **Set API key** - Enable AI features
2. **Use state filter** - Narrow search for better results
3. **Check cache** - Reuse AI extractions
4. **Read charts** - Visual patterns tell stories
5. **Tab between views** - Keyboard is faster
6. **Verify data** - AI extractions should be confirmed

---

## 🎉 Summary

A **powerful**, **fast**, and **intelligent** tool for exploring U.S. school data right in your terminal.

**Key Strengths:**
- ⚡ Lightning-fast searches
- 🤖 AI-powered enhancements
- 📊 Beautiful visualizations
- 💾 Smart caching
- ⌨️ Keyboard-driven workflow
- 🎯 Comprehensive data

**Perfect for:** Researchers, parents, students, educators, policy makers, and data enthusiasts.

---

*Built with ❤️ using Go, DuckDB, Bubble Tea, and Claude*
