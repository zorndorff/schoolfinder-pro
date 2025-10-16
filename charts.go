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
