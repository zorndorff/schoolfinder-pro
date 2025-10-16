# Charts & Visualizations Guide

The School Finder TUI now includes rich ASCII-based charts and visualizations to help you understand school data at a glance.

## 📊 Available Visualizations

### 1. Bar Charts (Horizontal)
Display numeric values as horizontal bars with color coding.

**Features:**
- Colored bars with filled (█) and empty (░) segments
- Value labels
- Customizable width and colors
- Percentage-based scaling

**Used For:**
- Enrollment comparison
- Teacher count comparison
- Relative metrics

**Example:**
```
Enrollment      ████████████████████░░░░░░░░░░░░░░░░░░░░ 850
Teachers (FTE)  ██████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 35.5
```

### 2. Ratio Indicator
Visual scale showing where a school's student/teacher ratio falls.

**Features:**
- 4-zone scale (Excellent, Good, Average, High)
- Diamond marker (◆) showing current position
- Color-coded zones:
  - 🟢 Green: Low ratio (excellent)
  - 🟡 Yellow: Medium ratio (good/average)
  - 🔴 Red: High ratio (concerning)
- Labeled benchmarks

**Benchmark Zones:**
- **Excellent**: 0-15 students per teacher
- **Good**: 15-20 students per teacher
- **Average**: 20-25 students per teacher
- **High**: 25+ students per teacher

**Example:**
```
───┃─────────┃─────────◆─────┃──────────┃───
Excellent     Good      Average      High
Current Ratio: 18.5:1
```

### 3. Summary Statistics Bar
Shows aggregate statistics for search results.

**Displays:**
- Total number of schools found
- Average enrollment across results
- Average teacher count across results

**Example:**
```
Results: 45 schools | Avg Enrollment: 623 | Avg Teachers: 32.4
```

## 📈 Visualization Locations

### Search Results View
**Top of Results:**
- Summary statistics bar showing aggregated metrics
- Helps understand the dataset before diving into details

### School Detail View
**Metrics Visualization Section:**

Appears after the Enrollment & Staffing section when data is available.

**Includes:**
1. **Enrollment Bar Chart**
   - Compares school enrollment to national average (~500)
   - Max scale: 1000 students
   - Color: Cyan (#33)

2. **Teachers Bar Chart**
   - Compares teacher count to average (~30 FTE)
   - Max scale: 60 teachers
   - Color: Magenta (#201)

3. **Student/Teacher Ratio Analysis**
   - Visual ratio indicator
   - Current ratio display
   - Benchmarked against:
     - Low benchmark: 15:1
     - High benchmark: 25:1

## 🎨 Color Coding

### Standard Colors:
- **Cyan (33)**: Primary data, enrollment
- **Magenta (201)**: Secondary data, teachers
- **Green (82)**: Positive/good values
- **Yellow (226)**: Moderate/warning values
- **Orange (214)**: Caution values
- **Red (196)**: High/concerning values
- **Blue (62)**: Headers and titles
- **Gray (240-241)**: Labels and help text

### Contextual Colors:
- **Student/Teacher Ratio**: Lower is better (green), higher is concerning (red)
- **Enrollment**: Relative to average, no value judgment
- **Teachers**: Relative to average, no value judgment

## 🔧 Chart Components

### Built-in Chart Functions (charts.go)

**BarChart()**
- Horizontal bar with value comparison
- Parameters: label, value, max, width, color

**RatioIndicator()**
- Visual scale with position marker
- Parameters: ratio, benchmarkLow, benchmarkHigh

**PercentageBar()**
- Percentage-based progress bar
- Auto-color based on percentage thresholds

**GaugeChart()**
- Circular-style gauge indicator
- Position-based visualization

**Sparkline()**
- Mini line chart from array of values
- Characters: ▁▂▃▄▅▆▇█

**BoxPlot()**
- Statistical distribution visualization
- Shows min, Q1, median, Q3, max

**ComparisonBar()**
- Two-value side-by-side comparison
- Shows percentage split

**MetricCard()**
- Bordered card with title, value, subtitle
- Optional percentage bar

**InfoBox()**
- Small bordered box for key metrics
- Label + value layout

## 📐 Chart Dimensions

**Standard Widths:**
- Bar charts: 40 characters
- Ratio indicator: 40 characters
- Full-width displays: Adapt to terminal size

**Responsive Design:**
- Charts scale with terminal size
- Minimum viable display at 80x24 terminal
- Optimal display at 120x40 or larger

## 🎯 Interpretation Guide

### Enrollment Bar
- **Short bar (<25%)**: Small school (under 250 students)
- **Medium bar (25-75%)**: Average school (250-750 students)
- **Long bar (>75%)**: Large school (750+ students)

### Teachers Bar
- **Short bar (<25%)**: Small staff (under 15 teachers)
- **Medium bar (25-75%)**: Average staff (15-45 teachers)
- **Long bar (>75%)**: Large staff (45+ teachers)

### Ratio Indicator
- **Left side (Green zone)**: Excellent ratio, more individualized attention
- **Middle (Yellow zone)**: Average ratio, typical for most schools
- **Right side (Red zone)**: High ratio, may indicate overcrowding

## 💡 Future Chart Ideas

**Potential Additions:**
- Grade distribution bars (by grade level)
- Enrollment trend over time (if historical data available)
- Comparison with district averages
- Demographic distribution charts
- Free/reduced lunch percentage bars
- School type distribution (in search results)
- Geographic clustering visualization
- Teacher experience distribution
- Student achievement indicators
- Resource allocation charts

**Interactive Features:**
- Toggle between absolute and relative views
- Customizable benchmark values
- Export charts as text/ASCII art
- Chart filtering options
- Multi-school comparison mode

## 🚀 Technical Implementation

**Library:** Pure Go + Lipgloss styling
- No external charting dependencies
- ASCII/Unicode box-drawing characters
- ANSI color support
- Terminal-native rendering

**Performance:**
- Instant rendering (no network calls)
- Minimal CPU overhead
- Scales to large datasets
- Efficient string building

**Data Flow:**
1. Query DuckDB for school data
2. Calculate metrics and statistics
3. Generate chart strings with styling
4. Render in TUI view

## 📝 Examples in Action

### Small Elementary School
```
📊 Metrics Visualization

Enrollment      ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 245
Teachers (FTE)  ███████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 18.5

Student/Teacher Ratio Analysis:
───────◆──────┃──────────┃──────────┃──────────┃───
Excellent     Good      Average      High
Current Ratio: 13.2:1
```

### Large High School
```
📊 Metrics Visualization

Enrollment      ███████████████████████████████████░░░░░ 1650
Teachers (FTE)  ████████████████████████████░░░░░░░░░░░░ 87.3

Student/Teacher Ratio Analysis:
───┃──────────┃──────◆───┃──────────┃──────────┃───
Excellent     Good      Average      High
Current Ratio: 18.9:1
```

### Charter School with High Ratio
```
📊 Metrics Visualization

Enrollment      █████████████████░░░░░░░░░░░░░░░░░░░░░░░ 425
Teachers (FTE)  ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 14.2

Student/Teacher Ratio Analysis:
───┃──────────┃──────────┃──────────┃─────────◆┃───
Excellent     Good      Average      High
Current Ratio: 29.9:1
```

## 🔍 Tips for Using Charts

1. **Compare Schools**: Search for multiple schools in a district to see relative sizes
2. **Identify Outliers**: Look for very short or very long bars
3. **Check Ratios**: Green zone ratios typically indicate better student support
4. **Context Matters**: Small schools may have different ratios than large schools
5. **Use Filters**: Filter by state to see regional patterns
