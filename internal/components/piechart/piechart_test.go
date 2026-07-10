package piechart

import (
	"context"
	"strings"
	"testing"
)

func mockLanguageBreakdown() []DataPoint {
	return []DataPoint{
		{Label: "Go", Value: 450},
		{Label: "Python", Value: 250},
		{Label: "JavaScript", Value: 150},
		{Label: "Rust", Value: 100},
		{Label: "TypeScript", Value: 80},
		{Label: "Ruby", Value: 60},
		{Label: "Java", Value: 50},
		{Label: "C", Value: 40},
		{Label: "C++", Value: 35},
		{Label: "Swift", Value: 30},
		{Label: "Kotlin", Value: 25},
		{Label: "PHP", Value: 20},
		{Label: "Scala", Value: 15},
		{Label: "Elixir", Value: 12},
		{Label: "Haskell", Value: 10},
		{Label: "Zig", Value: 8},
		{Label: "Lua", Value: 5},
	}
}

func TestParseData_Empty(t *testing.T) {
	if parseData(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
	if parseData([]DataPoint{}) != nil {
		t.Fatal("expected nil for empty input")
	}
	if parseData([]DataPoint{{Label: "Go", Value: 0}}) != nil {
		t.Fatal("expected nil for zero-value input")
	}
}

func TestParseData_SingleItem(t *testing.T) {
	entries := parseData([]DataPoint{{Label: "Go", Value: 100}})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Label != "Go" {
		t.Errorf("expected Label 'Go', got %q", e.Label)
	}
	if e.Value != 100 {
		t.Errorf("expected Value 100, got %d", e.Value)
	}
	if e.Percentage != 100.0 {
		t.Errorf("expected Percentage 100, got %.2f", e.Percentage)
	}
	if e.StartPct != 0 {
		t.Errorf("expected StartPct 0, got %d", e.StartPct)
	}
	if e.EndPct != 100 {
		t.Errorf("expected EndPct 100, got %d", e.EndPct)
	}
	if e.OtherValue != 0 {
		t.Errorf("expected OtherValue 0, got %d", e.OtherValue)
	}
}

func TestParseData_MultipleItems(t *testing.T) {
	points := []DataPoint{
		{Label: "Go", Value: 45},
		{Label: "Python", Value: 25},
		{Label: "JavaScript", Value: 15},
		{Label: "Rust", Value: 10},
		{Label: "Other", Value: 5},
	}
	entries := parseData(points)
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Check first entry
	if e := entries[0]; e.Label != "Go" || e.Value != 45 || int(e.Percentage) != 45 {
		t.Errorf("first entry: %+v", entries[0])
	}

	// Check last entry (Other)
	if e := entries[4]; e.Label != "Other" || e.Value != 5 || int(e.Percentage) != 5 {
		t.Errorf("last entry: %+v", entries[4])
	}
}

func TestParseData_Overflow(t *testing.T) {
	points := mockLanguageBreakdown()
	entries := parseData(points)

	// 17 items, 10 colors → 7 overflow items → 1 "Other" entry
	// Entries: 10 (first 10) + 1 (Other) = 11
	if len(entries) != 11 {
		t.Fatalf("expected 11 entries (10 + 1 Other), got %d", len(entries))
	}

	// Check that the last entry is "Other" with summed value
	other := entries[10]
	if other.Label != "Other" {
		t.Errorf("expected last entry label 'Other', got %q", other.Label)
	}
	// Sum of overflowed: 25+20+15+12+10+8+5 = 95
	if other.OtherValue != 95 {
		t.Errorf("expected OtherValue 95, got %d", other.OtherValue)
	}
	if other.Value != 0 {
		t.Errorf("expected Value 0 for Other entry, got %d", other.Value)
	}

	// Check that the 10th entry (index 9, "Swift") has the 10th color
	swift := entries[9]
	if swift.Label != "Swift" {
		t.Errorf("expected entry[9] label 'Swift', got %q", swift.Label)
	}
	if swift.Value != 30 {
		t.Errorf("expected entry[9] Value 30, got %d", swift.Value)
	}
}

func TestParseData_PreservesColors(t *testing.T) {
	points := []DataPoint{
		{Label: "A", Value: 1},
		{Label: "B", Value: 1},
		{Label: "C", Value: 1},
		{Label: "D", Value: 1},
		{Label: "E", Value: 1},
		{Label: "F", Value: 1},
		{Label: "G", Value: 1},
		{Label: "H", Value: 1},
		{Label: "I", Value: 1},
		{Label: "J", Value: 1},
		{Label: "K", Value: 1},
	}
	entries := parseData(points)

	// First 10 should have chart colors
	for i := 0; i < 10; i++ {
		if entries[i].Color == "#6b7280" {
			t.Errorf("entry[%d] should have a chart color, got overflow color", i)
		}
	}

	// Entry 10 (index 10, "K") should have overflow color
	if entries[10].Color != "#6b7280" {
		t.Errorf("entry[10] should have overflow color '#6b7280', got %s", entries[10].Color)
	}
}

func TestPieChartCircle_NoData(t *testing.T) {
	var buf strings.Builder
	err := pieChartCircle([]DataPoint{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no data, got %q", buf.String())
	}
}

func TestPieChartCircle_WithData(t *testing.T) {
	points := []DataPoint{
		{Label: "Go", Value: 70},
		{Label: "Python", Value: 30},
	}
	var buf strings.Builder
	err := pieChartCircle(points).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "conic-gradient") {
		t.Errorf("expected conic-gradient in output, got %q", output)
	}
	if !strings.Contains(output, "#a21caf") {
		t.Errorf("expected july-700 color in output, got %q", output)
	}
}

func TestColoredLegend_NoData(t *testing.T) {
	var buf strings.Builder
	err := coloredLegend([]DataPoint{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no data, got %q", buf.String())
	}
}

func TestColoredLegend_WithMockData(t *testing.T) {
	points := mockLanguageBreakdown()
	var buf strings.Builder
	err := coloredLegend(points).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	// Should contain Go entry
	if !strings.Contains(output, "Go: 450") {
		t.Errorf("expected 'Go: 450' in output, got %q", output)
	}

	// Should contain Swift (10th item, last with chart color)
	if !strings.Contains(output, "Swift: 30") {
		t.Errorf("expected 'Swift: 30' in output, got %q", output)
	}

	// Should contain Other with summed value (25+20+15+12+10+8+5 = 95)
	if !strings.Contains(output, "Other: 95") {
		t.Errorf("expected 'Other: 95' in output, got %q", output)
	}
}
