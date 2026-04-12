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

// MetricLLMSystemPrompt is the system instruction for browser-side LLM reviews (WebLLM ~4k context, Chrome Prompt API).
const MetricLLMSystemPrompt = `You are a helpful code quality coach. Be concise and use markdown (short headings, bullets).

Do not paste or quote license text, long legal boilerplate, or repeat large excerpts from the prompt—reference file paths when useful.

The automated scan already produced a score; do not give a new numeric grade or argue about the number. Focus on what is missing or weak and concrete steps to improve.`

// metricAdvice provides per-category context so the LLM knows what "good" looks like.
var metricAdvice = map[string]string{
	"readme":    `A good README clearly states what the project does, how to install it, and how to use it with code examples. Badges, screenshots, and a table of contents help for larger projects.`,
	"tests":     `Good test coverage means tests exist for core functionality, edge cases are handled, and tests are organized and readable. Both unit and integration tests add value.`,
	"ci":        `Good CI runs tests, linting, and builds automatically on push and PR. Matrix testing across OS/versions, caching, and security scanning are signs of maturity.`,
	"structure": `Good project structure follows language conventions (e.g. cmd/internal for Go, src/ for JS/Python), has clear entry points, a proper .gitignore, a LICENSE file, and a build/task configuration.`,
	"linting":   `Good linting setup includes a formatter, a linter with non-trivial rules, editor config, and ideally pre-commit hooks or CI enforcement.`,
	"deps":      `Good dependency management means a lock file is committed, versions are pinned, the dependency count is reasonable, and automated update tooling (Dependabot/Renovate) is configured.`,
	"docs":      `Good documentation goes beyond the README — a docs/ directory, generated doc site, quickstart guide, and API reference show a mature project.`,
	"ai_ready":  `AI-ready projects include configuration files like CLAUDE.md, AGENTS.md, .cursorrules, or copilot-instructions.md that give AI tools context about the project's architecture, conventions, and boundaries.`,
}

// Limits for browser LLM user prompts (small context windows; old DB rows may still hold large snippets).
const (
	maxMetricLLMUserBytes     = 3400
	maxEvidenceSnippetBytes     = 1000
	maxEvidenceSnippetsTotalBytes = 2200
)

// BuildMetricLLMUserContent assembles focused evidence for the browser LLM.
// Language is read from data[LanguageKey] if present.
func BuildMetricLLMUserContent(metricType string, data map[string]any, repoName string, l1Score, level int16) string {
	var b strings.Builder
	title := MetricDisplayName(metricType)
	language, _ := data[LanguageKey].(string)

	b.WriteString(fmt.Sprintf("## Reviewing: %s — %s\n\n", repoName, title))
	if language != "" {
		b.WriteString(fmt.Sprintf("Primary language: **%s**\n", language))
	}
	b.WriteString(fmt.Sprintf("Context: this area already scored **%d/10** on our automated scan (that grade is final here). ", l1Score))
	b.WriteString("Your job is to explain gaps and improvements—not to re-score.\n\n")

	// what good looks like for this category
	if advice, ok := metricAdvice[metricType]; ok {
		b.WriteString(fmt.Sprintf("What good looks like: %s\n\n", advice))
	}

	// evidence section
	b.WriteString("### What the scan found\n\n")
	writePromptEvidence(&b, data)

	b.WriteString("### What to write\n\n")
	b.WriteString("In plain markdown, briefly cover:\n")
	b.WriteString("1. What is missing or weakest compared to what “good” looks like for this area\n")
	b.WriteString("2. A few concrete, actionable steps the maintainer could take\n\n")
	if language != "" {
		b.WriteString(fmt.Sprintf("Tailor suggestions to a **%s** project (tooling, layout, conventions).\n\n", language))
	}
	b.WriteString("Keep it short. Do not output JSON.")

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
