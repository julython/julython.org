package metrics

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Metric is implemented by every tile struct.
// Score counts true boolean fields as partial credit out of 10.
type Metric interface {
	score() int16
}

// Score computes partial credit: (trueCount / totalBools) * 10.
// Exported so the handler can call it without knowing the concrete type.
func Score(m Metric) int16 {
	return m.score()
}

// scoreFields uses reflection to count true bool fields.
// All tile structs embed this to get Score() for free.
type scoreFields struct{}

func (scoreFields) score() int16 { return 0 } // never called directly

func countBools(v any) int16 {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	rt := rv.Type()
	total, passed := 0, 0
	for i := range rt.NumField() {
		if rt.Field(i).Type.Kind() == reflect.Bool {
			total++
			if rv.Field(i).Bool() {
				passed++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return int16((passed * 10) / total)
}

// ── Tile structs ──────────────────────────────────────────────────
// JSON keys become UI lookup keys for displaying feedback to the user.

type Readme struct {
	HasReadme         bool `json:"has_readme"`
	ReadmeSubstantial bool `json:"readme_substantial"` // > 500 bytes
	ReadmeHasInstall  bool `json:"readme_has_install"`
	ReadmeHasUsage    bool `json:"readme_has_usage"`
	ReadmeHasBanners  bool `json:"readme_has_banners"`
}

func (r Readme) score() int16 { return countBools(r) }

type Tests struct {
	HasTestDir       bool `json:"has_test_dir"`
	HasTestFiles     bool `json:"has_test_files"`
	HasTestFramework bool `json:"has_test_framework"`
	HasTestScript    bool `json:"has_test_script"`
}

func (t Tests) score() int16 { return countBools(t) }

type CI struct {
	HasCI        bool `json:"has_ci"`
	HasLintStep  bool `json:"has_lint_step"`
	HasTestStep  bool `json:"has_test_step"`
	HasBuildStep bool `json:"has_build_step"`
}

func (c CI) score() int16 { return countBools(c) }

type Structure struct {
	HasSrcDir      bool `json:"has_src_dir"`
	HasIgnoreFile  bool `json:"has_ignore_file"`
	HasLicenseFile bool `json:"has_license_file"`
}

func (s Structure) score() int16 { return countBools(s) }

type Linting struct {
	HasLintConfig     bool `json:"has_lint_config"`
	HasPreCommitHooks bool `json:"has_pre_commit_hooks"`
}

func (l Linting) score() int16 { return countBools(l) }

type Deps struct {
	HasLockFile   bool `json:"has_lock_file"`
	HasDependabot bool `json:"has_dependabot"`
	HasRenovate   bool `json:"has_renovate"`
}

func (d Deps) score() int16 { return countBools(d) }

type Docs struct {
	HasDocsDir       bool `json:"has_docs_dir"`
	HasChangelog     bool `json:"has_changelog"`
	HasContributing  bool `json:"has_contributing"`
	HasCodeOfConduct bool `json:"has_code_of_conduct"`
}

func (d Docs) score() int16 { return countBools(d) }

type AIReady struct {
	HasCursorRules   bool `json:"has_cursor_rules"`
	HasClaudeConfig  bool `json:"has_claude_config"`
	HasCopilotConfig bool `json:"has_copilot_config"`
	HasAiIgnore      bool `json:"has_ai_ignore"`
}

func (a AIReady) score() int16 { return countBools(a) }

// ── Dispatch ──────────────────────────────────────────────────────

// Parse unmarshals raw JSON into the correct metric struct for the given type
// and returns its score. Returns an error for unknown metric types.
func Parse(metricType string, data json.RawMessage) (Metric, error) {
	switch metricType {
	case "readme":
		return unmarshal[Readme](data)
	case "tests":
		return unmarshal[Tests](data)
	case "ci":
		return unmarshal[CI](data)
	case "structure":
		return unmarshal[Structure](data)
	case "linting":
		return unmarshal[Linting](data)
	case "deps":
		return unmarshal[Deps](data)
	case "docs":
		return unmarshal[Docs](data)
	case "ai_ready":
		return unmarshal[AIReady](data)
	}
	return nil, fmt.Errorf("unknown metric type: %s", metricType)
}

func unmarshal[T Metric](data json.RawMessage) (Metric, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}
