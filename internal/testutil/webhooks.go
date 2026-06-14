package testutil

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"july/internal/webhooks"
)

// WebhookOpts overrides fields on the default webhook payload.
// Zero values are ignored — only explicitly set fields are applied.
type WebhookOpts struct {
	Ref         string
	Forced      bool
	RepoID      int64
	RepoName    string
	FullName    string
	HTMLURL     string
	Private     bool
	Fork        bool
	ForksCount  int
	Watchers    int
	Description string
	Author      webhooks.GitHubAuthor
	Username    string
	AvatarURL   string
	Files       []string
	Message     string
	Timestamp   time.Time
}

// buildAuthor applies optional username and avatar overrides to an author.
func buildAuthor(base webhooks.GitHubAuthor, username string, avatarURL string) webhooks.GitHubAuthor {
	if username != "" {
		base.Username = username
	}
	if avatarURL != "" {
		base.AvatarURL = avatarURL
	}
	return base
}

// WebhookPayload builds a minimal valid GitHubPushEvent with sensible defaults.
// Apply option funcs to override specific fields.
func WebhookPayload(hash string, opts ...func(*WebhookOpts)) webhooks.GitHubPushEvent {
	author := webhooks.GitHubAuthor{Name: "Test User", Email: "test@example.com"}
	o := &WebhookOpts{
		Ref:      "refs/heads/main",
		RepoID:   12345,
		RepoName: "test-repo",
		FullName: "testuser/test-repo",
		HTMLURL:  "https://github.com/testuser/test-repo",
		Author:   author,
		Message:  "Add a meaningful change",
		Files:    []string{"main.go"},
	}
	for _, opt := range opts {
		opt(o)
	}
	author = buildAuthor(o.Author, o.Username, o.AvatarURL)
	return webhooks.GitHubPushEvent{
		Ref:    o.Ref,
		Forced: o.Forced,
		Repository: webhooks.GitHubRepo{
			ID:          o.RepoID,
			Name:        o.RepoName,
			FullName:    o.FullName,
			HTMLURL:     o.HTMLURL,
			Description: o.Description,
			Fork:        o.Fork,
			ForksCount:  o.ForksCount,
			Watchers:    o.Watchers,
		},
		Commits: []webhooks.GitHubCommit{{
			ID:        hash,
			Message:   o.Message,
			Timestamp: o.Timestamp,
			URL:       o.HTMLURL + "/commit/" + hash,
			Author:    author,
			Added:     o.Files,
		}},
	}
}

// WebhookCommit posts a single commit through the real webhook endpoint and
// asserts it was created. This exercises the full pipeline: webhook handler →
// project upsert → language detection → game service → boards/players.
//
// Returns the decoded ProcessResult so callers can make further assertions.
func WebhookCommit(t *testing.T, env *TestEnv, hash string, opts ...func(*WebhookOpts)) webhooks.ProcessResult {
	t.Helper()
	payload := WebhookPayload(hash, opts...)
	resp := PostJSON(t, env, "/api/v1/github", payload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "webhook POST failed for hash %s", hash)

	var result webhooks.ProcessResult
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, 1, result.Created, "expected commit %s to be created, got: %+v", hash, result)
	return result
}
