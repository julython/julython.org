package blog_test

import (
	"io"
	"july/internal/testutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlogListPageHTTP(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	resp, err := env.Client.Get(env.Server.URL + "/blog")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Blog")
}

func TestBlogPostPageHTTP(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	resp, err := env.Client.Get(env.Server.URL + "/blog/scoring-2026")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	b := strings.ToLower(string(body))
	assert.Contains(t, b, "scoring updates")
	assert.Contains(t, b, "how do i win this thing")
	assert.NotContains(t, b, "--- date:")
}

func TestBlogPostGoRewriteHTTP(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	resp, err := env.Client.Get(env.Server.URL + "/blog/rewrite-python-go")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	b := strings.ToLower(string(body))
	assert.Contains(t, b, "python to go")
	assert.Contains(t, b, "why rewrite julython?")
	assert.Contains(t, b, "mermaid.initialize")
	assert.NotContains(t, b, "--- date:")
}
