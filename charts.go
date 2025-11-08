package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BarChart creates a horizontal bar chart
func BarChart(label string, value, max float64, width int, color lipgloss.Color) string {
	if max == 0 {
		max = value
	}

	percentage := value / max
	if percentage > 1 {
		percentage = 1
	}

	filledWidth := int(float64(width) * percentage)
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > width {
		filledWidth = width
	}

	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", width-filledWidth)

	barStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return fmt.Sprintf("%s %s%s %.0f",
		label,
		barStyle.Render(filled),
		emptyStyle.Render(empty),
		value,
	)
}

// PercentageBar creates a percentage-based progress bar
func PercentageBar(label string, percentage float64, width int) string {
	if percentage > 100 {
		percentage = 100
	}
	if percentage < 0 {
		percentage = 0
	}

	filledWidth := int(float64(width) * percentage / 100)
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", width-filledWidth)

	// Color based on percentage
	var color lipgloss.Color
	switch {
	case percentage >= 75:
		color = lipgloss.Color("82") // Green
	case percentage >= 50:
		color = lipgloss.Color("226") // Yellow
	case percentage >= 25:
		color = lipgloss.Color("214") // Orange
	default:
		color = lipgloss.Color("196") // Red
	}

	barStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return fmt.Sprintf("%s %s%s %.1f%%",
		label,
		barStyle.Render(filled),
		emptyStyle.Render(empty),
		percentage,
	)
}

// Sparkline creates a simple sparkline from values
func Sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Sparkline characters from bottom to top
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var result strings.Builder
	for _, v := range values {
		var idx int
		if max == min {
			idx = len(chars) / 2
		} else {
			normalized := (v - min) / (max - min)
			idx = int(normalized * float64(len(chars)-1))
		}
		result.WriteRune(chars[idx])
	}

	return result.String()
}

// GaugeChart creates a visual gauge
func GaugeChart(value, max float64, width int) string {
	if max == 0 {
		max = value
	}

	percentage := (value / max) * 100
	if percentage > 100 {
		percentage = 100
	}

	// Determine position in gauge (0-width)
	position := int((percentage / 100) * float64(width))
	if position < 0 {
		position = 0
	}
	if position >= width {
		position = width - 1
	}

	// Build gauge
	var gauge strings.Builder
	gauge.WriteString("│")
	for i := 0; i < width; i++ {
		if i == position {
			gauge.WriteString("●")
		} else if i < width/4 {
			gauge.WriteString("─")
		} else if i < width/2 {
			gauge.WriteString("─")
		} else if i < 3*width/4 {
			gauge.WriteString("─")
		} else {
			gauge.WriteString("─")
		}
	}
	gauge.WriteString("│")

	// Color based on position
	var color lipgloss.Color
	if percentage < 33 {
		color = lipgloss.Color("82") // Green (low ratio is good)
	} else if percentage < 66 {
		color = lipgloss.Color("226") // Yellow
	} else {
		color = lipgloss.Color("196") // Red (high ratio)
	}

	gaugeStyle := lipgloss.NewStyle().Foreground(color)
	return gaugeStyle.Render(gauge.String())
}

// BoxPlot creates a simple box plot visualization
func BoxPlot(value, min, q1, median, q3, max float64, width int) string {
	// Normalize all values to 0-width range
	normalize := func(v float64) int {
		if max == min {
			return width / 2
		}
		pos := int(((v - min) / (max - min)) * float64(width))
		if pos < 0 {
			pos = 0
		}
		if pos >= width {
			pos = width - 1
		}
		return pos
	}

	minPos := 0
	maxPos := width - 1
	q1Pos := normalize(q1)
	medianPos := normalize(median)
	q3Pos := normalize(q3)
	valuePos := normalize(value)

	// Build the plot
	plot := make([]rune, width)
	for i := range plot {
		plot[i] = ' '
	}

	// Draw whiskers
	for i := minPos; i <= maxPos; i++ {
		if plot[i] == ' ' {
			plot[i] = '─'
		}
	}

	// Draw box
	for i := q1Pos; i <= q3Pos; i++ {
		if i == q1Pos || i == q3Pos {
			plot[i] = '│'
		} else {
			plot[i] = '█'
		}
	}

	// Draw median
	if medianPos >= 0 && medianPos < width {
		plot[medianPos] = '┃'
	}

	// Draw value marker
	if valuePos >= 0 && valuePos < width {
		plot[valuePos] = '●'
	}

	boxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	medianStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("201"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))

	// Color the string
	result := ""
	for _, r := range plot {
		if r == '┃' {
			result += medianStyle.Render(string(r))
		} else if r == '●' {
			result += valueStyle.Render(string(r))
		} else if r == '█' || r == '│' {
			result += boxStyle.Render(string(r))
		} else {
			result += string(r)
		}
	}

	return result
}

// InfoBox creates a styled info box with a value
func InfoBox(label string, value string, color lipgloss.Color) string {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("240")).
		Width(18).
		Align(lipgloss.Left)

	valueStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Width(12).
		Align(lipgloss.Right)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		labelStyle.Render(label),
		valueStyle.Render(value),
	)

	return boxStyle.Render(content)
}

// MetricCard creates a card showing a metric with visual indicator
func MetricCard(title, value, subtitle string, percentage float64) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	valueStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226"))

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(30)

	// Create percentage bar if applicable
	var bar string
	if percentage >= 0 {
		bar = "\n" + PercentageBar("", percentage, 20)
	}

	content := titleStyle.Render(title) + "\n" +
		valueStyle.Render(value) + "\n" +
		subtitleStyle.Render(subtitle) +
		bar

	return cardStyle.Render(content)
}

// ComparisonBar shows two values side by side
func ComparisonBar(label1, label2 string, value1, value2 float64, width int) string {
	total := value1 + value2
	if total == 0 {
		return label1 + " vs " + label2 + ": No data"
	}

	pct1 := (value1 / total) * 100
	pct2 := (value2 / total) * 100

	width1 := int((pct1 / 100) * float64(width))
	width2 := width - width1

	bar1 := strings.Repeat("█", width1)
	bar2 := strings.Repeat("█", width2)

	style1 := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	style2 := lipgloss.NewStyle().Foreground(lipgloss.Color("201"))

	return fmt.Sprintf("%s %s%s %s (%.0f%% vs %.0f%%)",
		label1,
		style1.Render(bar1),
		style2.Render(bar2),
		label2,
		pct1,
		pct2,
	)
}

// RatioIndicator creates a visual ratio indicator
func RatioIndicator(ratio float64, benchmarkLow, benchmarkHigh float64) string {
	// Create a visual scale
	width := 40

	// Determine position
	var percentage float64
	if ratio <= benchmarkLow {
		percentage = 25
	} else if ratio >= benchmarkHigh {
		percentage = 75
	} else {
		// Interpolate between low and high
		range_ := benchmarkHigh - benchmarkLow
		offset := ratio - benchmarkLow
		percentage = 25 + (offset/range_)*50
	}

	position := int((percentage / 100) * float64(width))
	if position < 0 {
		position = 0
	}
	if position >= width {
		position = width - 1
	}

	// Build indicator
	indicator := make([]rune, width)
	for i := range indicator {
		if i == width/4 || i == width/2 || i == 3*width/4 {
			indicator[i] = '┃'
		} else {
			indicator[i] = '─'
		}
	}

	indicator[position] = '◆'

	// Color coding
	var color lipgloss.Color
	if percentage < 33 {
		color = lipgloss.Color("82") // Green - low ratio is good
	} else if percentage < 66 {
		color = lipgloss.Color("226") // Yellow
	} else {
		color = lipgloss.Color("196") // Red - high ratio
	}

	style := lipgloss.NewStyle().Foreground(color)

	labels := fmt.Sprintf("\n%8s %8s %8s %8s",
		"Excellent",
		"Good",
		"Average",
		"High",
	)

	return style.Render(string(indicator)) + "\n" + labels
}

// CreateEnrollmentVisualization creates a visual representation of enrollment distribution
func CreateEnrollmentVisualization(enrollment int64, avgEnrollment float64) string {
	if avgEnrollment == 0 {
		avgEnrollment = 500 // Default assumption
	}

	percentage := (float64(enrollment) / avgEnrollment) * 100

	// Cap at 200% for visualization
	if percentage > 200 {
		percentage = 200
	}

	width := 50
	return BarChart("School vs Average", float64(enrollment), avgEnrollment*2, width, lipgloss.Color("33"))
}

// DistributionBar shows multiple segments
func DistributionBar(segments []struct {
	Label string
	Value float64
	Color lipgloss.Color
}, width int) string {
	total := 0.0
	for _, seg := range segments {
		total += seg.Value
	}

	if total == 0 {
		return "No data"
	}

	var bar strings.Builder
	remaining := width

	for i, seg := range segments {
		segWidth := int(math.Round((seg.Value / total) * float64(width)))

		// Adjust last segment to fill exactly
		if i == len(segments)-1 {
			segWidth = remaining
		}

		if segWidth > remaining {
			segWidth = remaining
		}

		style := lipgloss.NewStyle().Foreground(seg.Color)
		bar.WriteString(style.Render(strings.Repeat("█", segWidth)))
		remaining -= segWidth
	}

	return bar.String()
}

// NAEPAchievementBar shows NAEP achievement level distribution
func NAEPAchievementBar(label string, belowBasic, atBasic, atProficient, atAdvanced float64, width int) string {
	// Calculate percentages (should sum to ~100%)
	total := belowBasic + atBasic + atProficient + atAdvanced

	// If no data, return empty
	if total == 0 {
		return label + " " + strings.Repeat("░", width) + " (No data)"
	}

	// Calculate widths for each segment
	belowBasicWidth := int(math.Round((belowBasic / 100.0) * float64(width)))
	atBasicWidth := int(math.Round((atBasic / 100.0) * float64(width)))
	atProficientWidth := int(math.Round((atProficient / 100.0) * float64(width)))
	atAdvancedWidth := width - belowBasicWidth - atBasicWidth - atProficientWidth // Ensure exact width

	// Styles for each level
	belowBasicStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))   // Red
	atBasicStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))      // Orange
	atProficientStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Yellow
	atAdvancedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))    // Green

	bar := belowBasicStyle.Render(strings.Repeat("█", belowBasicWidth)) +
		atBasicStyle.Render(strings.Repeat("█", atBasicWidth)) +
		atProficientStyle.Render(strings.Repeat("█", atProficientWidth)) +
		atAdvancedStyle.Render(strings.Repeat("█", atAdvancedWidth))

	// Show proficient+ percentage as key metric
	proficientPlus := atProficient + atAdvanced

	return fmt.Sprintf("%-30s %s (%.0f%% Prof+)", label, bar, proficientPlus)
}

// NAEPTrendIndicator shows score change over time with arrow
func NAEPTrendIndicator(change float64) string {
	var arrow string
	var color lipgloss.Color

	if change > 0.5 {
		arrow = "↑"
		color = lipgloss.Color("82") // Green for improvement
	} else if change < -0.5 {
		arrow = "↓"
		color = lipgloss.Color("196") // Red for decline
	} else {
		arrow = "→"
		color = lipgloss.Color("240") // Gray for no change
	}

	style := lipgloss.NewStyle().Foreground(color).Bold(true)

	if change > 0 {
		return style.Render(fmt.Sprintf("%s +%.0f", arrow, change))
	} else if change < 0 {
		return style.Render(fmt.Sprintf("%s %.0f", arrow, change))
	} else {
		return style.Render(fmt.Sprintf("%s No change", arrow))
	}
}

// NAEPScoreGauge shows NAEP score on typical scale (0-500)
func NAEPScoreGauge(score float64, width int) string {
	// NAEP scores typically range from ~100-350
	// We'll use 0-500 as the full scale
	minScore := 0.0
	maxScore := 500.0

	if score < minScore {
		score = minScore
	}
	if score > maxScore {
		score = maxScore
	}

	percentage := (score - minScore) / (maxScore - minScore)
	position := int(percentage * float64(width))

	if position < 0 {
		position = 0
	}
	if position >= width {
		position = width - 1
	}

	// Build gauge
	gauge := make([]rune, width)
	for i := range gauge {
		if i == position {
			gauge[i] = '◆'
		} else {
			gauge[i] = '─'
		}
	}

	// Add markers for common benchmarks
	basicPos := int(0.4 * float64(width)) // ~200 score
	profPos := int(0.6 * float64(width))  // ~300 score

	if basicPos >= 0 && basicPos < width && gauge[basicPos] != '◆' {
		gauge[basicPos] = '┆'
	}
	if profPos >= 0 && profPos < width && gauge[profPos] != '◆' {
		gauge[profPos] = '┆'
	}

	// Color based on score level
	var color lipgloss.Color
	switch {
	case score >= 300:
		color = lipgloss.Color("82") // Green - Proficient
	case score >= 250:
		color = lipgloss.Color("226") // Yellow - Basic
	case score >= 200:
		color = lipgloss.Color("214") // Orange - Approaching Basic
	default:
		color = lipgloss.Color("196") // Red - Below Basic
	}

	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(string(gauge))
}

// NAEPProficiencyBreakdown creates a stacked bar showing achievement level distribution
func NAEPProficiencyBreakdown(label string, belowBasic, basic, proficient, advanced float64, width int) string {
	total := belowBasic + basic + proficient + advanced
	if total == 0 {
		return label + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", width)) + " No data"
	}

	// Calculate widths for each segment
	belowBasicWidth := int((belowBasic / total) * float64(width))
	basicWidth := int((basic / total) * float64(width))
	proficientWidth := int((proficient / total) * float64(width))
	advancedWidth := width - belowBasicWidth - basicWidth - proficientWidth // Remaining goes to advanced

	// Ensure at least 1 char for non-zero values
	if belowBasic > 0 && belowBasicWidth == 0 {
		belowBasicWidth = 1
	}
	if basic > 0 && basicWidth == 0 {
		basicWidth = 1
	}
	if proficient > 0 && proficientWidth == 0 {
		proficientWidth = 1
	}
	if advanced > 0 && advancedWidth == 0 {
		advancedWidth = 1
	}

	// Build the bar
	var bar strings.Builder

	// Below Basic (red)
	if belowBasicWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(strings.Repeat("█", belowBasicWidth)))
	}

	// Basic (yellow)
	if basicWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(strings.Repeat("█", basicWidth)))
	}

	// Proficient (light green)
	if proficientWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(strings.Repeat("█", proficientWidth)))
	}

	// Advanced (bright green)
	if advancedWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(strings.Repeat("█", advancedWidth)))
	}

	// Add percentages
	proficientPlus := proficient + advanced
	return fmt.Sprintf("%s %s %.0f%% Prof+", label, bar.String(), proficientPlus)
}

// NAEPTrendChart creates a mini sparkline showing score trends over multiple years
func NAEPTrendChart(scores []float64, years []int, width int) string {
	if len(scores) == 0 || len(years) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No trend data")
	}

	// Find min and max for scaling
	minScore := scores[0]
	maxScore := scores[0]
	for _, score := range scores {
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	// Add some padding
	scoreRange := maxScore - minScore
	if scoreRange == 0 {
		scoreRange = 1
	}

	// Create sparkline characters
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var result strings.Builder
	for i, score := range scores {
		normalized := (score - minScore) / scoreRange
		charIndex := int(normalized * float64(len(sparkChars)-1))
		if charIndex < 0 {
			charIndex = 0
		}
		if charIndex >= len(sparkChars) {
			charIndex = len(sparkChars) - 1
		}

		// Color based on trend
		var color lipgloss.Color
		if i > 0 {
			if score > scores[i-1] {
				color = lipgloss.Color("82") // Green - improving
			} else if score < scores[i-1] {
				color = lipgloss.Color("196") // Red - declining
			} else {
				color = lipgloss.Color("226") // Yellow - stable
			}
		} else {
			color = lipgloss.Color("33")
		}

		result.WriteString(lipgloss.NewStyle().Foreground(color).Render(string(sparkChars[charIndex])))
	}

	// Add year labels
	yearLabels := fmt.Sprintf(" (%d-%d)", years[0], years[len(years)-1])
	result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(yearLabels))

	return result.String()
}

// NAEPSubjectComparison creates a side-by-side comparison of subjects
func NAEPSubjectComparison(mathScore, readingScore, scienceScore float64, width int) string {
	var result strings.Builder

	result.WriteString(lipgloss.NewStyle().Bold(true).Render("Subject Comparison:"))
	result.WriteString("\n")

	// Determine which subject is strongest
	maxScore := math.Max(math.Max(mathScore, readingScore), scienceScore)

	// Math bar
	if mathScore > 0 {
		mathBar := BarChart("  Mathematics ", mathScore, maxScore, width-20, lipgloss.Color("33"))
		if mathScore == maxScore && maxScore > 0 {
			mathBar += " ★"
		}
		result.WriteString(mathBar)
		result.WriteString("\n")
	}

	// Reading bar
	if readingScore > 0 {
		readingBar := BarChart("  Reading     ", readingScore, maxScore, width-20, lipgloss.Color("201"))
		if readingScore == maxScore && maxScore > 0 {
			readingBar += " ★"
		}
		result.WriteString(readingBar)
		result.WriteString("\n")
	}

	// Science bar
	if scienceScore > 0 {
		scienceBar := BarChart("  Science     ", scienceScore, maxScore, width-20, lipgloss.Color("82"))
		if scienceScore == maxScore && maxScore > 0 {
			scienceBar += " ★"
		}
		result.WriteString(scienceBar)
		result.WriteString("\n")
	}

	return result.String()
}

// NAEPParentSummaryCard creates a parent-friendly summary of NAEP performance
func NAEPParentSummaryCard(subject string, grade int, proficientPercent float64, score float64, trend string) string {
	var result strings.Builder

	// Determine performance level
	var performanceLevel string
	var performanceColor lipgloss.Color

	if proficientPercent >= 40 {
		performanceLevel = "Strong Performance"
		performanceColor = lipgloss.Color("82") // Green
	} else if proficientPercent >= 25 {
		performanceLevel = "Moderate Performance"
		performanceColor = lipgloss.Color("226") // Yellow
	} else {
		performanceLevel = "Needs Improvement"
		performanceColor = lipgloss.Color("196") // Red
	}

	// Create card
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	levelStyle := lipgloss.NewStyle().Bold(true).Foreground(performanceColor)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Grade %d %s", grade, strings.Title(subject))))
	result.WriteString("\n")
	result.WriteString(levelStyle.Render(fmt.Sprintf("  %s", performanceLevel)))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  %.0f%% of students are proficient or advanced", proficientPercent))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  Average score: %.0f", score))

	if trend != "" {
		result.WriteString(fmt.Sprintf(" %s", trend))
	}

	return result.String()
}

// NAEPProficiencyLegend creates a legend explaining achievement levels
func NAEPProficiencyLegend() string {
	var result strings.Builder

	result.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240")).Render("Achievement Levels:"))
	result.WriteString("\n  ")
	result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("█"))
	result.WriteString(" Below Basic  ")
	result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("█"))
	result.WriteString(" Basic  ")
	result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("█"))
	result.WriteString(" Proficient  ")
	result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("█"))
	result.WriteString(" Advanced")

	return result.String()
}

// NAEPNationalComparison creates a comparison visualization showing local vs national scores
func NAEPNationalComparison(label string, localScore, nationalScore float64, width int) string {
	var result strings.Builder

	// Determine color based on performance relative to national
	var localColor lipgloss.Color
	var indicator string
	diff := localScore - nationalScore

	if diff >= 5 {
		localColor = lipgloss.Color("46") // Bright green - significantly above
		indicator = "↑↑"
	} else if diff >= 2 {
		localColor = lipgloss.Color("82") // Green - above
		indicator = "↑"
	} else if diff >= -2 {
		localColor = lipgloss.Color("226") // Yellow - near national average
		indicator = "≈"
	} else if diff >= -5 {
		localColor = lipgloss.Color("208") // Orange - below
		indicator = "↓"
	} else {
		localColor = lipgloss.Color("196") // Red - significantly below
		indicator = "↓↓"
	}

	nationalColor := lipgloss.Color("240") // Gray for national baseline

	// Create bars (scale to 500 max for NAEP scores)
	maxScale := 500.0
	localWidth := int(float64(width) * (localScore / maxScale))
	nationalWidth := int(float64(width) * (nationalScore / maxScale))

	if localWidth > width {
		localWidth = width
	}
	if nationalWidth > width {
		nationalWidth = width
	}

	// Format label with padding
	paddedLabel := fmt.Sprintf("%-12s", label)

	// Local score bar
	result.WriteString(fmt.Sprintf("%s %s %s %.0f %s\n",
		paddedLabel,
		lipgloss.NewStyle().Foreground(localColor).Render(strings.Repeat("█", localWidth)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", width-localWidth)),
		localScore,
		lipgloss.NewStyle().Foreground(localColor).Bold(true).Render(indicator),
	))

	// National score bar
	result.WriteString(fmt.Sprintf("%s %s %s %.0f %s",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("National Avg"),
		lipgloss.NewStyle().Foreground(nationalColor).Render(strings.Repeat("█", nationalWidth)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", width-nationalWidth)),
		nationalScore,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("—"),
	))

	return result.String()
}

// NAEPComparisonIndicator shows a compact comparison of local vs national score
func NAEPComparisonIndicator(localScore, nationalScore float64) string {
	diff := localScore - nationalScore

	var color lipgloss.Color
	var symbol string
	var description string

	if diff >= 5 {
		color = lipgloss.Color("46")
		symbol = "↑↑"
		description = fmt.Sprintf("+%.1f above national", diff)
	} else if diff >= 2 {
		color = lipgloss.Color("82")
		symbol = "↑"
		description = fmt.Sprintf("+%.1f above national", diff)
	} else if diff >= -2 {
		color = lipgloss.Color("226")
		symbol = "≈"
		description = "near national average"
	} else if diff >= -5 {
		color = lipgloss.Color("208")
		symbol = "↓"
		description = fmt.Sprintf("%.1f below national", diff)
	} else {
		color = lipgloss.Color("196")
		symbol = "↓↓"
		description = fmt.Sprintf("%.1f below national", diff)
	}

	return fmt.Sprintf("%s %s",
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(symbol),
		lipgloss.NewStyle().Foreground(color).Render(description),
	)
}

// NAEPNationalComparisonCard creates a summary card comparing local proficiency to national
func NAEPNationalComparisonCard(subject string, grade int, localProficient, nationalProficient float64) string {
	var result strings.Builder

	diff := localProficient - nationalProficient

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	result.WriteString(headerStyle.Render(fmt.Sprintf("Grade %d %s vs. National", grade, subject)))
	result.WriteString("\n")

	// Proficiency comparison
	var performanceColor lipgloss.Color
	var performanceLabel string

	if diff >= 10 {
		performanceColor = lipgloss.Color("46")
		performanceLabel = "Well Above National"
	} else if diff >= 5 {
		performanceColor = lipgloss.Color("82")
		performanceLabel = "Above National"
	} else if diff >= -5 {
		performanceColor = lipgloss.Color("226")
		performanceLabel = "Near National Average"
	} else if diff >= -10 {
		performanceColor = lipgloss.Color("208")
		performanceLabel = "Below National"
	} else {
		performanceColor = lipgloss.Color("196")
		performanceLabel = "Well Below National"
	}

	result.WriteString(fmt.Sprintf("  Local Proficient+:    %s\n",
		lipgloss.NewStyle().Foreground(performanceColor).Bold(true).Render(fmt.Sprintf("%.1f%%", localProficient))))
	result.WriteString(fmt.Sprintf("  National Proficient+: %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%.1f%%", nationalProficient))))
	result.WriteString(fmt.Sprintf("  Difference:           %s\n",
		lipgloss.NewStyle().Foreground(performanceColor).Render(fmt.Sprintf("%+.1f%%", diff))))
	result.WriteString(fmt.Sprintf("  Performance:          %s",
		lipgloss.NewStyle().Foreground(performanceColor).Bold(true).Render(performanceLabel)))

	return result.String()
}
