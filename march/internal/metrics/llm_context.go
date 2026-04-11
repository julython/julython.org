package metrics

import (
	"fmt"
	"sort"
	"strings"
)

// MetricHelpAnchor returns the HTML fragment ID for per-metric help (/help#…).
func MetricHelpAnchor(metricType string) string {
	return "analysis-metric-" + metricType
}

// MetricDisplayName returns a short human-readable title for a metric key.
func MetricDisplayName(metricType string) string {
	switch metricType {
	case "readme":
		return "README"
	case "tests":
		return "Tests"
	case "ci":
		return "CI"
	case "structure":
		return "Structure"
	case "linting":
		return "Linting"
	case "deps":
		return "Dependencies"
	case "docs":
		return "Documentation"
	case "ai_ready":
		return "AI-ready setup"
	default:
		return metricType
	}
}

// MetricLLMSystemPrompt is the system instruction for browser-side WebLLM JSON scoring.
// web/js/llm.ts detects metric mode via substring "single JSON object only" — keep that phrase if you edit this text.
const MetricLLMSystemPrompt = `You are an expert software repository reviewer. You receive evidence for a single quality metric. Be strict but fair. You must respond with a single JSON object only — no markdown fences, no prose before or after.`

// BuildMetricLLMUserContent assembles focused evidence and the JSON response instruction for the browser LLM.
func BuildMetricLLMUserContent(metricType string, data map[string]any, repoName string, l1Score, level int16) string {
	var b strings.Builder
	title := MetricDisplayName(metricType)
	b.WriteString(fmt.Sprintf("Repository: %s\n", repoName))
	b.WriteString(fmt.Sprintf("Metric: %s\n", title))
	b.WriteString(fmt.Sprintf("Server-side L1 heuristic score: %d/10 (AI tier level %d out of 3).\n\n", l1Score, level))
	b.WriteString("Evidence from the repository scan:\n\n")

	hasSources := false
	if pc, ok := data[PromptContextKey].(map[string]any); ok {
		if sources, ok := pc["sources"].([]any); ok && len(sources) > 0 {
			hasSources = true
			for _, src := range sources {
				sm, ok := src.(map[string]any)
				if !ok {
					continue
				}
				path, _ := sm["path"].(string)
				snip, _ := sm["snippet"].(string)
				if path == "" {
					continue
				}
				b.WriteString(fmt.Sprintf("### %s\n", path))
				if snip != "" {
					b.WriteString("```\n")
					b.WriteString(snip)
					b.WriteString("\n```\n\n")
				}
			}
		}
	}
	if !hasSources {
		b.WriteString("(No file snippets in this scan; heuristic flags only.)\n\n")
		b.WriteString(heuristicSummaryFromData(data))
		b.WriteString("\n")
	}

	b.WriteString("\nTask: Rate this project on this metric from 0 to 10 (10 = best). Respond with JSON only, no markdown: ")
	b.WriteString(`{"score":N,"message":"..."} where message is 2-4 sentences with concrete suggestions.`)
	return b.String()
}

func heuristicSummaryFromData(data map[string]any) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == PromptContextKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "(No heuristic flags.)"
	}
	var b strings.Builder
	b.WriteString("Heuristic signals:\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- %s: %v\n", k, data[k]))
	}
	return strings.TrimSpace(b.String())
}
