package metrics

import "path"

// depsMatchers defines which files the scanner should fetch for the Deps metric.
var depsMatchers = []pathMatcher{
	{
		Category: "dependencies",
		Limit:    2,
		Match: func(p string, isBlob bool) bool {
			if !isBlob || path.Dir(p) != "." {
				return false
			}
			depFiles := []string{
				"go.mod", "go.sum",
				"package.json",
				"requirements.txt", "pyproject.toml", "setup.py", "setup.cfg",
				"Cargo.toml", "Gemfile", "pom.xml", "build.gradle",
				"composer.json", "mix.exs",
			}
			for _, f := range depFiles {
				if p == f {
					return true
				}
			}
			return false
		},
	},
}

// evalDeps scores the Deps metric from a completed L1 scan.
func evalDeps(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)

	lock := paths["go.sum"] || paths["package-lock.json"] || paths["pnpm-lock.yaml"] || paths["yarn.lock"] || paths["Cargo.lock"] || paths["poetry.lock"]
	dependabot := paths[".github/dependabot.yml"] || paths[".github/dependabot.yaml"]
	renovate := paths["renovate.json"] || paths["renovate.json5"]

	d := Deps{
		HasLockFile:   lock,
		HasDependabot: dependabot,
		HasRenovate:   renovate,
	}
	data, _ := structToMap(d)
	var specs []sourceSpec
	for _, m := range res.ByCategory["dependencies"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1500})
			if len(specs) >= 2 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return d, data, d.score()
}
