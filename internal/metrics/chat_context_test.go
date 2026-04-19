package metrics

import (
	"strings"
	"testing"
)

func TestMatchMetricFromMessage_CI(t *testing.T) {
	m, ok := MatchMetricFromMessage("how do I setup ci for this repo?")
	if !ok || m != "ci" {
		t.Fatalf("got %q, %v", m, ok)
	}
	m, ok = MatchMetricFromMessage("GitHub Actions workflow for PRs")
	if !ok || m != "ci" {
		t.Fatalf("got %q, %v", m, ok)
	}
}

func TestMatchMetricFromMessage_DefaultNoMatch(t *testing.T) {
	_, ok := MatchMetricFromMessage("what is this project about?")
	if ok {
		t.Fatal("expected no metric match")
	}
}

func TestMatchMetricFromMessage_ReadmeKeywords(t *testing.T) {
	m, ok := MatchMetricFromMessage("improve the readme introduction")
	if !ok || m != "readme" {
		t.Fatalf("got %q, %v", m, ok)
	}
}

func TestBuildChatLLMUserContent_IncludesQuestion(t *testing.T) {
	data := map[string]any{
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "README.md", "snippet": "# Hi"},
			},
		},
	}
	info := ChatContextInfo{
		TopicMetric: "readme", UsedDefaultTopic: true, GeneralChat: false, NoScanEvidence: false,
	}
	out := BuildChatLLMUserContent("o/r", info, data, "How do I improve the intro?")
	if !strings.Contains(out, "How do I improve the intro?") {
		t.Fatalf("missing question: %s", out)
	}
	if !strings.Contains(out, "default") {
		t.Fatalf("expected default topic hint: %s", out)
	}
}

func TestBuildChatLLMUserContent_NoScanEvidence_FollowUps(t *testing.T) {
	info := ChatContextInfo{
		TopicMetric: "ci", KeywordMatched: true, NoScanEvidence: true, PrimaryLanguage: "Python",
	}
	out := BuildChatLLMUserContent("o/r", info, map[string]any{}, "how do I setup ci")
	if !strings.Contains(out, "No files from our scan") {
		t.Fatalf("expected no-evidence line: %s", out)
	}
	if !strings.Contains(out, "Python") || !strings.Contains(out, "Try asking next") {
		t.Fatalf("expected language + follow-ups: %s", out)
	}
}

func TestLanguageFromData(t *testing.T) {
	if got := LanguageFromData(nil); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := LanguageFromData(map[string]any{"language": "  Rust  "}); got != "Rust" {
		t.Fatalf("got %q", got)
	}
}

func TestScanEvidenceNonEmpty(t *testing.T) {
	if ScanEvidenceNonEmpty(nil) || ScanEvidenceNonEmpty(map[string]any{}) {
		t.Fatal()
	}
	if !ScanEvidenceNonEmpty(map[string]any{"has_ci": true}) {
		t.Fatal()
	}
}

func TestIsGenericChatMessage(t *testing.T) {
	if !IsGenericChatMessage("hello") || !IsGenericChatMessage("Hi!") || !IsGenericChatMessage("thanks") {
		t.Fatal("expected generic")
	}
	if IsGenericChatMessage("how do I set up CI?") || IsGenericChatMessage("what does this project do?") {
		t.Fatal("expected not generic")
	}
}

func TestBuildChatLLMUserContent_GeneralChat(t *testing.T) {
	data := map[string]any{
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "README.md", "snippet": "# Great project"},
			},
		},
	}
	info := ChatContextInfo{TopicMetric: "readme", GeneralChat: true, PrimaryLanguage: "Python"}
	out := BuildChatLLMUserContent("o/r", info, data, "hello")
	if strings.Contains(out, "What “good” looks like") || strings.Contains(out, "Automated score") {
		t.Fatalf("general chat should not include review framing: %s", out)
	}
	if !strings.Contains(out, "### Context") || !strings.Contains(out, "### Message") || !strings.Contains(out, "hello") {
		t.Fatalf("expected general template: %s", out)
	}
}

func TestChatExpertSystemPrompt(t *testing.T) {
	sp := ChatExpertSystemPrompt(ChatContextInfo{TopicMetric: "ci", PrimaryLanguage: "Python", GeneralChat: false})
	if !strings.Contains(sp, "Python") || !strings.Contains(sp, "CI/CD") {
		t.Fatalf("unexpected: %s", sp)
	}
	sp2 := ChatExpertSystemPrompt(ChatContextInfo{GeneralChat: true})
	if sp2 != ChatLLMSystemPromptGeneral {
		t.Fatalf("got %q", sp2)
	}
}
