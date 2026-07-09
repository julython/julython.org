package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeRedirect(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"empty string", "", "/"},
		{"valid path", "/profile", "/profile"},
		{"path with query", "/profile?tab=1", "/profile?tab=1"},
		{"path with fragment", "/profile#section", "/profile#section"},
		{"external https", "https://evil.com", "/"},
		{"external http", "http://evil.com", "/"},
		{"protocol-relative", "//evil.com", "/"},
		{"javascript", "javascript:alert(1)", "/"},
		{"data URL", "data:text/html,<script>", "/"},
		{"backslash bypass", "\\evil.com", "/"},
		{"absolute path no slash", "profile", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeRedirect(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
