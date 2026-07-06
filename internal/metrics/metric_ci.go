package metrics

import "strings"

// ciMatchers defines which files the scanner should fetch for the CI metric.
var ciMatchers = []pathMatcher{
	{
		Category: "ci",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			return strings.HasPrefix(p, ".github/workflows/") ||
				p == ".gitlab-ci.yml" ||
				strings.HasPrefix(p, ".circleci/") ||
				p == "Jenkinsfile" ||
				strings.HasPrefix(p, ".buildkite/")
		},
	},
}

// evalCI scores the CI metric from a completed L1 scan.
func evalCI(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)
	has := len(res.ByCategory["ci"]) > 0 || paths[".gitlab-ci.yml"] || paths["Jenkinsfile"]
	var yamlText string
	var yamlPath string
	for _, m := range res.ByCategory["ci"] {
		if strings.HasSuffix(m.Path, ".yml") || strings.HasSuffix(m.Path, ".yaml") {
			yamlPath = m.Path
			yamlText = m.Content
			break
		}
	}
	if yamlText == "" {
		for _, m := range res.ByCategory["ci"] {
			yamlPath = m.Path
			yamlText = m.Content
			break
		}
	}
	yl := strings.ToLower(yamlText)

	c := CI{
		HasCI:        has,
		HasLintStep:  has && (strings.Contains(yl, "lint") || strings.Contains(yl, "eslint") || strings.Contains(yl, "golangci")),
		HasTestStep:  has && (strings.Contains(yl, "test") || strings.Contains(yl, "pytest") || strings.Contains(yl, "go test")),
		HasBuildStep: has && (strings.Contains(yl, "build") || strings.Contains(yl, "compile") || strings.Contains(yl, "make")),
	}
	data, _ := structToMap(c)
	var specs []sourceSpec
	if yamlPath != "" && yamlText != "" {
		specs = append(specs, sourceSpec{path: yamlPath, content: yamlText, maxBytes: 1800})
	} else {
		for _, m := range res.ByCategory["ci"] {
			if m.Content != "" {
				specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1800})
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return c, data, c.score()
}
