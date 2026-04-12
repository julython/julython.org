package metrics

import (
	"fmt"
	"regexp"
	"strings"
)

// ChatLLMSystemPrompt is used when the user is asking about the repo (quality, CI, tests, etc.).
const ChatLLMSystemPrompt = `You are a helpful assistant for an open source repository. The user message may include a short excerpt from our automated scan—use it for project-specific answers when it fits the question.

If the excerpt does not cover the question, say so briefly. Be concise. Use markdown (short headings, bullets).

When a primary language is stated, tailor suggestions (tools, idioms, layout) to that ecosystem.

Do not paste or quote license text or long legal boilerplate—reference file paths when useful.`

// ChatLLMSystemPromptGeneral is used for greetings and casual messages: do not treat the chat as a README review.
const ChatLLMSystemPromptGeneral = `You are a friendly assistant chatting with a developer about their open source project.

Respond naturally to what they said. If they only greet you or make small talk, keep it brief—do not evaluate the README, give scores, or summarize scan results unless they ask about the project or documentation.

If they later ask how to improve something, you can draw on any background context in the message. When a primary language is stated, tailor suggestions to that ecosystem.`

// ChatContextInfo describes how the chat prompt was built (for the user prompt text and optional UI).
type ChatContextInfo struct {
	ContextMetric     string // metric key whose data is shown (e.g. readme, ci)
	MatchedMetric     string // non-empty if a keyword matched the user's message
	UsedDefaultReadme bool   // true when no keyword matched and README context is used
	FallbackToReadme  bool   // true when a keyword matched but README data was used (no row for matched metric)
	// GeneralChat is true for greetings / small talk—prompts must not read like a README quality review.
	GeneralChat bool
	// PrimaryLanguage is the repo primary language from L1 (typically from README metric); used to tailor answers.
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

func appendChatPrimaryLanguageLine(b *strings.Builder, lang string) {
	if lang == "" {
		return
	}
	b.WriteString(fmt.Sprintf(
		"Primary language (from scan): **%s** — use tooling, conventions, and examples appropriate for this stack.\n\n",
		lang,
	))
}

var (
	reGenericChatOnly = regexp.MustCompile(`(?i)^\s*(hi|hello|hey|hiya|howdy|yo|sup|thanks|thank you|thx|ty|ok|okay|k|bye|goodbye|cya|later|good morning|good afternoon|good evening|gm|evening|morning)\b[!.?]*\s*$`)
	reShortCasual      = regexp.MustCompile(`^(hi|hello|hey|hiya|thanks|thx|ty)\s+\w+([!.]?)$`)
)

// IsGenericChatMessage reports whether the message is casual small talk rather than a project question.
// In that case we still may attach README scan data as optional background, but we must not frame the reply as a review.
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

// ChatSystemPromptFor returns the system prompt for project chat (general vs repo-focused).
func ChatSystemPromptFor(generalChat bool) string {
	if generalChat {
		return ChatLLMSystemPromptGeneral
	}
	return ChatLLMSystemPrompt
}

// metricKeywordLists maps analysis metric keys to substrings / phrases (matched case-insensitively).
// Longer phrases are scored higher; ambiguous short tokens use word-boundary checks in MatchMetricFromMessage.
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
// It returns ("", false) when no topic matches well—callers should default to README context.
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

// BuildChatLLMUserContent builds the user message for browser chat: scan excerpt + original question.
func BuildChatLLMUserContent(repoName string, info ChatContextInfo, data map[string]any, l1Score int16, userQuestion string) string {
	if data == nil {
		data = map[string]any{}
	}
	var b strings.Builder
	if info.GeneralChat {
		b.WriteString(fmt.Sprintf("## Repository: %s\n\n", repoName))
		appendChatPrimaryLanguageLine(&b, info.PrimaryLanguage)
		b.WriteString("The user sent a casual or short message (not asking you to review or score the repository).\n\n")
		b.WriteString("Reply briefly and naturally. Do **not** praise or critique the README, do **not** mention scan scores, and do **not** list what “good” looks like unless they ask about the project.\n\n")
		b.WriteString("### Optional background (ignore unless they ask something project-related)\n\n")
		writePromptEvidence(&b, data)
		b.WriteString("### User message\n\n")
		b.WriteString(strings.TrimSpace(userQuestion))
		out := b.String()
		if len(out) > maxMetricLLMUserBytes {
			out = truncateUTF8(out, maxMetricLLMUserBytes)
		}
		return out
	}

	title := MetricDisplayName(info.ContextMetric)
	b.WriteString(fmt.Sprintf("## Repository: %s\n\n", repoName))
	appendChatPrimaryLanguageLine(&b, info.PrimaryLanguage)

	switch {
	case info.FallbackToReadme && info.MatchedMetric != "":
		b.WriteString(fmt.Sprintf(
			"We matched your question to **%s**, but there is no stored scan for that area yet—using **README** context instead.\n\n",
			MetricDisplayName(info.MatchedMetric),
		))
	case info.UsedDefaultReadme:
		b.WriteString("**Context:** README (default — your message did not match a specific analysis area).\n\n")
	default:
		b.WriteString(fmt.Sprintf("**Context:** %s (matched from your message).\n\n", title))
	}

	if advice, ok := metricAdvice[info.ContextMetric]; ok {
		b.WriteString(fmt.Sprintf("What “good” looks like for %s: %s\n\n", title, advice))
	}

	if l1Score > 0 {
		b.WriteString(fmt.Sprintf("Automated score for this area: **%d/10** (for context only).\n\n", l1Score))
	} else {
		b.WriteString("_(No score yet for this scan area — run **Rescan analysis (L1)** on the project page if needed.)_\n\n")
	}

	b.WriteString("### Scan evidence\n\n")
	writePromptEvidence(&b, data)

	b.WriteString("### User question\n\n")
	b.WriteString(strings.TrimSpace(userQuestion))

	out := b.String()
	if len(out) > maxMetricLLMUserBytes {
		out = truncateUTF8(out, maxMetricLLMUserBytes)
	}
	return out
}
