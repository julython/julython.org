package testutil

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// BodyContains reads the response body and asserts each fragment is present.
// On failure it prints a short plain-text summary instead of raw HTML:
// the status code, the content-type, and any non-tag lines near the miss.
func BodyContains(t *testing.T, resp *http.Response, fragments ...string) string {
	t.Helper()
	body := DecodeBody(t, resp)
	for _, f := range fragments {
		if !strings.Contains(body, f) {
			assert.Fail(t, "fragment not found in response",
				"looking for: %q\nstatus:      %d\ncontent-type: %s\ntext lines:\n%s",
				f,
				resp.StatusCode,
				resp.Header.Get("Content-Type"),
				textLines(body, 100),
			)
		}
	}
	return body
}

// textLines strips HTML tags and returns up to n non-empty lines of visible text.
func textLines(html string, n int) string {
	var b strings.Builder
	count := 0
	inTag := false
	var line strings.Builder

	flush := func() {
		s := strings.TrimSpace(line.String())
		if s != "" && count < n {
			b.WriteString("  ")
			b.WriteString(s)
			b.WriteByte('\n')
			count++
		}
		line.Reset()
	}

	for _, ch := range html {
		switch {
		case ch == '<':
			inTag = true
			flush()
		case ch == '>':
			inTag = false
		case ch == '\n':
			if !inTag {
				flush()
			}
		case !inTag:
			line.WriteRune(ch)
		}
	}
	flush()
	return b.String()
}
