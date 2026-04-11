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

// MetricLLMSystemPrompt is the system instruction for browser-side LLM reviews.
const MetricLLMSystemPrompt = `You are a friendly code quality coach reviewing open source projects. Give concise, actionable feedback. Use markdown formatting for structure.`

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

// BuildMetricLLMUserContent assembles focused evidence for the browser LLM.
func BuildMetricLLMUserContent(metricType string, data map[string]any, repoName string, l1Score, level int16) string {
	var b strings.Builder
	title := MetricDisplayName(metricType)

	b.WriteString(fmt.Sprintf("## Reviewing: %s — %s\n\n", repoName, title))
	b.WriteString(fmt.Sprintf("Current automated score: **%d/10**\n\n", l1Score))

	// what good looks like for this category
	if advice, ok := metricAdvice[metricType]; ok {
		b.WriteString(fmt.Sprintf("What good looks like: %s\n\n", advice))
	}

	// evidence section
	b.WriteString("### What the scan found\n\n")

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
				b.WriteString(fmt.Sprintf("**%s**\n", path))
				if snip != "" {
					b.WriteString("```\n")
					b.WriteString(snip)
					b.WriteString("\n```\n\n")
				}
			}
		}
	}

	if !hasSources {
		b.WriteString(formatHeuristicSignals(data))
		b.WriteString("\n\n")
	}

	b.WriteString("### Your task\n\n")
	b.WriteString("Based on the evidence above:\n")
	b.WriteString("1. Briefly explain what this project is doing well\n")
	b.WriteString("2. List 2-3 specific, actionable steps the maintainer could take today to improve\n")
	b.WriteString("3. End with an updated score suggestion out of 10\n\n")
	b.WriteString("Keep it concise — aim for a short paragraph plus a few bullet points.")

	return b.String()
}

// formatHeuristicSignals turns boolean/string flags into readable checkmarks.
func formatHeuristicSignals(data map[string]any) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == PromptContextKey {
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
