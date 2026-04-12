package metrics

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
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

// MetricLLMSystemPrompt is the system instruction for browser-side metric tile reviews.
const MetricLLMSystemPrompt = `You are a concise code-quality coach. Use markdown bullets. The L1 score already reflects the scan—explain gaps and next steps; do not assign a new numeric grade.`

// Limits for browser LLM user prompts (small context windows; old DB rows may still hold large snippets).
const (
	maxMetricLLMUserBytes     = 3400
	maxEvidenceSnippetBytes     = 1000
	maxEvidenceSnippetsTotalBytes = 2200
)

// BuildMetricLLMUserContent assembles focused evidence for the browser LLM (metric tile AI button).
// Language is read from data[LanguageKey] if present.
func BuildMetricLLMUserContent(metricType string, data map[string]any, repoName string, l1Score, level int16) string {
	var b strings.Builder
	title := MetricDisplayName(metricType)
	language, _ := data[LanguageKey].(string)

	b.WriteString(fmt.Sprintf("## %s — %s\n\n", repoName, title))
	if language != "" {
		b.WriteString(fmt.Sprintf("Language: **%s**\n", language))
	}
	b.WriteString(fmt.Sprintf("L1 score: **%d/10** (reference only)\n\n", l1Score))

	b.WriteString("### Evidence\n\n")
	writePromptEvidence(&b, data)

	b.WriteString("### Task\n\n")
	b.WriteString("Briefly: strengths, gaps, 2–3 concrete improvements")
	if language != "" {
		b.WriteString(fmt.Sprintf(" (for **%s**)", language))
	}
	b.WriteString(".")

	out := b.String()
	if len(out) > maxMetricLLMUserBytes {
		out = truncateUTF8(out, maxMetricLLMUserBytes)
	}
	return out
}

// writePromptEvidence appends file snippets and/or heuristic checklist lines from L1 data.
func writePromptEvidence(b *strings.Builder, data map[string]any) {
	hasSources := false
	evidenceBudget := maxEvidenceSnippetsTotalBytes
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
				b.WriteString(fmt.Sprintf("**%s**\n", path))
				if snip == "" {
					b.WriteString("_File present — contents omitted._\n\n")
					continue
				}
				snip = truncateUTF8(snip, maxEvidenceSnippetBytes)
				if len(snip) > evidenceBudget {
					snip = truncateUTF8(snip, evidenceBudget)
				}
				evidenceBudget -= len(snip)
				b.WriteString("```\n")
				b.WriteString(snip)
				b.WriteString("\n```\n\n")
				if evidenceBudget < 80 {
					break
				}
			}
		}
	}

	if !hasSources {
		b.WriteString(formatHeuristicSignals(data))
		b.WriteString("\n\n")
	}
}

func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// formatHeuristicSignals turns boolean/string flags into readable checkmarks.
func formatHeuristicSignals(data map[string]any) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == PromptContextKey || k == LanguageKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "(No signals detected.)"
	}

	var b strings.Builder
	for _, k := range keys {
		v := data[k]
		label := humanizeSignalKey(k)
		switch val := v.(type) {
		case bool:
			if val {
				b.WriteString(fmt.Sprintf("- ✅ %s\n", label))
			} else {
				b.WriteString(fmt.Sprintf("- ❌ %s\n", label))
			}
		default:
			b.WriteString(fmt.Sprintf("- %s: %v\n", label, v))
		}
	}
	return strings.TrimSpace(b.String())
}

// humanizeSignalKey converts snake_case keys into readable labels.
func humanizeSignalKey(key string) string {
	replacements := map[string]string{
		"has_readme":       "Has README",
		"has_license_file": "Has LICENSE file",
		"has_ignore_file":  "Has .gitignore",
		"has_src_dir":      "Has source directory",
		"has_test_dir":     "Has test directory",
		"has_ci":           "Has CI configuration",
		"has_docs_dir":     "Has docs directory",
		"has_lockfile":     "Has dependency lock file",
		"has_linter":       "Has linter configuration",
		"has_formatter":    "Has code formatter",
		"has_editorconfig": "Has .editorconfig",
		"has_ai_config":    "Has AI configuration",
	}
	if label, ok := replacements[key]; ok {
		return label
	}
	// fallback: replace underscores with spaces, capitalize first letter
	s := strings.ReplaceAll(key, "_", " ")
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
