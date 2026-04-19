package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateL1_ReadmeSubstantial(t *testing.T) {
	res := &L1ScanResult{
		SHA: "abc",
		Tree: []TreeEntry{
			{Path: "README.md", Type: "blob"},
		},
		ByCategory: map[string][]FileMatch{
			"readme": {{
				Path:     "README.md",
				Category: "readme",
				Content:  strings.Repeat("x", 600) + "\n## Usage\ninstall: `make`\n[![badge](https://shields.io/x)]",
			}},
		},
	}
	out := EvaluateL1(res)
	r := out["readme"]
	require.Equal(t, int16(10), r.Score)
	require.NotNil(t, r.Data[PromptContextKey])
}

func TestEvaluateL1_StructureLicense(t *testing.T) {
	res := &L1ScanResult{
		SHA: "abc",
		Tree: []TreeEntry{
			{Path: "LICENSE", Type: "blob"},
			{Path: ".gitignore", Type: "blob"},
			{Path: "src/main.go", Type: "blob"},
		},
		ByCategory: map[string][]FileMatch{
			"license": {{
				Path:     "LICENSE",
				Category: "license",
				Content:  "MIT License\nCopyright (c) Someone\n" + strings.Repeat("x", 8000),
			}},
		},
	}
	out := EvaluateL1(res)
	require.True(t, out["structure"].Score > 0)
	pc := out["structure"].Data[PromptContextKey].(map[string]any)
	sources := pc["sources"].([]map[string]string)
	require.Len(t, sources, 1)
	require.Equal(t, "LICENSE", sources[0]["path"])
	require.Equal(t, "", sources[0]["snippet"], "LICENSE body should not be stuffed into browser LLM context")
}

func TestPromptSourcesForMetric_TotalBudget(t *testing.T) {
	huge := strings.Repeat("a", 10000)
	out := promptSourcesForMetric([]sourceSpec{
		{path: "a.txt", content: huge, maxBytes: 5000},
		{path: "b.txt", content: huge, maxBytes: 5000},
	})
	var total int
	for _, m := range out {
		total += len(m["snippet"])
	}
	require.LessOrEqual(t, total, maxPromptContextBytes)
}
