package blog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	t.Run("returns posts sorted newest first", func(t *testing.T) {
		posts, err := All()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(posts), 1)

		for i := 0; i < len(posts)-1; i++ {
			assert.True(t, posts[i].Date.After(posts[i+1].Date),
				"post %q (%s) should come before post %q (%s)",
				posts[i].Title, posts[i].Date,
				posts[i+1].Title, posts[i+1].Date,
			)
		}
	})

	t.Run("returns expected posts", func(t *testing.T) {
		posts, err := All()
		require.NoError(t, err)

		slugs := make([]string, len(posts))
		for i, p := range posts {
			slugs[i] = p.Slug
		}
		assert.Contains(t, slugs, "scoring-2026")
	})

	t.Run("parses frontmatter fields", func(t *testing.T) {
		posts, err := All()
		require.NoError(t, err)

		scoring, ok := findPostBySlug(posts, "scoring-2026")
		require.True(t, ok)

		assert.Equal(t, "Scoring Updates", scoring.Title)
		assert.Equal(t, "scoring-2026", scoring.Slug)
		assert.Equal(t, "How do I win this thing?", scoring.Blurb)
	})
}

func TestBySlug(t *testing.T) {
	t.Run("returns post for existing slug", func(t *testing.T) {
		post, err := BySlug("scoring-2026")
		require.NoError(t, err)

		assert.Equal(t, "Scoring Updates", post.Title)
		assert.Equal(t, "scoring-2026", post.Slug)
	})

	t.Run("returns error for missing slug", func(t *testing.T) {
		_, err := BySlug("nonexistent-slug")
		assert.Error(t, err)
	})
}

func TestParseMermaid(t *testing.T) {
	posts, err := All()
	require.NoError(t, err)

	scoring, ok := findPostBySlug(posts, "scoring-2026")
	require.True(t, ok)

	t.Run("detects mermaid blocks", func(t *testing.T) {
		assert.True(t, scoring.HasMermaid)
	})

	t.Run("renders mermaid as HTML", func(t *testing.T) {
		assert.Contains(t, scoring.Body, `<pre class="mermaid"`)
	})
}

func TestParseHTMLContent(t *testing.T) {
	posts, err := All()
	require.NoError(t, err)

	scoring, ok := findPostBySlug(posts, "scoring-2026")
	require.True(t, ok)

	t.Run("renders markdown as HTML", func(t *testing.T) {
		// Headings should be rendered as <h2> tags
		assert.Contains(t, scoring.Body, "<h2")
	})

	t.Run("renders links", func(t *testing.T) {
		assert.Contains(t, scoring.Body, "<a")
	})

	t.Run("renders code blocks", func(t *testing.T) {
		// The blog has inline code and code blocks
		assert.Contains(t, scoring.Body, "<code")
	})

	t.Run("body is non-empty", func(t *testing.T) {
		assert.True(t, len(scoring.Body) > 100)
	})
}

func TestPostDateParsing(t *testing.T) {
	posts, err := All()
	require.NoError(t, err)

	scoring, ok := findPostBySlug(posts, "scoring-2026")
	require.True(t, ok)

	t.Run("parses date correctly", func(t *testing.T) {
		expected := "2026-04-06"
		assert.Equal(t, expected, scoring.Date.Format("2006-01-02"))
	})
}

func findPostBySlug(posts []Post, slug string) (Post, bool) {
	for _, p := range posts {
		if p.Slug == slug {
			return p, true
		}
	}
	return Post{}, false
}

// ── HTTP client tests (integration) ────────────────────────────────
// These require the full test environment (i18n, layout) so they use
// setupTestEnv from testutil. They live here in package blog because
// the testutil import cycle prevents an external test package.
//
// See internal/features/profile/profile_test.go for examples of HTTP
// client tests using SetupTestEnv.
//
// func TestBlogListPageHTTP(t *testing.T) {
// 	env := testutil.SetupTestEnv(t)
// 	resp, err := env.Client.Get(env.Server.URL + "/blog")
// 	require.NoError(t, err)
// 	defer resp.Body.Close()
// 	require.Equal(t, http.StatusOK, resp.StatusCode)
// 	body, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)
// 	assert.Contains(t, string(body), "Blog")
// }
//
// func TestBlogPostPageHTTP(t *testing.T) {
// 	env := testutil.SetupTestEnv(t)
// 	resp, err := env.Client.Get(env.Server.URL + "/blog/scoring-2026")
// 	require.NoError(t, err)
// 	defer resp.Body.Close()
// 	require.Equal(t, http.StatusOK, resp.StatusCode)
// 	body, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)
// 	b := strings.ToLower(string(body))
// 	assert.Contains(t, b, "scoring updates")
// 	assert.Contains(t, b, "how do i win this thing")
// 	assert.Contains(t, b, "mermaid.initialize")
// 	assert.NotContains(t, b, "--- date:")
// }
