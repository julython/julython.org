package metrics

import "path"

// lintingMatchers defines which files the scanner should fetch for the Linting metric.
var lintingMatchers = []pathMatcher{
	{
		Category: "linting",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := path.Base(p)
			lintFiles := []string{
				".golangci.yml", ".golangci.yaml",
				".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yml",
				"eslint.config.js", "eslint.config.mjs",
				".flake8", "ruff.toml", ".ruff.toml",
				".prettierrc", ".prettierrc.json",
				".editorconfig", ".clang-format",
				"biome.json", "deno.json",
			}
			for _, f := range lintFiles {
				if base == f {
					return true
				}
			}
			return false
		},
	},
}

// evalLinting scores the Linting metric from a completed L1 scan.
func evalLinting(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)

	lintFiles := len(res.ByCategory["linting"]) > 0
	pre := paths[".pre-commit-config.yaml"] || paths[".pre-commit-config.yml"]

	l := Linting{
		HasLintConfig:     lintFiles,
		HasPreCommitHooks: pre,
	}
	data, _ := structToMap(l)
	var specs []sourceSpec
	for _, m := range res.ByCategory["linting"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
			if len(specs) >= 2 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return l, data, l.score()
}
