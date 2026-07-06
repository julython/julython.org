package metrics

import (
	"path"
	"strings"
)

// testsMatchers defines which files the scanner should fetch for the Tests metric.
var testsMatchers = []pathMatcher{
	{
		Category: "tests",
		Limit:    5,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := path.Base(p)
			dir := strings.ToLower(path.Dir(p))
			// go test files
			if strings.HasSuffix(base, "_test.go") {
				return true
			}
			// python test files
			if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
				return true
			}
			// common test directories
			parts := strings.Split(dir, "/")
			for _, part := range parts {
				switch part {
				case "test", "tests", "__tests__", "spec", "specs":
					return true
				}
			}
			return false
		},
	},
}

// evalTests scores the Tests metric from a completed L1 scan.
func evalTests(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)
	content := contentByPath(res)

	hasDir := pathHasTestDir(paths)
	hasFiles := countTestFiles(paths) >= 1
	fw := detectTestFramework(content, paths)
	hasScript := packageJSONHasTestScript(content["package.json"])

	t := Tests{
		HasTestDir:       hasDir,
		HasTestFiles:     hasFiles,
		HasTestFramework: fw,
		HasTestScript:    hasScript,
	}
	data, _ := structToMap(t)
	var specs []sourceSpec
	for _, m := range res.ByCategory["tests"] {
		if len(specs) >= 3 {
			break
		}
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return t, data, t.score()
}

func pathHasTestDir(paths map[string]bool) bool {
	for p := range paths {
		if strings.HasPrefix(p, "test/") || strings.HasPrefix(p, "tests/") ||
			strings.HasPrefix(p, "spec/") || strings.Contains(p, "__tests__/") {
			return true
		}
	}
	return false
}

func countTestFiles(paths map[string]bool) int {
	n := 0
	for p := range paths {
		base := path.Base(p)
		if strings.HasSuffix(base, "_test.go") {
			n++
			continue
		}
		if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
			n++
		}
	}
	return n
}

func detectTestFramework(content map[string]string, paths map[string]bool) bool {
	if c := content["pyproject.toml"]; c != "" && strings.Contains(strings.ToLower(c), "pytest") {
		return true
	}
	if c := content["package.json"]; c != "" && (strings.Contains(c, `"jest"`) || strings.Contains(c, `"vitest"`)) {
		return true
	}
	if paths["jest.config.js"] || paths["vitest.config.ts"] {
		return true
	}
	return false
}

func packageJSONHasTestScript(pkg string) bool {
	if pkg == "" {
		return false
	}
	return strings.Contains(pkg, `"test"`) && strings.Contains(pkg, "scripts")
}
