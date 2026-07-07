package metrics

import (
	"path"
	"strings"
)

// structureMatchers defines which files the scanner should fetch for the Structure metric.
var structureMatchers = []pathMatcher{
	{
		Category: "license",
		Limit:    1,
		Match: func(p string, isBlob bool) bool {
			base := strings.ToLower(path.Base(p))
			return isBlob && path.Dir(p) == "." &&
				(strings.HasPrefix(base, "license") || strings.HasPrefix(base, "licence"))
		},
	},
}

// evalStructure scores the Structure metric from a completed L1 scan.
func evalStructure(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)

	src := hasTopLevelDir(paths, []string{"src", "lib", "app", "pkg", "cmd"})
	ignore := paths[".gitignore"] || paths[".git/info/exclude"]
	lic := len(res.ByCategory["license"]) > 0 || hasRootLicense(paths)

	s := Structure{
		HasSrcDir:      src,
		HasIgnoreFile:  ignore,
		HasLicenseFile: lic,
	}
	data, _ := structToMap(s)
	var specs []sourceSpec
	// Presence of LICENSE matters for scoring; full text is noise for small-context browser LLMs.
	if p := firstPath(res, "license"); p != "" {
		specs = append(specs, sourceSpec{path: p, content: "", maxBytes: 0})
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return s, data, s.score()
}

func hasTopLevelDir(paths map[string]bool, dirs []string) bool {
	for p := range paths {
		for _, d := range dirs {
			if strings.HasPrefix(p, d+"/") {
				return true
			}
		}
	}
	return false
}

func hasRootLicense(paths map[string]bool) bool {
	for p := range paths {
		if path.Dir(p) != "." {
			continue
		}
		b := strings.ToLower(path.Base(p))
		if strings.HasPrefix(b, "license") || strings.HasPrefix(b, "licence") {
			return true
		}
	}
	return false
}
