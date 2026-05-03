package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOllamaClient_defaults(t *testing.T) {
	// Ensure env vars are not set.
	t.Setenv("OLLAMA_API_URL", "")
	t.Setenv("I18N_MODEL", "")

	c := NewOllamaClient()
	if c.BaseURL != "http://localhost:11434" {
		t.Errorf("base URL = %q, want %q", c.BaseURL, "http://localhost:11434")
	}
	if c.Model != "llama3.2:3b" {
		t.Errorf("model = %q, want %q", c.Model, "llama3.2:3b")
	}
	if c.HTTP.Timeout != 120*time.Second {
		t.Errorf("timeout = %v, want 120s", c.HTTP.Timeout)
	}
}

func TestNewOllamaClient_env(t *testing.T) {
	t.Setenv("OLLAMA_API_URL", "http://example.com:9999")
	t.Setenv("I18N_MODEL", "mistral")

	c := NewOllamaClient()
	if c.BaseURL != "http://example.com:9999" {
		t.Errorf("base URL = %q, want %q", c.BaseURL, "http://example.com:9999")
	}
	if c.Model != "mistral" {
		t.Errorf("model = %q, want %q", c.Model, "mistral")
	}
}

func TestIsOllamaAvailable(t *testing.T) {
	// Server that responds OK.
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
	}))
	defer okServer.Close()

	if !isOllamaAvailable(context.Background(), okServer.URL) {
		t.Error("expected Ollama to be available")
	}

	// Server that doesn't respond.
	// We use a port that nothing listens on.
	if isOllamaAvailable(context.Background(), "http://localhost:1") {
		t.Error("expected Ollama to be unavailable on unused port")
	}
}

func TestTranslate_success(t *testing.T) {
	want := map[string]string{
		"Home":     "Inicio",
		"SignIn":   "Iniciar sesion",
	}
	resp := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": mapToString(want),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":       10,
			"completion_tokens":   20,
			"total_tokens":        30,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	entries := []missingEntry{
		{key: "Home", value: "Hello"},
		{key: "SignIn", value: "Sign in"},
	}

	result, err := client.Translate(context.Background(), entries, "es")
	if err != nil {
		t.Fatalf("Translate() error: %v", err)
	}

	for k, wantV := range want {
		if gotV, ok := result[k]; !ok {
			t.Errorf("missing key %q in result", k)
		} else if gotV != wantV {
			t.Errorf("result[%q] = %q, want %q", k, gotV, wantV)
		}
	}
}

func TestTranslate_errorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	_, err := client.Translate(context.Background(), []missingEntry{{key: "Home"}}, "es")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestTranslate_emptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
			"usage":   map[string]int{},
		})
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	_, err := client.Translate(context.Background(), []missingEntry{{key: "Home"}}, "es")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestTranslate_invalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json at all"))
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	_, err := client.Translate(context.Background(), []missingEntry{{key: "Home"}}, "es")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestTranslate_preserveFormatSpecifiers(t *testing.T) {
	want := map[string]string{
		"Commits": "%{count} commits",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": mapToString(want),
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	entries := []missingEntry{{key: "Commits", value: "%{count} commits"}}
	result, err := client.Translate(context.Background(), entries, "es")
	if err != nil {
		t.Fatalf("Translate() error: %v", err)
	}
	if got, ok := result["Commits"]; !ok {
		t.Error("missing key Commits")
	} else if got != "%{count} commits" {
		t.Errorf("Commits = %q, want %%{count} commits", got)
	}
}

// mapToString serializes a map to JSON string for embedding in test responses.
func mapToString(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func TestTranslate_ollamaErrorField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
			"error": map[string]interface{}{
				"type":    "model_not_found",
				"code":    404,
				"message": "model 'bad' not found",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "bad", HTTP: &http.Client{Timeout: 10 * time.Second}}
	_, err := client.Translate(context.Background(), []missingEntry{{key: "Home"}}, "es")
	if err == nil {
		t.Fatal("expected error for ollama error field")
	}
}

func TestTranslate_stripsMarkdownFences(t *testing.T) {
	want := map[string]string{"Home": "Inicio"}
	content := "```\n" + mapToString(want) + "\n```"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": content,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":       5,
				"completion_tokens":   10,
				"total_tokens":        15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	entries := []missingEntry{{key: "Home", value: "Hello"}}
	result, err := client.Translate(context.Background(), entries, "es")
	if err != nil {
		t.Fatalf("Translate() error: %v", err)
	}
	if got, ok := result["Home"]; !ok {
		t.Error("missing key Home")
	} else if got != "Inicio" {
		t.Errorf("Home = %q, want %q", got, "Inicio")
	}
}

func TestTranslate_stripsMarkdownFencesUpper(t *testing.T) {
	want := map[string]string{"Home": "Inicio"}
	content := "```JSON\n" + mapToString(want) + "\n```"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": content,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	entries := []missingEntry{{key: "Home", value: "Hello"}}
	result, err := client.Translate(context.Background(), entries, "es")
	if err != nil {
		t.Fatalf("Translate() error: %v", err)
	}
	if got, ok := result["Home"]; !ok {
		t.Error("missing key Home")
	} else if got != "Inicio" {
		t.Errorf("Home = %q, want %q", got, "Inicio")
	}
}

func TestTranslate_errorStatus_readsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": {"message": "model not loaded"}}`))
	}))
	defer server.Close()

	client := &OllamaClient{BaseURL: server.URL, Model: "test", HTTP: &http.Client{Timeout: 10 * time.Second}}
	_, err := client.Translate(context.Background(), []missingEntry{{key: "Home"}}, "es")
	if err == nil {
		t.Fatal("expected error for 502 status")
	}
	if !strings.Contains(err.Error(), "model not loaded") {
		t.Errorf("expected error to contain body: %v", err)
	}
}
