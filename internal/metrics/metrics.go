package metrics

import (
	"encoding/json"
	"reflect"
)

// maxPromptContextBytes is the total cap for all snippet text stored in analysis_metrics.data
// (prompt_context) and fed to browser LLMs. Keeps WebLLM (~4k context) and Chrome AI prompts usable.
const maxPromptContextBytes = 3000

// PromptContextKey is stored alongside scored bool fields in analysis_metrics.data for L2/L3 prompts.
const PromptContextKey = "prompt_context"

// LanguageKey is stored in every metric's data so prompts can reference it.
const LanguageKey = "language"

// MetricResult is one tile's JSON payload and score for UpsertAnalysisMetric.
type MetricResult struct {
	Data  map[string]any
	Score int16
}

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

type sourceSpec struct {
	path     string
	content  string
	maxBytes int
}

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
	HasReadme           bool `json:"has_readme"`
	ReadmeSubstantial   bool `json:"readme_substantial"` // > 500 bytes
	ReadmeHasInstall    bool `json:"readme_has_install"`
	ReadmeHasUsage      bool `json:"readme_has_usage"`
	ReadmeHasBanners    bool `json:"readme_has_banners"`
	ReadmeHasCodeBlocks bool `json:"readme_has_code_blocks"`
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

// Parse unmarshals raw JSON into the correct metric struct for the given type.
// Unknown types and nil data return a zero-value struct (all bools false).
// This is safe for partial JSON — missing fields default to false.
func Parse(metricType string, data json.RawMessage) (Metric, error) {
	switch metricType {
	case "readme":
		var r Readme
		if len(data) > 0 {
			_ = json.Unmarshal(data, &r)
		}
		return r, nil
	case "tests":
		var t Tests
		if len(data) > 0 {
			_ = json.Unmarshal(data, &t)
		}
		return t, nil
	case "ci":
		var c CI
		if len(data) > 0 {
			_ = json.Unmarshal(data, &c)
		}
		return c, nil
	case "structure":
		var s Structure
		if len(data) > 0 {
			_ = json.Unmarshal(data, &s)
		}
		return s, nil
	case "linting":
		var l Linting
		if len(data) > 0 {
			_ = json.Unmarshal(data, &l)
		}
		return l, nil
	case "deps":
		var d Deps
		if len(data) > 0 {
			_ = json.Unmarshal(data, &d)
		}
		return d, nil
	case "docs":
		var do Docs
		if len(data) > 0 {
			_ = json.Unmarshal(data, &do)
		}
		return do, nil
	case "ai_ready":
		var a AIReady
		if len(data) > 0 {
			_ = json.Unmarshal(data, &a)
		}
		return a, nil
	}
	return Readme{}, nil
}

// structToMap marshals a struct to JSON and unmarshals back to a map[string]any.
func structToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// promptSourcesForMetric builds context entries for LLM prompts, respecting the total byte budget.
func promptSourcesForMetric(specs []sourceSpec) []map[string]string {
	const maxSources = 5
	var out []map[string]string
	total := 0
	for _, s := range specs {
		if s.path == "" {
			continue
		}
		// Path-only: file matters for the metric but body is not useful in-context (e.g. LICENSE).
		if s.content == "" {
			out = append(out, map[string]string{"path": s.path, "snippet": ""})
			if len(out) >= maxSources {
				break
			}
			continue
		}
		max := s.maxBytes
		if max <= 0 {
			max = 2000
		}
		if max > 2200 {
			max = 2200
		}
		sn := s.content
		if len(sn) > max {
			sn = sn[:max]
		}
		if total+len(sn) > maxPromptContextBytes {
			remain := maxPromptContextBytes - total
			if remain < 80 {
				break
			}
			sn = sn[:remain]
		}
		out = append(out, map[string]string{"path": s.path, "snippet": sn})
		total += len(sn)
		if len(out) >= maxSources {
			break
		}
	}
	return out
}

func addPromptContext(data map[string]any, sources []map[string]string) {
	if len(sources) == 0 {
		return
	}
	data[PromptContextKey] = map[string]any{"sources": sources}
}

func setLanguage(data map[string]any, lang string) {
	if lang != "" {
		data[LanguageKey] = lang
	}
}

func pathSetFromTree(tree []TreeEntry) map[string]bool {
	m := make(map[string]bool, len(tree))
	for _, e := range tree {
		if e.Type == "blob" {
			m[e.Path] = true
		}
	}
	return m
}

func contentByPath(res *ScanResult) map[string]string {
	cm := make(map[string]string)
	for _, m := range res.Matches {
		if m.Content != "" {
			cm[m.Path] = m.Content
		}
	}
	return cm
}

func firstContent(res *ScanResult, category string) string {
	for _, m := range res.ByCategory[category] {
		if m.Content != "" {
			return m.Content
		}
	}
	return ""
}

func firstPath(res *ScanResult, category string) string {
	for _, m := range res.ByCategory[category] {
		if m.Path != "" {
			return m.Path
		}
	}
	return ""
}
