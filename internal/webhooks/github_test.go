package webhooks_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
	"july/internal/webhooks"
)

func TestMain(m *testing.M) {
	if err := testutil.SetupSharedEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "shared setup failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// PayloadOpts allows callers to override specific fields of the default payload.
type PayloadOpts struct {
	Ref         string
	Forced      bool
	RepoID      int64
	RepoName    string
	FullName    string
	HTMLURL     string
	Description string
	Fork        bool
	ForksCount  int
	Watchers    int
	Author      webhooks.GitHubAuthor
	Files       []string
	Message     string
	Timestamp   time.Time
}

// webhookPayload builds a minimal valid GitHubPushEvent. Hash is required;
// opts allow callers to override fields without constructing the full struct.
func webhookPayload(hash string, opts ...func(*PayloadOpts)) webhooks.GitHubPushEvent {
	o := &PayloadOpts{
		Ref:      "refs/heads/main",
		RepoID:   12345,
		RepoName: "test-repo",
		FullName: "alice/test-repo",
		HTMLURL:  "https://github.com/alice/test-repo",
		Author:   webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
		Message:  "Add a meaningful change",
		Files:    []string{"main.go"},
	}
	if o.Timestamp.IsZero() {
		o.Timestamp = time.Now()
	}
	for _, opt := range opts {
		opt(o)
	}
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
		Commits: []webhooks.GitHubCommit{
			{
				ID:        hash,
				Message:   o.Message,
				Timestamp: o.Timestamp,
				URL:       o.HTMLURL + "/commit/" + hash,
				Author:    o.Author,
				Added:     o.Files,
			},
		},
	}
}

func TestGitHubWebhook(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	t.Run("processes valid push event", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       12345,
				Name:     "my-project",
				FullName: "alice/my-project",
				HTMLURL:  "https://github.com/alice/my-project",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "abc123def456789",
					Message:   "Add user authentication feature",
					Timestamp: time.Now(),
					URL:       "https://github.com/alice/my-project/commit/abc123",
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
					Added:     []string{"auth.go", "auth_test.go"},
					Modified:  []string{"main.go"},
				},
				{
					ID:        "def456abc789012",
					Message:   "Fix bug in login flow",
					Timestamp: time.Now(),
					URL:       "https://github.com/alice/my-project/commit/def456",
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
					Modified:  []string{"auth.go"},
				},
			},
		}

		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 2, result.Received)
		assert.Equal(t, 2, result.Created)
		assert.Equal(t, 0, result.Skipped)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-alice-my-project")
		require.NoError(t, err)
		assert.Equal(t, "my-project", project.Name)
		assert.Equal(t, "github", project.Service)
	})

	t.Run("skips duplicate commits", func(t *testing.T) {
		testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("duplicate-hash-001"))

		resp := testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("duplicate-hash-001"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Received)
		assert.Equal(t, 1, result.Skipped)
		assert.Equal(t, 0, result.Created)
	})

	t.Run("skips non-default branches", func(t *testing.T) {
		payload := webhookPayload("branch-hash-001", func(o *PayloadOpts) {
			o.Ref = "refs/heads/feature-branch"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "not default branch")
	})

	t.Run("skips force pushes", func(t *testing.T) {
		payload := webhookPayload("force-hash-001", func(o *PayloadOpts) {
			o.Forced = true
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "force push")
	})

	t.Run("skips invalid commits", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       99999,
				Name:     "test-repo",
				FullName: "bob/test-repo",
				HTMLURL:  "https://github.com/bob/test-repo",
			},
			Commits: []webhooks.GitHubCommit{
				{ID: "wip-001", Message: "wip", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "merge-001", Message: "Merge branch 'feature' into main", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "short-001", Message: "fix", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "valid-001", Message: "Add proper feature implementation", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
			},
		}
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 4, result.Received)
		assert.Equal(t, 1, result.Created)
		assert.Equal(t, 3, result.Skipped)
	})

	t.Run("creates new project from webhook", func(t *testing.T) {
		payload := webhookPayload("newrepo-hash-001", func(o *PayloadOpts) {
			o.RepoID = 77777
			o.RepoName = "new-repo"
			o.FullName = "carol/new-repo"
			o.HTMLURL = "https://github.com/carol/new-repo"
			o.Description = "A brand new repository"
			o.ForksCount = 5
			o.Watchers = 10
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-carol-new-repo")
		require.NoError(t, err)
		assert.Equal(t, "new-repo", project.Name)
		assert.Equal(t, "https://github.com/carol/new-repo", project.Url)
		assert.Equal(t, "github", project.Service)
		assert.Equal(t, int64(77777), project.RepoID.Int64)
		assert.False(t, project.Forked)
		assert.Equal(t, int32(5), project.Forks)
		assert.Equal(t, int32(10), project.Watchers)
	})

	t.Run("adds commit to the active game", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)
		payload := webhookPayload("game-hash-001", func(o *PayloadOpts) {
			o.RepoID = 77777
			o.FullName = "carol/new-repo"
			o.HTMLURL = "https://github.com/carol/new-repo"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commits, err := env.Queries.GetRecentCommits(t.Context(), db.GetRecentCommitsParams{
			GameID:     db.UUID(game.ID),
			LimitCount: 10,
		})
		require.NoError(t, err)
		assert.Len(t, commits, 1)
	})

	t.Run("links commit to existing user", func(t *testing.T) {
		user := testutil.CreateUser(t, env, "dave", "Dave Developer")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "dave@test.com", true, true)

		payload := webhookPayload("dave-hash-001", func(o *PayloadOpts) {
			o.RepoID = 88888
			o.RepoName = "dave-repo"
			o.FullName = "dave/dave-repo"
			o.HTMLURL = "https://github.com/dave/dave-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Dave Developer", Email: "dave@test.com"}
			o.Files = []string{"feature.py"}
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "dave-hash-001")
		require.NoError(t, err)
		assert.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)
	})

	t.Run("detects languages from files", func(t *testing.T) {
		payload := webhookPayload("poly-hash-001", func(o *PayloadOpts) {
			o.RepoID = 66666
			o.RepoName = "polyglot"
			o.FullName = "poly/polyglot"
			o.HTMLURL = "https://github.com/poly/polyglot"
			o.Author = webhooks.GitHubAuthor{Name: "Poly", Email: "poly@test.com"}
			o.Files = []string{"main.go", "app.py", "index.ts", "README.md"}
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "poly-hash-001")
		require.NoError(t, err)
		assert.Contains(t, commit.Languages, "Go")
		assert.Contains(t, commit.Languages, "Python")
		assert.Contains(t, commit.Languages, "TypeScript")
	})

	t.Run("skips inactive project", func(t *testing.T) {
		project := testutil.CreateProjectWithRepoID(t, env, "inactive-project", "gh-test-inactive-project", "https://github.com/test/inactive-project", 55555)
		require.NoError(t, env.Queries.DeactivateProject(t.Context(), project.ID))

		payload := webhookPayload("inactive-hash-001", func(o *PayloadOpts) {
			o.RepoID = 55555
			o.RepoName = "inactive-project"
			o.FullName = "test/inactive-project"
			o.HTMLURL = "https://github.com/test/inactive-project"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "project inactive")
	})

	t.Run("handles ping event", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader("{}"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "ping")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "pong", testutil.DecodeBody(t, resp))
	})
}

func TestGitHubWebhookContentTypes(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	t.Run("accepts application/json", func(t *testing.T) {
		resp := testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("ct-json-001"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)
	})

	t.Run("accepts form-encoded payload", func(t *testing.T) {
		payload := webhookPayload("ct-form-001")
		b, err := json.Marshal(payload)
		require.NoError(t, err)

		form := url.Values{"payload": {string(b)}}
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)
	})

	t.Run("rejects unsupported content type", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader("data"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("rejects form without payload field", func(t *testing.T) {
		form := url.Values{"wrong_field": {"data"}}
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "Go"},
		{"app.py", "Python"},
		{"index.ts", "TypeScript"},
		{"component.tsx", "TypeScript"},
		{"script.js", "JavaScript"},
		{"style.css", "CSS"},
		{"lib.rs", "Rust"},
		{"Main.java", "Java"},
		{"README.md", ""},
		{"Makefile", ""},
		{".gitignore", ""},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, webhooks.DetectLanguage(tt.file))
		})
	}
}

func TestIsValidCommit(t *testing.T) {
	tests := []struct {
		name    string
		message string
		valid   bool
	}{
		{"normal commit", "Add user authentication", true},
		{"wip commit", "wip: working on feature", false},
		{"WIP uppercase", "WIP", false},
		{"merge commit", "Merge branch 'feature' into main", false},
		{"short message", "fix", false},
		{"skip ci", "Update docs [skip ci]", false},
		{"ci skip", "Minor change [ci skip]", false},
		{"valid short", "Fix login bug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, webhooks.IsValidCommit(webhooks.GitHubCommit{Message: tt.message}))
		})
	}
}
