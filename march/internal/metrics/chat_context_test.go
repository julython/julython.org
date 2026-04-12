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
	info := ChatContextInfo{ContextMetric: "readme", UsedDefaultReadme: true, GeneralChat: false, PrimaryLanguage: "Go"}
	out := BuildChatLLMUserContent("o/r", info, data, 5, "How do I improve the intro?")
	if !strings.Contains(out, "How do I improve the intro?") {
		t.Fatalf("missing question: %s", out)
	}
	if !strings.Contains(out, "default") {
		t.Fatalf("expected default readme hint")
	}
	if !strings.Contains(out, "Primary language") || !strings.Contains(out, "Go") {
		t.Fatalf("expected primary language line: %s", out)
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

func TestIsGenericChatMessage(t *testing.T) {
	if !IsGenericChatMessage("hello") || !IsGenericChatMessage("Hi!") || !IsGenericChatMessage("thanks") {
		t.Fatal("expected generic")
	}
	if IsGenericChatMessage("how do I set up CI?") || IsGenericChatMessage("what does this project do?") {
		t.Fatal("expected not generic")
	}
}

func TestBuildChatLLMUserContent_GeneralChat_NoReviewFraming(t *testing.T) {
	data := map[string]any{
		PromptContextKey: map[string]any{
			"sources": []any{
				map[string]any{"path": "README.md", "snippet": "# Great project"},
			},
		},
	}
	info := ChatContextInfo{ContextMetric: "readme", UsedDefaultReadme: true, GeneralChat: true, PrimaryLanguage: "Python"}
	out := BuildChatLLMUserContent("o/r", info, data, 9, "hello")
	if strings.Contains(out, "What “good” looks like") || strings.Contains(out, "Automated score") {
		t.Fatalf("general chat should not include review framing: %s", out)
	}
	if !strings.Contains(out, "Optional background") || !strings.Contains(out, "hello") {
		t.Fatalf("expected general template: %s", out)
	}
	if !strings.Contains(out, "Python") {
		t.Fatalf("expected primary language in general chat: %s", out)
	}
}

func TestChatSystemPromptFor(t *testing.T) {
	if ChatSystemPromptFor(true) != ChatLLMSystemPromptGeneral {
		t.Fatal()
	}
	if ChatSystemPromptFor(false) != ChatLLMSystemPrompt {
		t.Fatal()
	}
}
