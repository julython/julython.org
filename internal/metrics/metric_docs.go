package metrics

import (
	"path"
	"strings"
)

// docsMatchers defines which files the scanner should fetch for the Docs metric.
var docsMatchers = []pathMatcher{
	{
		Category: "docs",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			dir := strings.Split(p, "/")[0]
			switch strings.ToLower(dir) {
			case "docs", "doc", "documentation", "wiki":
				return true
			}
			// mkdocs, sphinx, etc.
			base := path.Base(p)
			return base == "mkdocs.yml" || base == "conf.py" ||
				base == "docusaurus.config.js" || base == ".readthedocs.yml"
		},
	},
}

// evalDocs scores the Docs metric from a completed L1 scan.
func evalDocs(res *ScanResult) (Metric, map[string]any, int16) {
	paths := pathSetFromTree(res.Tree)

	hasDocsDir := pathPrefixExists(paths, "docs/") || pathPrefixExists(paths, "documentation/")
	changelog := paths["CHANGELOG.md"] || paths["CHANGELOG.rst"] || paths["CHANGELOG"]
	contrib := paths["CONTRIBUTING.md"] || paths["CONTRIBUTING.rst"]
	coc := paths["CODE_OF_CONDUCT.md"] || paths["CODE_OF_CONDUCT"]

	d := Docs{
		HasDocsDir:       hasDocsDir || len(res.ByCategory["docs"]) > 0,
		HasChangelog:     changelog,
		HasContributing:  contrib,
		HasCodeOfConduct: coc,
	}
	data, _ := structToMap(d)
	var specs []sourceSpec
	for _, m := range res.ByCategory["docs"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
			if len(specs) >= 3 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return d, data, d.score()
}

func pathPrefixExists(paths map[string]bool, prefix string) bool {
	for p := range paths {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
