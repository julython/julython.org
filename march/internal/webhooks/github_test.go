package webhooks_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/services"
	"july/internal/testutil"
	"july/internal/webhooks"
)

func setupHandler(t *testing.T, env *testutil.TestEnv) *webhooks.Handler {
	t.Helper()
	gameService := services.NewGameService(env.Queries)
	return webhooks.NewHandler(env.Queries, gameService)
}

func TestGitHubWebhook(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	handler := setupHandler(t, env)

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

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result webhooks.ProcessResult
		err := json.NewDecoder(rec.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Received)
		assert.Equal(t, 2, result.Created)
		assert.Equal(t, 0, result.Skipped)

		// Verify project was created
		project, err := env.Queries.GetProjectBySlug(t.Context(), "alice/my-project")
		require.NoError(t, err)
		assert.Equal(t, "my-project", project.Name)
		assert.Equal(t, "github", project.Service)
	})

	t.Run("skips duplicate commits", func(t *testing.T) {
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
					ID:        "abc123def456789", // Already exists from previous test
					Message:   "Add user authentication feature",
					Timestamp: time.Now(),
					URL:       "https://github.com/alice/my-project/commit/abc123",
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result webhooks.ProcessResult
		json.NewDecoder(rec.Body).Decode(&result)
		assert.Equal(t, 1, result.Received)
		assert.Equal(t, 1, result.Skipped)
		assert.Equal(t, 0, result.Created)
	})

	t.Run("skips non-default branches", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/feature-branch",
			Repository: webhooks.GitHubRepo{
				ID:       12345,
				FullName: "alice/my-project",
				HTMLURL:  "https://github.com/alice/my-project",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "xyz789",
					Message:   "Feature work in progress",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "not default branch")
	})

	t.Run("skips force pushes", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref:    "refs/heads/main",
			Forced: true,
			Repository: webhooks.GitHubRepo{
				ID:       12345,
				FullName: "alice/my-project",
				HTMLURL:  "https://github.com/alice/my-project",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "forced123",
					Message:   "Rewritten history",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "force push")
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
				{
					ID:        "wip123",
					Message:   "wip",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Bob", Email: "bob@test.com"},
				},
				{
					ID:        "merge123",
					Message:   "Merge branch 'feature' into main",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Bob", Email: "bob@test.com"},
				},
				{
					ID:        "short1",
					Message:   "fix",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Bob", Email: "bob@test.com"},
				},
				{
					ID:        "valid123",
					Message:   "Add proper feature implementation",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Bob", Email: "bob@test.com"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result webhooks.ProcessResult
		json.NewDecoder(rec.Body).Decode(&result)
		assert.Equal(t, 4, result.Received)
		assert.Equal(t, 1, result.Created) // Only valid123
		assert.Equal(t, 3, result.Skipped) // wip, merge, short
	})

	t.Run("creates new project from webhook", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:          77777,
				Name:        "new-repo",
				FullName:    "carol/new-repo",
				HTMLURL:     "https://github.com/carol/new-repo",
				Description: "A brand new repository",
				Fork:        false,
				ForksCount:  5,
				Watchers:    10,
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "newrepo123",
					Message:   "Initial commit with README",
					Timestamp: time.Now(),
					URL:       "https://github.com/carol/new-repo/commit/newrepo123",
					Author:    webhooks.GitHubAuthor{Name: "Carol", Email: "carol@test.com"},
					Added:     []string{"README.md"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify project was created with all fields
		project, err := env.Queries.GetProjectBySlug(t.Context(), "carol/new-repo")
		require.NoError(t, err)
		assert.Equal(t, "new-repo", project.Name)
		assert.Equal(t, "https://github.com/carol/new-repo", project.Url)
		assert.Equal(t, "github", project.Service)
		assert.Equal(t, int64(77777), project.RepoID.Int64)
		assert.False(t, project.Forked)
		assert.Equal(t, int32(5), project.Forks)
		assert.Equal(t, int32(10), project.Watchers)
	})

	t.Run("links commit to existing user", func(t *testing.T) {
		// Create user with email identifier
		user := testutil.CreateUser(t, env, "dave", "Dave Developer")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "dave@test.com", true, true)

		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       88888,
				Name:     "dave-repo",
				FullName: "dave/dave-repo",
				HTMLURL:  "https://github.com/dave/dave-repo",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "dave123",
					Message:   "Add feature from Dave",
					Timestamp: time.Now(),
					URL:       "https://github.com/dave/dave-repo/commit/dave123",
					Author:    webhooks.GitHubAuthor{Name: "Dave Developer", Email: "dave@test.com"},
					Added:     []string{"feature.py"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify commit is linked to user
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "dave123")
		require.NoError(t, err)
		assert.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)
	})

	t.Run("detects languages from files", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       66666,
				Name:     "polyglot",
				FullName: "poly/polyglot",
				HTMLURL:  "https://github.com/poly/polyglot",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "poly123",
					Message:   "Add multiple language files",
					Timestamp: time.Now(),
					URL:       "https://github.com/poly/polyglot/commit/poly123",
					Author:    webhooks.GitHubAuthor{Name: "Poly", Email: "poly@test.com"},
					Added:     []string{"main.go", "app.py", "index.ts"},
					Modified:  []string{"README.md"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "poly123")
		require.NoError(t, err)
		assert.Contains(t, commit.Languages, "Go")
		assert.Contains(t, commit.Languages, "Python")
		assert.Contains(t, commit.Languages, "TypeScript")
	})

	t.Run("skips inactive project", func(t *testing.T) {
		// Create project with repo_id and deactivate it
		project := testutil.CreateProjectWithRepoID(t, env, "inactive-project", "test/inactive-project", "https://github.com/test/inactive-project", 55555)
		err := env.Queries.DeactivateProject(t.Context(), project.ID)
		require.NoError(t, err)

		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       55555,
				Name:     "inactive-project",
				FullName: "test/inactive-project",
				HTMLURL:  "https://github.com/test/inactive-project",
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "inactive123",
					Message:   "This should be skipped",
					Timestamp: time.Now(),
					Author:    webhooks.GitHubAuthor{Name: "Test", Email: "test@test.com"},
				},
			},
		}

		req := makeWebhookRequest(t, payload)
		rec := httptest.NewRecorder()

		handler.HandleGitHubWebhook(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "project inactive")
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
			result := webhooks.DetectLanguage(tt.file)
			assert.Equal(t, tt.expected, result)
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
			commit := webhooks.GitHubCommit{Message: tt.message}
			result := webhooks.IsValidCommit(commit)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func makeWebhookRequest(t *testing.T, payload any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
