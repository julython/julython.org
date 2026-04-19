package metrics

import (
	"encoding/json"
	"path"
	"strings"
)

// maxPromptContextBytes is the total cap for all snippet text stored in analysis_metrics.data
// (prompt_context) and fed to browser LLMs. Keeps WebLLM (~4k context) and Chrome AI prompts usable.
const maxPromptContextBytes = 3000

// PromptContextKey is stored alongside scored bool fields in analysis_metrics.data for L2/L3 prompts.
const PromptContextKey = "prompt_context"

// LanguageKey is stored in every metric's data so prompts can reference it.
const LanguageKey = "language"

// L1MetricResult is one tile's JSON payload and score for UpsertAnalysisMetric.
type L1MetricResult struct {
	Data  map[string]any
	Score int16
}

// EvaluateL1 builds the eight metric rows from a completed L1 scan (tree + file contents).
func EvaluateL1(res *L1ScanResult) map[string]L1MetricResult {
	by := res.ByCategory
	paths := pathSetFromTree(res.Tree)
	lang := res.Language

	out := make(map[string]L1MetricResult, 8)

	readme := evalReadme(by, paths)
	setLanguage(readme.data, lang)
	out["readme"] = L1MetricResult{Data: readme.data, Score: Score(readme.tile)}

	tests := evalTests(by, paths, contentByPath(res))
	setLanguage(tests.data, lang)
	out["tests"] = L1MetricResult{Data: tests.data, Score: Score(tests.tile)}

	ci := evalCI(by, paths)
	setLanguage(ci.data, lang)
	out["ci"] = L1MetricResult{Data: ci.data, Score: Score(ci.tile)}

	structure := evalStructure(by, paths)
	setLanguage(structure.data, lang)
	out["structure"] = L1MetricResult{Data: structure.data, Score: Score(structure.tile)}

	lint := evalLinting(by, paths)
	setLanguage(lint.data, lang)
	out["linting"] = L1MetricResult{Data: lint.data, Score: Score(lint.tile)}

	deps := evalDeps(by, paths)
	setLanguage(deps.data, lang)
	out["deps"] = L1MetricResult{Data: deps.data, Score: Score(deps.tile)}

	docs := evalDocs(by, paths)
	setLanguage(docs.data, lang)
	out["docs"] = L1MetricResult{Data: docs.data, Score: Score(docs.tile)}

	ai := evalAIReady(by, paths)
	setLanguage(ai.data, lang)
	out["ai_ready"] = L1MetricResult{Data: ai.data, Score: Score(ai.tile)}

	return out
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

func contentByPath(res *L1ScanResult) map[string]string {
	cm := make(map[string]string)
	for _, m := range res.Matches {
		if m.Content != "" {
			cm[m.Path] = m.Content
		}
	}
	return cm
}

func firstContent(by map[string][]FileMatch, category string) string {
	for _, m := range by[category] {
		if m.Content != "" {
			return m.Content
		}
	}
	return ""
}

func firstPath(by map[string][]FileMatch, category string) string {
	for _, m := range by[category] {
		if m.Path != "" {
			return m.Path
		}
	}
	return ""
}

// --- per-metric evaluators ---

type readmeEval struct {
	tile Readme
	data map[string]any
}

func evalReadme(by map[string][]FileMatch, paths map[string]bool) readmeEval {
	text := firstContent(by, "readme")
	lower := strings.ToLower(text)

	r := Readme{
		HasReadme:         len(strings.TrimSpace(text)) > 0,
		ReadmeSubstantial: len(text) > 500,
		ReadmeHasInstall:  strings.Contains(lower, "install") || strings.Contains(lower, "getting started"),
		ReadmeHasUsage:    strings.Contains(lower, "usage") || strings.Contains(lower, "## usage"),
		ReadmeHasBanners:  strings.Contains(lower, "badge") || strings.Contains(lower, "shields.io"),
	}
	data, _ := structToMap(r)
	src := promptSourcesForMetric([]sourceSpec{
		{path: firstPath(by, "readme"), content: text, maxBytes: 2200},
	})
	addPromptContext(data, src)
	return readmeEval{tile: r, data: data}
}

type testsEval struct {
	tile Tests
	data map[string]any
}

func evalTests(by map[string][]FileMatch, paths map[string]bool, content map[string]string) testsEval {
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
	for _, m := range by["tests"] {
		if len(specs) >= 3 {
			break
		}
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return testsEval{tile: t, data: data}
}

type ciEval struct {
	tile CI
	data map[string]any
}

func evalCI(by map[string][]FileMatch, paths map[string]bool) ciEval {
	has := len(by["ci"]) > 0 || paths[".gitlab-ci.yml"] || paths["Jenkinsfile"]
	var yamlText string
	var yamlPath string
	for _, m := range by["ci"] {
		if strings.HasSuffix(m.Path, ".yml") || strings.HasSuffix(m.Path, ".yaml") {
			yamlPath = m.Path
			yamlText = m.Content
			break
		}
	}
	if yamlText == "" {
		for _, m := range by["ci"] {
			yamlPath = m.Path
			yamlText = m.Content
			break
		}
	}
	yl := strings.ToLower(yamlText)

	c := CI{
		HasCI:        has,
		HasLintStep:  has && (strings.Contains(yl, "lint") || strings.Contains(yl, "eslint") || strings.Contains(yl, "golangci")),
		HasTestStep:  has && (strings.Contains(yl, "test") || strings.Contains(yl, "pytest") || strings.Contains(yl, "go test")),
		HasBuildStep: has && (strings.Contains(yl, "build") || strings.Contains(yl, "compile") || strings.Contains(yl, "make")),
	}
	data, _ := structToMap(c)
	var specs []sourceSpec
	if yamlPath != "" && yamlText != "" {
		specs = append(specs, sourceSpec{path: yamlPath, content: yamlText, maxBytes: 1800})
	} else {
		for _, m := range by["ci"] {
			if m.Content != "" {
				specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1800})
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return ciEval{tile: c, data: data}
}

type structureEval struct {
	tile Structure
	data map[string]any
}

func evalStructure(by map[string][]FileMatch, paths map[string]bool) structureEval {
	src := hasTopLevelDir(paths, []string{"src", "lib", "app", "pkg", "cmd"})
	ignore := paths[".gitignore"] || paths[".git/info/exclude"]
	lic := len(by["license"]) > 0 || hasRootLicense(paths)

	s := Structure{
		HasSrcDir:      src,
		HasIgnoreFile:  ignore,
		HasLicenseFile: lic,
	}
	data, _ := structToMap(s)
	var specs []sourceSpec
	// Presence of LICENSE matters for scoring; full text is noise for small-context browser LLMs.
	if p := firstPath(by, "license"); p != "" {
		specs = append(specs, sourceSpec{path: p, content: "", maxBytes: 0})
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return structureEval{tile: s, data: data}
}

type lintEval struct {
	tile Linting
	data map[string]any
}

func evalLinting(by map[string][]FileMatch, paths map[string]bool) lintEval {
	lintFiles := len(by["linting"]) > 0
	pre := paths[".pre-commit-config.yaml"] || paths[".pre-commit-config.yml"]

	l := Linting{
		HasLintConfig:     lintFiles,
		HasPreCommitHooks: pre,
	}
	data, _ := structToMap(l)
	var specs []sourceSpec
	for _, m := range by["linting"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
			if len(specs) >= 2 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return lintEval{tile: l, data: data}
}

type depsEval struct {
	tile Deps
	data map[string]any
}

func evalDeps(by map[string][]FileMatch, paths map[string]bool) depsEval {
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
	for _, m := range by["dependencies"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1500})
			if len(specs) >= 2 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return depsEval{tile: d, data: data}
}

type docsEval struct {
	tile Docs
	data map[string]any
}

func evalDocs(by map[string][]FileMatch, paths map[string]bool) docsEval {
	hasDocsDir := pathPrefixExists(paths, "docs/") || pathPrefixExists(paths, "documentation/")
	changelog := paths["CHANGELOG.md"] || paths["CHANGELOG.rst"] || paths["CHANGELOG"]
	contrib := paths["CONTRIBUTING.md"] || paths["CONTRIBUTING.rst"]
	coc := paths["CODE_OF_CONDUCT.md"] || paths["CODE_OF_CONDUCT"]

	d := Docs{
		HasDocsDir:       hasDocsDir || len(by["docs"]) > 0,
		HasChangelog:     changelog,
		HasContributing:  contrib,
		HasCodeOfConduct: coc,
	}
	data, _ := structToMap(d)
	var specs []sourceSpec
	for _, m := range by["docs"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1200})
			if len(specs) >= 3 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return docsEval{tile: d, data: data}
}

type aiEval struct {
	tile AIReady
	data map[string]any
}

func evalAIReady(by map[string][]FileMatch, paths map[string]bool) aiEval {
	cr := paths[".cursorrules"] || paths[".cursor/rules"]
	claude := paths["CLAUDE.md"] || paths["claude.md"] || paths["Claude.md"]
	copilot := paths[".github/copilot-instructions.md"] || paths["copilot-instructions.md"]
	ign := paths[".cursorignore"]

	a := AIReady{
		HasCursorRules:   cr,
		HasClaudeConfig:  claude,
		HasCopilotConfig: copilot,
		HasAiIgnore:      ign,
	}
	data, _ := structToMap(a)
	var specs []sourceSpec
	for _, m := range by["ai_readability"] {
		if m.Content != "" {
			specs = append(specs, sourceSpec{path: m.Path, content: m.Content, maxBytes: 1500})
			if len(specs) >= 3 {
				break
			}
		}
	}
	addPromptContext(data, promptSourcesForMetric(specs))
	return aiEval{tile: a, data: data}
}

// --- helpers ---

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

type sourceSpec struct {
	path     string
	content  string
	maxBytes int
}

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

func pathPrefixExists(paths map[string]bool, prefix string) bool {
	for p := range paths {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
