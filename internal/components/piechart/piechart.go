package piechart

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// DataPoint is a single data point for the pie chart.
// The consumer passes label + value; the component calculates percentages.
type DataPoint struct {
	Label string
	Value int
}

// chartColors are colors for the pie chart. Slot 1 is july-700, rest are contrasting colors.
var chartColors = []string{
	"#a21caf", // july-700 (slot 1)
	"#f59e0b", // amber-500
	"#10b981", // emerald-500
	"#3b82f6", // blue-500
	"#ef4444", // red-500
	"#ec4899", // pink-500
	"#06b6d4", // cyan-500
	"#f97316", // orange-500
	"#84cc16", // lime-500
	"#8b5cf6", // violet-500
}

// chartEntry holds the parsed and computed data for a single slice of the pie chart.
type chartEntry struct {
	Color       string
	Label       string
	Value       int
	OtherValue  int
	Percentage  float64
	StartPct    int
	EndPct      int
}

// parseData processes data points once, computing percentages, colors, and label overrides.
// Returns nil if total is 0.
func parseData(points []DataPoint) []chartEntry {
	total := 0
	for _, p := range points {
		total += p.Value
	}
	if total == 0 {
		return nil
	}

	maxItems := len(chartColors)
	entries := make([]chartEntry, 0, len(points))
	cumulative := 0.0
	for i, p := range points {
		if i < maxItems {
			color := chartColors[i]
			percentage := float64(p.Value) / float64(total) * 100
			start := int(cumulative)
			cumulative += percentage
			entries = append(entries, chartEntry{
				Color:      color,
				Label:      p.Label,
				Value:      p.Value,
				Percentage: percentage,
				StartPct:   start,
				EndPct:     int(cumulative),
			})
		}
	}

	// Consolidate overflowed items into a single "Other" entry.
	otherValue := 0
	for i := maxItems; i < len(points); i++ {
		otherValue += points[i].Value
	}
	if otherValue > 0 {
		percentage := float64(otherValue) / float64(total) * 100
		start := int(cumulative)
		cumulative += percentage
		entries = append(entries, chartEntry{
			Color:      "#6b7280",
			Label:      "Other",
			Value:      0,
			OtherValue: otherValue,
			Percentage: percentage,
			StartPct:   start,
			EndPct:     int(cumulative),
		})
	}
	return entries
}

func pieChartCircle(points []DataPoint) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		entries := parseData(points)
		if entries == nil {
			return nil
		}
		var parts []string
		for _, e := range entries {
			parts = append(parts, fmt.Sprintf("%s %d%% %d%%", e.Color, e.StartPct, e.EndPct))
		}
		_, err := io.WriteString(w, fmt.Sprintf(`<div class="w-full h-full rounded-full" style="background: conic-gradient(%s);"></div>`, strings.Join(parts, ", ")))
		return err
	})
}

func coloredLegend(points []DataPoint) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		entries := parseData(points)
		if entries == nil {
			return nil
		}
		var parts []string
		for _, e := range entries {
			label := e.Label
			value := e.Value
			if e.OtherValue > 0 {
				value = e.OtherValue
			}
			parts = append(parts, fmt.Sprintf(`<div class="flex items-center gap-2"><div class="w-3 h-3 rounded shrink-0" style="background-color: %s;"></div><span class="text-sm text-gray-300">%s: %d</span></div>`, e.Color, label, value))
		}
		_, err := io.WriteString(w, strings.Join(parts, ""))
		return err
	})
}
