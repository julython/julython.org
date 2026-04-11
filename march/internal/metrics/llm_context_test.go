package metrics

import (
	"strings"
	"testing"
)

func TestBuildMetricLLMUserContent_IncludesInstruction(t *testing.T) {
	data := map[string]any{
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
	if !strings.Contains(out, "README.md") || !strings.Contains(out, "# Hello") {
		t.Fatalf("expected snippet in output")
	}
	if !strings.Contains(out, `{"score":N,"message"`) {
		t.Fatalf("expected JSON instruction tail")
	}
}

func TestMetricDisplayName(t *testing.T) {
	if MetricDisplayName("ai_ready") != "AI-ready setup" {
		t.Fatalf("unexpected display name")
	}
}
