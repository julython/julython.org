package metrics

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Tile scoring ---

func TestReadmeScore(t *testing.T) {
	r := Readme{}
	assert.Equal(t, int16(0), r.score())

	r = Readme{HasReadme: true, ReadmeSubstantial: true}
	assert.Equal(t, int16(4), r.score()) // 2 of 5 = (2*10)/5 = 4

	r = Readme{
		HasReadme: true, ReadmeSubstantial: true, ReadmeHasInstall: true,
		ReadmeHasUsage: true, ReadmeHasBanners: true,
	}
	assert.Equal(t, int16(10), r.score())
}

func TestTestsScore(t *testing.T) {
	t0 := Tests{}
	assert.Equal(t, int16(0), t0.score())

	t1 := Tests{HasTestDir: true}
	assert.Equal(t, int16(2), t1.score()) // 1 of 4 = (1*10)/4 = 2

	t2 := Tests{HasTestDir: true, HasTestFiles: true, HasTestFramework: true}
	assert.Equal(t, int16(7), t2.score()) // 3 of 4 = (3*10)/4 = 7

	tAll := Tests{HasTestDir: true, HasTestFiles: true, HasTestFramework: true, HasTestScript: true}
	assert.Equal(t, int16(10), tAll.score()) // 4 of 4 = 100%
}

func TestCIScore(t *testing.T) {
	ci := CI{}
	assert.Equal(t, int16(0), ci.score())

	ci = CI{HasCI: true, HasLintStep: true}
	assert.Equal(t, int16(5), ci.score()) // 2 of 4 = (2*10)/4 = 5

	ci = CI{HasCI: true, HasLintStep: true, HasTestStep: true, HasBuildStep: true}
	assert.Equal(t, int16(10), ci.score()) // 4 of 4 = 100%
}

func TestStructureScore(t *testing.T) {
	s := Structure{HasSrcDir: true, HasIgnoreFile: true, HasLicenseFile: true}
	assert.Equal(t, int16(10), s.score()) // 3 of 3 = 100%

	s = Structure{}
	assert.Equal(t, int16(0), s.score())

	s = Structure{HasSrcDir: true}
	assert.Equal(t, int16(3), s.score()) // 1 of 3 = (1*10)/3 = 3
}

func TestLintingScore(t *testing.T) {
	l := Linting{HasLintConfig: true, HasPreCommitHooks: true}
	assert.Equal(t, int16(10), l.score()) // 2 of 2 = 100%

	l = Linting{}
	assert.Equal(t, int16(0), l.score())
}

func TestDepsScore(t *testing.T) {
	d := Deps{HasLockFile: true, HasDependabot: true, HasRenovate: true}
	assert.Equal(t, int16(10), d.score()) // 3 of 3 = 100%

	d = Deps{HasLockFile: true}
	assert.Equal(t, int16(3), d.score()) // 1 of 3 = (1*10)/3 = 3
}

func TestDocsScore(t *testing.T) {
	d := Docs{HasDocsDir: true, HasChangelog: true, HasContributing: true, HasCodeOfConduct: true}
	assert.Equal(t, int16(10), d.score()) // 4 of 4 = 100%

	d = Docs{HasDocsDir: true}
	assert.Equal(t, int16(2), d.score()) // 1 of 4 = (1*10)/4 = 2
}

func TestAIReadyScore(t *testing.T) {
	a := AIReady{HasCursorRules: true, HasClaudeConfig: true, HasCopilotConfig: true, HasAiIgnore: true}
	assert.Equal(t, int16(10), a.score()) // 4 of 4 = 100%

	a = AIReady{HasCursorRules: true}
	assert.Equal(t, int16(2), a.score()) // 1 of 4 = (1*10)/4 = 2
}

func TestAIReadyPartialScore(t *testing.T) {
	a := AIReady{HasCursorRules: true, HasClaudeConfig: true}
	assert.Equal(t, int16(5), a.score()) // 2 of 4 = (2*10)/4 = 5
}

// --- Score (exported wrapper) ---

func TestScore(t *testing.T) {
	readme := Readme{HasReadme: true, ReadmeSubstantial: true}
	assert.Equal(t, int16(4), Score(readme))

	ci := CI{HasCI: true, HasTestStep: true}
	assert.Equal(t, int16(5), Score(ci))
}

// --- Parse ---

func TestParseReadme(t *testing.T) {
	data := json.RawMessage(`{"has_readme": true, "readme_substantial": true}`)
	m, err := Parse("readme", data)
	require.NoError(t, err)
	r, ok := m.(Readme)
	require.True(t, ok)
	assert.True(t, r.HasReadme)
	assert.True(t, r.ReadmeSubstantial)
}

func TestParseTests(t *testing.T) {
	data := json.RawMessage(`{"has_test_dir": true}`)
	m, err := Parse("tests", data)
	require.NoError(t, err)
	assert.Equal(t, int16(2), Score(m))
}

func TestParseCI(t *testing.T) {
	data := json.RawMessage(`{"has_ci": true, "has_lint_step": true}`)
	m, err := Parse("ci", data)
	require.NoError(t, err)
	assert.Equal(t, int16(5), Score(m))
}

func TestParseStructure(t *testing.T) {
	data := json.RawMessage(`{"has_src_dir": true, "has_ignore_file": true, "has_license_file": true}`)
	m, err := Parse("structure", data)
	require.NoError(t, err)
	assert.Equal(t, int16(10), Score(m))
}

func TestParseLinting(t *testing.T) {
	data := json.RawMessage(`{"has_lint_config": true}`)
	m, err := Parse("linting", data)
	require.NoError(t, err)
	assert.Equal(t, int16(5), Score(m))
}

func TestParseDeps(t *testing.T) {
	data := json.RawMessage(`{"has_lock_file": true, "has_dependabot": true, "has_renovate": true}`)
	m, err := Parse("deps", data)
	require.NoError(t, err)
	assert.Equal(t, int16(10), Score(m))
}

func TestParseDocs(t *testing.T) {
	data := json.RawMessage(`{"has_docs_dir": true}`)
	m, err := Parse("docs", data)
	require.NoError(t, err)
	assert.Equal(t, int16(2), Score(m))
}

func TestParseAIReady(t *testing.T) {
	data := json.RawMessage(`{"has_cursor_rules": true, "has_claude_config": true, "has_copilot_config": true, "has_ai_ignore": true}`)
	m, err := Parse("ai_ready", data)
	require.NoError(t, err)
	assert.Equal(t, int16(10), Score(m))
}

func TestParseUnknownType(t *testing.T) {
	_, err := Parse("unknown_type", json.RawMessage("{}"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown metric type")
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := Parse("readme", json.RawMessage(`{invalid}`))
	require.Error(t, err)
}

func TestParseAllKnownTypes(t *testing.T) {
	// Verify every known type is handled
	knownTypes := []string{"readme", "tests", "ci", "structure", "linting", "deps", "docs", "ai_ready"}
	for _, typ := range knownTypes {
		t.Run(typ, func(t *testing.T) {
			data := json.RawMessage(`{}`)
			m, err := Parse(typ, data)
			require.NoError(t, err, "type %s should be parseable", typ)
			assert.NotNil(t, m, "parsed metric should not be nil for type %s", typ)
		})
	}
}

// --- Unexported unmarshal ---

func TestUnmarshalEmptyJSON(t *testing.T) {
	data := json.RawMessage(`{}`)
	m, err := Parse("readme", data)
	require.NoError(t, err)
	// All bools false => score 0
	assert.Equal(t, int16(0), Score(m))
}
