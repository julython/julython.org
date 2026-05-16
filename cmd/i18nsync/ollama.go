package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ollamaChatRequest is the request body for Ollama's /v1/chat/completions endpoint.
type ollamaChatRequest struct {
	Model    string               `json:"model"`
	Messages []ollamaChatMessage  `json:"messages"`
	Stream   bool                 `json:"stream"`
}

// ollamaChatMessage represents a message in the chat request.
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatResponse is the non-streamed response from Ollama's /v1/chat/completions endpoint.
type ollamaChatResponse struct {
	Choices []ollamaChoice        `json:"choices"`
	Usage   ollamaUsage           `json:"usage"`
	Error   *ollamaError          `json:"error,omitempty"`
}

// ollamaChoice represents a single choice in the v1 response.
type ollamaChoice struct {
	Index        int                 `json:"index"`
	Message      ollamaChatMessage   `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

// ollamaUsage provides token usage statistics.
type ollamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ollamaError carries error details from the v1 API.
type ollamaError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code,omitempty"`
}

// OllamaClient wraps connection details for a local Ollama instance.
type OllamaClient struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

// NewOllamaClient creates a client configured from environment variables.
// Defaults: OLLAMA_API_URL=http://localhost:11434, I18N_MODEL=llama3.2:3b.
func NewOllamaClient() *OllamaClient {
	url := os.Getenv("OLLAMA_API_URL")
	if url == "" {
		url = "http://localhost:11434"
	}
	model := os.Getenv("I18N_MODEL")
	if model == "" {
		model = "llama3.2:3b"
	}
	return &OllamaClient{
		BaseURL: url,
		Model:   model,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Translate calls Ollama to translate the given missing entries into targetLang.
// Returns a map[key]string of translated values.
// For keys not returned by Ollama, the map will not contain them (caller falls back).
// On Ollama failure, returns nil, error so the caller can fall back.
func (c *OllamaClient) Translate(ctx context.Context, entries []missingEntry, targetLang string) (map[string]string, error) {
	// Build prompt with all missing key-value pairs.
	var pairs strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&pairs, "%s: %s\n", e.key, e.value)
	}

	prompt := fmt.Sprintf(`You are a professional translator. Translate the following English text strings into %s.

Rules:
- Return ONLY a valid JSON object mapping each key to its translated value.
- Preserve format specifiers like %%{count}, %%{link}, etc. exactly as they appear.
- Do not add any extra text, explanation, or markdown formatting — only the JSON object.

English source:
%s

Target language: %s

JSON output:`, targetLang, pairs.String(), targetLang)

	reqBody := ollamaChatRequest{
		Model:  c.Model,
		Stream: false,
		Messages: []ollamaChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the non-streamed response.
	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Check for explicit error from the v1 API.
	if ollamaResp.Error != nil {
		return nil, fmt.Errorf("Ollama error (%s, code %v): %s",
			ollamaResp.Error.Type, ollamaResp.Error.Code, ollamaResp.Error.Message)
	}

	if len(ollamaResp.Choices) == 0 {
		return nil, fmt.Errorf("Ollama returned empty choices")
	}
	content := ollamaResp.Choices[0].Message.Content
	finishReason := ollamaResp.Choices[0].FinishReason

	// Strip markdown code fences if present.
	jsonStr := content
	jsonStr = strings.TrimPrefix(jsonStr, "```\n")
	jsonStr = strings.TrimPrefix(jsonStr, "```json\n")
	jsonStr = strings.TrimPrefix(jsonStr, "```JSON\n")
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```JSON")
	jsonStr = strings.TrimSuffix(jsonStr, "\n```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	// Trim any surrounding whitespace.
	jsonStr = strings.TrimSpace(jsonStr)

	// Log raw content for debugging.
	fmt.Printf("[i18nsync] Ollama raw response: %q\n", jsonStr)

	// Parse the JSON object from the model's response.
	var result map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse translation JSON: %w", err)
	}

	fmt.Printf("[i18nsync] Ollama %s — prompt: %d tokens, completion: %d tokens, total: %d tokens, finish_reason: %s\n",
		c.Model, ollamaResp.Usage.PromptTokens,
		ollamaResp.Usage.CompletionTokens, ollamaResp.Usage.TotalTokens,
		finishReason)

	return result, nil
}

// isOllamaAvailable checks if Ollama is reachable without making a full translation call.
func isOllamaAvailable(ctx context.Context, baseURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// logf is a helper for logging warnings to stderr.
func logf(format string, args ...any) {
	log.Printf("[i18nsync] "+format, args...)
}
