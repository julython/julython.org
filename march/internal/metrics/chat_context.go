package metrics

import (
	"fmt"
	"regexp"
	"strings"
)

// ChatLLMSystemPromptGeneral is for greetings / casual messages (short).
const ChatLLMSystemPromptGeneral = `You are a friendly assistant for developers. Keep replies brief and natural.`

// ChatExpertSystemPrompt returns a short role-focused system prompt for the project assistant.
func ChatExpertSystemPrompt(info ChatContextInfo) string {
	if info.GeneralChat {
		return ChatLLMSystemPromptGeneral
	}
	topic := expertTopicPhrase(info.TopicMetric)
	if info.PrimaryLanguage != "" {
		return fmt.Sprintf(
			"You are an expert %s developer focused on %s. Give short, practical answers. Use markdown when helpful.",
			info.PrimaryLanguage, topic,
		)
	}
	return fmt.Sprintf(
		"You are an expert in %s for open source projects. Give short, practical answers. Use markdown when helpful.",
		topic,
	)
}

func expertTopicPhrase(metric string) string {
	switch metric {
	case "readme":
		return "READMEs and project documentation"
	case "tests":
		return "automated testing"
	case "ci":
		return "CI/CD pipelines"
	case "structure":
		return "repository layout and project structure"
	case "linting":
		return "linting and code style"
	case "deps":
		return "dependencies and supply chain"
	case "docs":
		return "technical documentation"
	case "ai_ready":
		return "AI assistant and agent configuration"
	default:
		return "this repository area"
	}
}

// ChatContextInfo describes how the chat prompt was built.
type ChatContextInfo struct {
	// TopicMetric is what the user is asking about (keyword match or "readme" by default).
	TopicMetric string
	// KeywordMatched is true when MatchMetricFromMessage found a topic.
	KeywordMatched bool
	// MatchedKeyword is the metric key when KeywordMatched (e.g. "ci").
	MatchedKeyword string
	// UsedDefaultTopic is true when no keyword matched and we default to README topic.
	UsedDefaultTopic bool
	// NoScanEvidence is true when we have no L1 snippets/heuristics for TopicMetric (answer generically).
	NoScanEvidence bool
	// GeneralChat is true for greetings / small talk.
	GeneralChat bool
	// PrimaryLanguage from L1 (usually README metric) when available.
	PrimaryLanguage string
}

// LanguageFromData returns the primary language from L1 metric JSON (language key).
func LanguageFromData(data map[string]any) string {
	if data == nil {
		return ""
	}
	v, ok := data[LanguageKey].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

var (
	reGenericChatOnly = regexp.MustCompile(`(?i)^\s*(hi|hello|hey|hiya|howdy|yo|sup|thanks|thank you|thx|ty|ok|okay|k|bye|goodbye|cya|later|good morning|good afternoon|good evening|gm|evening|morning)\b[!.?]*\s*$`)
	reShortCasual      = regexp.MustCompile(`^(hi|hello|hey|hiya|thanks|thx|ty)\s+\w+([!.]?)$`)
)

// IsGenericChatMessage reports whether the message is casual small talk rather than a project question.
func IsGenericChatMessage(msg string) bool {
	s := strings.TrimSpace(msg)
	if s == "" {
		return true
	}
	if reGenericChatOnly.MatchString(s) {
		return true
	}
	lower := strings.Trim(strings.ToLower(s), "!?.")
	if len(lower) <= 24 && !strings.ContainsAny(lower, "?") && reShortCasual.MatchString(lower) {
		return true
	}
	return false
}

// metricFollowUpSuggestions gives concrete next questions the user can try (helps small models).
func metricFollowUpSuggestions(metric string) []string {
	switch metric {
	case "ci":
		return []string{
			"How do I set up GitHub Actions for this repository?",
			"How do I configure GitLab CI?",
			"How do I add a CircleCI config?",
		}
	case "tests":
		return []string{
			"How should I structure unit tests here?",
			"How do I run the test suite locally?",
		}
	case "readme":
		return []string{
			"What should this README include for new contributors?",
			"How do I add install instructions?",
		}
	case "linting":
		return []string{
			"How do I set up ESLint / Ruff / golangci-lint for this stack?",
			"How do I run the formatter in CI?",
		}
	case "deps":
		return []string{
			"How should I pin dependencies safely?",
			"How do I enable Dependabot or Renovate?",
		}
	case "docs":
		return []string{
			"How do I add API docs for this project?",
			"Should I use a docs/ site or wiki?",
		}
	case "structure":
		return []string{
			"What folder layout fits this language ecosystem?",
			"Where should entry points and libraries live?",
		}
	case "ai_ready":
		return []string{
			"What should go in CLAUDE.md or AGENTS.md?",
			"How do I document conventions for AI tools?",
		}
	default:
		return nil
	}
}

// metricKeywordLists maps analysis metric keys to substrings / phrases (matched case-insensitively).
var metricKeywordLists = map[string][]string{
	"readme": {"readme", "readme.md", "project description", "getting started", "introduction", "badge", "badges"},
	"tests":  {"testing", "pytest", "jest", "mocha", "coverage", "unit test", "integration test", "tdd", "vitest", "cypress", "go test"},
	"ci": {"github actions", "gitlab ci", "gitlab-ci", "continuous integration", "pipeline", "workflow",
		"jenkins", "circleci", "buildkite", "travis", "jenkinsfile", ".gitlab-ci", "ci/cd", "ci cd"},
	"structure": {"structure", "layout", "folder", "directory", "monorepo", "project layout", "entrypoint", "entry point"},
	"linting": {"linter", "eslint", "prettier", "golangci", "formatter", "format code", "style check", "ruff", "black", "flake8"},
	"deps": {"dependenc", "dependabot", "renovate", "lockfile", "package.json", "go.mod", "cargo.toml", "pyproject", "npm", "yarn", "pnpm", "vulnerability"},
	"docs": {"documentation", "doc site", "mkdocs", "sphinx", "apidoc", "wiki", "changelog", "contributing"},
	"ai_ready": {"claude.md", "agents.md", "cursor", "copilot", ".cursorrules", "cursorrules", "ai assistant", "llm context"},
}

var (
	reCIWord    = regexp.MustCompile(`\b(ci|cd)\b|ci\s*/\s*cd`)
	reReadmeFN  = regexp.MustCompile(`readme\.(md|rst|txt)\b`)
	reTestsWord = regexp.MustCompile(`\btests?\b|pytest|jest|mocha|vitest|cypress`)
	reLintWord  = regexp.MustCompile(`\blint\b|eslint|prettier|ruff|flake8|golangci`)
	reDocsWord  = regexp.MustCompile(`\bdocs\b|documentation|mkdocs|sphinx`)
)

// MatchMetricFromMessage returns a metric key when the message appears to ask about that analysis area.
func MatchMetricFromMessage(msg string) (metric string, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return "", false
	}

	scores := map[string]int{}
	for m, kws := range metricKeywordLists {
		for _, kw := range kws {
			if strings.Contains(lower, kw) {
				scores[m] += len(kw) + 2
			}
		}
	}

	if reCIWord.MatchString(lower) {
		scores["ci"] += 10
	}
	if reReadmeFN.MatchString(lower) {
		scores["readme"] += 12
	}
	if reTestsWord.MatchString(lower) {
		scores["tests"] += 12
	}
	if reLintWord.MatchString(lower) {
		scores["linting"] += 10
	}
	if reDocsWord.MatchString(lower) {
		scores["docs"] += 10
	}

	best := ""
	bestScore := 0
	order := []string{"readme", "tests", "ci", "structure", "linting", "deps", "docs", "ai_ready"}
	for _, m := range order {
		s := scores[m]
		if s > bestScore {
			bestScore = s
			best = m
		}
	}
	if bestScore < 6 {
		return "", false
	}
	return best, true
}

// ScanEvidenceNonEmpty returns true if data has prompt_context sources or heuristic flags besides language.
func ScanEvidenceNonEmpty(data map[string]any) bool {
	if data == nil {
		return false
	}
	if pc, ok := data[PromptContextKey].(map[string]any); ok {
		if sources, ok := pc["sources"].([]any); ok && len(sources) > 0 {
			return true
		}
	}
	for k := range data {
		if k == PromptContextKey || k == LanguageKey {
			continue
		}
		return true
	}
	return false
}

// BuildChatLLMUserContent builds the user message: repo line, optional scan snippets, question, optional follow-ups.
func BuildChatLLMUserContent(repoName string, info ChatContextInfo, data map[string]any, userQuestion string) string {
	if data == nil {
		data = map[string]any{}
	}
	var b strings.Builder

	if info.GeneralChat {
		b.WriteString(fmt.Sprintf("Repository: **%s**\n\n", repoName))
		if info.PrimaryLanguage != "" {
			b.WriteString(fmt.Sprintf("Stack (from scan): **%s**\n\n", info.PrimaryLanguage))
		}
		b.WriteString("### Context\n\n")
		if ScanEvidenceNonEmpty(data) {
			writePromptEvidence(&b, data)
		} else {
			b.WriteString("_No scan excerpts attached._\n\n")
		}
		b.WriteString("### Message\n\n")
		b.WriteString(strings.TrimSpace(userQuestion))
		out := b.String()
		if len(out) > maxMetricLLMUserBytes {
			out = truncateUTF8(out, maxMetricLLMUserBytes)
		}
		return out
	}

	b.WriteString(fmt.Sprintf("Repository: **%s**\n", repoName))
	b.WriteString(fmt.Sprintf("Topic: **%s**", MetricDisplayName(info.TopicMetric)))
	if info.UsedDefaultTopic {
		b.WriteString(" _(default — question did not match a specific area)_")
	}
	b.WriteString("\n\n")

	if info.NoScanEvidence {
		b.WriteString("### Context\n\n")
		b.WriteString("_No files from our scan for this topic yet._ Give **general, stack-appropriate** guidance")
		if info.PrimaryLanguage != "" {
			b.WriteString(fmt.Sprintf(" for **%s**", info.PrimaryLanguage))
		}
		b.WriteString(". Keep it short.\n\n")
		if ups := metricFollowUpSuggestions(info.TopicMetric); len(ups) > 0 {
			b.WriteString("**Try asking next:**\n")
			for _, u := range ups {
				b.WriteString(fmt.Sprintf("- %s\n", u))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("### Context (from our scan)\n\n")
		writePromptEvidence(&b, data)
	}

	b.WriteString("### Question\n\n")
	b.WriteString(strings.TrimSpace(userQuestion))

	out := b.String()
	if len(out) > maxMetricLLMUserBytes {
		out = truncateUTF8(out, maxMetricLLMUserBytes)
	}
	return out
}
