package metrics

import (
	"strings"
	"testing"
)

func TestBuildMetricLLMUserContent_WithSources(t *testing.T) {
	data := map[string]any{
		LanguageKey:  "Python",
		"has_readme": true,
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "README.md", "snippet": "# Hello"},
			},
		},
	}
	out := BuildMetricLLMUserContent("readme", data, "o/r", 6, 1)
	if !strings.Contains(out, "o/r") {
		t.Fatalf("expected repo name in output")
	}
	if !strings.Contains(out, "Python") {
		t.Fatalf("expected language in output")
	}
	if !strings.Contains(out, "README.md") || !strings.Contains(out, "# Hello") {
		t.Fatalf("expected snippet in output")
	}
	if !strings.Contains(out, "What to write") || !strings.Contains(out, "Do not output JSON") {
		t.Fatalf("expected coaching task instructions")
	}
}

func TestBuildMetricLLMUserContent_HeuristicOnly(t *testing.T) {
	data := map[string]any{
		LanguageKey:        "Go",
		"has_ignore_file":  true,
		"has_license_file": false,
		"has_src_dir":      false,
	}
	out := BuildMetricLLMUserContent("structure", data, "o/repo", 3, 1)
	if !strings.Contains(out, "✅ Has .gitignore") {
		t.Fatalf("expected checkmark for has_ignore_file, got:\n%s", out)
	}
	if !strings.Contains(out, "❌ Has LICENSE file") {
		t.Fatalf("expected X for has_license_file, got:\n%s", out)
	}
	if !strings.Contains(out, "3/10") {
		t.Fatalf("expected score in output")
	}
	if !strings.Contains(out, "**Go**") || !strings.Contains(out, "Tailor suggestions") {
		t.Fatalf("expected language-specific instruction, got:\n%s", out)
	}
}

func TestBuildMetricLLMUserContent_PathOnlySource(t *testing.T) {
	data := map[string]any{
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "LICENSE", "snippet": ""},
			},
		},
	}
	out := BuildMetricLLMUserContent("structure", data, "o/r", 5, 1)
	if !strings.Contains(out, "File present") {
		t.Fatalf("expected path-only hint, got:\n%s", out)
	}
}

func TestBuildMetricLLMUserContent_LegacyHugeSnippetTruncated(t *testing.T) {
	huge := strings.Repeat("x", 50000)
	data := map[string]any{
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "README.md", "snippet": huge},
			},
		},
	}
	out := BuildMetricLLMUserContent("readme", data, "o/r", 5, 1)
	if len(out) > maxMetricLLMUserBytes+50 {
		t.Fatalf("expected user prompt capped, got len %d", len(out))
	}
}

func TestBuildMetricLLMUserContent_NoLanguage(t *testing.T) {
	data := map[string]any{"has_readme": true}
	out := BuildMetricLLMUserContent("readme", data, "o/r", 5, 1)
	if strings.Contains(out, "Primary language") {
		t.Fatalf("should not include language line when empty")
	}
	if strings.Contains(out, "-specific") {
		t.Fatalf("should not include language-specific instruction when empty")
	}
}

func TestBuildMetricLLMUserContent_IncludesAdvice(t *testing.T) {
	for _, mt := range []string{"readme", "tests", "ci", "structure", "linting", "deps", "docs", "ai_ready"} {
		out := BuildMetricLLMUserContent(mt, map[string]any{}, "o/r", 5, 1)
		if !strings.Contains(out, "What good looks like") {
			t.Errorf("%s: expected advice section", mt)
		}
	}
}

func TestMetricDisplayName(t *testing.T) {
	if MetricDisplayName("ai_ready") != "AI-ready setup" {
		t.Fatalf("unexpected display name")
	}
}

func TestHumanizeSignalKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"has_readme", "Has README"},
		{"has_license_file", "Has LICENSE file"},
		{"some_unknown_flag", "Some unknown flag"},
	}
	for _, tt := range tests {
		got := humanizeSignalKey(tt.key)
		if got != tt.want {
			t.Errorf("humanizeSignalKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
