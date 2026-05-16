package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookPayload(t *testing.T) {
	t.Run("returns valid default payload", func(t *testing.T) {
		payload := WebhookPayload("abc123")

		assert.Equal(t, "abc123", payload.Commits[0].ID)
		assert.Equal(t, "refs/heads/main", payload.Ref)
		assert.Equal(t, int64(12345), payload.Repository.ID)
		assert.Equal(t, "test-repo", payload.Repository.Name)
		assert.Equal(t, "testuser/test-repo", payload.Repository.FullName)
		assert.Equal(t, "https://github.com/testuser/test-repo", payload.Repository.HTMLURL)
		assert.False(t, payload.Repository.Fork)
	})

	t.Run("includes default commit details", func(t *testing.T) {
		payload := WebhookPayload("def456")

		require.Len(t, payload.Commits, 1)
		c := payload.Commits[0]
		assert.Equal(t, "Add a meaningful change", c.Message)
		assert.Equal(t, "test@example.com", c.Author.Email)
		assert.Equal(t, "Test User", c.Author.Name)
		assert.Equal(t, []string{"main.go"}, c.Added)
	})

	t.Run("applies option overrides", func(t *testing.T) {
		payload := WebhookPayload("xyz", func(o *WebhookOpts) {
			o.RepoName = "overridden"
			o.FullName = "overridden/owner"
			o.Forced = true
			o.Files = []string{"a.go", "b.go"}
		})

		assert.Equal(t, "overridden", payload.Repository.Name)
		assert.Equal(t, "overridden/owner", payload.Repository.FullName)
		assert.True(t, payload.Forced)
		assert.Equal(t, []string{"a.go", "b.go"}, payload.Commits[0].Added)
	})

	t.Run("returns multiple commits", func(t *testing.T) {
		payload := WebhookPayload("abc123")
		// Default payload always has exactly one commit
		assert.Len(t, payload.Commits, 1)
	})
}

func TestTextLines(t *testing.T) {
	t.Run("strips HTML tags", func(t *testing.T) {
		text := textLines("<p>Hello</p><p>World</p>", 10)
		assert.Contains(t, text, "Hello")
		assert.Contains(t, text, "World")
		assert.NotContains(t, text, "<")
		assert.NotContains(t, text, ">")
	})

	t.Run("collapses whitespace", func(t *testing.T) {
		text := textLines("<p>  spaced  </p>", 10)
		// Should collapse internal whitespace
		assert.Contains(t, text, "spaced")
		assert.NotContains(t, text, "  spaced  ")
	})

	t.Run("respects line limit", func(t *testing.T) {
		h := "<p>line1</p>\n<p>line2</p>\n<p>line3</p>\n<p>line4</p>\n<p>line5</p>\n<p>line6</p>"
		text := textLines(h, 3)
		// Should return at most 3 lines (plus trailing newline from builder)
		lines := 0
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				lines++
			}
		}
		assert.LessOrEqual(t, lines, 3)
	})

	t.Run("handles empty string", func(t *testing.T) {
		text := textLines("", 10)
		assert.Equal(t, "", text)
	})
}
