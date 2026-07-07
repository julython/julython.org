package metrics

import (
	"path"
	"strings"
)

// aiReadyMatchers defines which files the scanner should fetch for the AI-Ready metric.
var aiReadyMatchers = []pathMatcher{
	{
		Category: "ai_readability",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := strings.ToLower(path.Base(p))
			return base == "claude.md" || base == "agents.md" ||
				base == ".cursorrules" || base == ".cursorignore" ||
				base == "copilot-instructions.md" ||
				base == ".github/copilot-instructions.md" ||
				base == "coderabbit.yaml" || base == ".coderabbit.yaml"
		},
	},
}

// evalAIReady scores the AI-Ready metric from a completed L1 scan.
func evalAIReady(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)

	cr := paths[".cursorrules"] || paths[".cursor/rules"]
	claude := paths["CLAUDE.md"] || paths["claude.md"] || paths["Claude.md"]
	copilot := paths[".github/copilot-instructions.md"] || paths["copilot-instructions.md"]
	ign := paths[".cursorignore"]

	a := AIReady{
		HasCursorRules:   cr,
		HasClaudeConfig:  claude,
		HasCopilotConfig: copilot,
		HasAiIgnore:      ign,
	}
	data, _ := structToMap(a)
	var specs []sourceSpec
	for _, m := range res.ByCategory["ai_readability"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1500})
			if len(specs) >= 3 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return a, data, a.score()
}
