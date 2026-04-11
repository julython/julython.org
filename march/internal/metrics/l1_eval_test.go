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
				Content:  "MIT License\n",
			}},
		},
	}
	out := EvaluateL1(res)
	require.True(t, out["structure"].Score > 0)
}
