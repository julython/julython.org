package metrics

import (
	"path"
	"regexp"
	"strings"
)

// readmeMatchers defines which files the scanner should fetch for the README metric.
var readmeMatchers = []pathMatcher{
	{
		Category: "readme",
		Limit:    1,
		Match: func(p string, isBlob bool) bool {
			base := strings.ToLower(path.Base(p))
			return isBlob && path.Dir(p) == "." &&
				strings.HasPrefix(base, "readme")
		},
	},
}

// evalReadme scores the README metric from a completed L1 scan.
func evalReadme(res *ScanResult) (Metric, map[string]any, int16) {
	text := firstContent(res, "readme")
	lower := strings.ToLower(text)

	r := Readme{
		HasReadme:           len(strings.TrimSpace(text)) > 0,
		ReadmeSubstantial:   len(text) > 500,
		ReadmeHasInstall:    strings.Contains(lower, "install") || strings.Contains(lower, "getting started"),
		ReadmeHasUsage:      strings.Contains(lower, "usage") || strings.Contains(lower, "## usage"),
		ReadmeHasBanners:    strings.Contains(lower, "badge") || strings.Contains(lower, "shields.io"),
		ReadmeHasCodeBlocks: hasCodeBlocks(text),
	}
	data, _ := structToMap(r)
	src := promptSourcesForMetric([]sourceSpec{
		{path: firstPath(res, "readme"), content: text, maxBytes: 2200},
	})
	addPromptContext(data, src)
	return r, data, r.score()
}

// hasCodeBlocks checks if text contains fenced code blocks with non-trivial content.
func hasCodeBlocks(text string) bool {
	return len(extractCodeBlocks(text)) > 0
}

// extractCodeBlocks finds fenced code blocks in markdown text.
// Uses a simple regex approach — handles ```lang ... ``` fences.
var codeBlockRe = regexp.MustCompile("(?m)^```[^\\n]*\\n(.*?)^```$")

func extractCodeBlocks(text string) []string {
	matches := codeBlockRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	blocks := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, m[1])
		}
	}
	return blocks
}
