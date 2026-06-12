package testutil

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/webhooks"
)

// ownerFromRepoURL extracts the owner from a GitHub-style repo URL.
func ownerFromRepoURL(repoURL string) string {
	// Handle https://github.com/owner/repo
	const githubPrefix = "https://github.com/"
	const gitPrefix = "git@github.com:"
	if strings.HasPrefix(repoURL, githubPrefix) {
		rest := strings.TrimPrefix(repoURL, githubPrefix)
		if slash := strings.Index(rest, "/"); slash >= 0 {
			return strings.SplitN(rest, "/", 2)[0]
		}
	}
	if strings.HasPrefix(repoURL, gitPrefix) {
		rest := strings.TrimPrefix(repoURL, gitPrefix)
		if slash := strings.Index(rest, "/"); slash >= 0 {
			return strings.SplitN(rest, "/", 2)[0]
		}
	}
	return ""
}

func CreateProject(t *testing.T, env *TestEnv, slug, repoURL string) db.Project {
	t.Helper()
	project, err := env.Queries.CreateProject(context.Background(), db.CreateProjectParams{
		ID:          db.NewID(),
		Url:         repoURL,
		Name:        slug,
		Slug:        slug,
		Description: db.NullText(),
		RepoID:      pgtype.Int8{},
		Service:     "github",
		Forked:      false,
		Forks:       0,
		Watchers:    0,
		ParentUrl:   db.NullText(),
		IsPrivate:   false,
		Owner:       ownerFromRepoURL(repoURL),
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

func CreateOwnedProject(t *testing.T, env *TestEnv, owner db.User, repo, repoURL string) db.Project {
	t.Helper()
	slug := fmt.Sprintf("gh-%s-%s", owner.Username, repo)
	return CreateProject(t, env, slug, repoURL)
}

func CreateProjectWithRepoID(t *testing.T, env *TestEnv, name, slug, repoURL string, repoID int64) db.Project {
	t.Helper()
	project, err := env.Queries.CreateProject(context.Background(), db.CreateProjectParams{
		ID:          db.NewID(),
		Url:         repoURL,
		Name:        name,
		Slug:        slug,
		Description: db.NullText(),
		RepoID:      db.BigInt(repoID),
		Service:     "github",
		Forked:      false,
		Forks:       0,
		Watchers:    0,
		ParentUrl:   db.NullText(),
		IsPrivate:   false,
		Owner:       ownerFromRepoURL(repoURL),
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

// CreateCommit inserts a raw commit row with no game association and no service
// processing. Use this only when the test doesn't care about scoring or boards
// (e.g. webhook ingestion tests, project detail pages).
func CreateCommit(t *testing.T, env *TestEnv, projectID uuid.UUID, hash, message string) db.Commit {
	t.Helper()
	return CreateCommitWithUser(t, env, projectID, hash, message, uuid.Nil, uuid.Nil)
}

// CreateCommitWithUser inserts a commit with explicit user_id and game_id.
// This is useful for tests that need to drive the player board assignment flow.
func CreateCommitWithUser(
	t *testing.T, env *TestEnv, projectID uuid.UUID, hash, message string,
	userID, gameID uuid.UUID,
) db.Commit {
	t.Helper()
	commit, err := env.Queries.CreateCommit(context.Background(), db.CreateCommitParams{
		ID:        db.NewID(),
		Hash:      db.Text(hash),
		ProjectID: projectID,
		UserID:    db.NullableUUID(userID),
		GameID:    db.NullableUUID(gameID),
		Author:    db.Text("Test Author"),
		Email:     db.Text("test@example.com"),
		Message:   message,
		Url:       fmt.Sprintf("https://github.com/owner/repo/commit/%s", hash),
		Timestamp: time.Now(),
		Languages: []string{},
		Files:     []byte("[]"),
	})
	if err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}
	return commit
}

func CreateGame(t *testing.T, env *TestEnv, name string, startAt, endAt time.Time, isActive bool) db.Game {
	t.Helper()
	game, err := env.Queries.CreateGame(context.Background(), db.CreateGameParams{
		ID:            db.NewID(),
		Name:          name,
		StartsAt:      startAt,
		EndsAt:        endAt,
		CommitPoints:  1,
		ProjectPoints: 10,
		IsActive:      isActive,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	return game
}

func CreateActiveGame(t *testing.T, env *TestEnv) db.Game {
	t.Helper()
	now := time.Now().UTC()
	return CreateGame(t, env,
		fmt.Sprintf("Test Game %s", db.NewID().String()[:8]),
		now.Add(-24*time.Hour),
		now.Add(24*time.Hour),
		true,
	)
}

// uniqueName extracts a unique identifier from the test name for use in
// generating unique usernames/project slugs across concurrent test runs.
// The result is truncated to 25 characters to fit the database username limit.
func uniqueName(t *testing.T, suffix string) string {
	// t.Name() returns e.g. "TestFoo/SubTest" — take the last segment
	parts := strings.Split(t.Name(), "/")
	name := parts[len(parts)-1] + suffix
	const maxLen = 25
	if len(name) > maxLen {
		return name[:maxLen]
	}
	return name
}

// CreateTestScenario sets up a user with a project and a bare commit row.
// No game association — use CreateGameScenario when scoring matters.
func CreateTestScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Commit) {
	t.Helper()

	name := uniqueName(t, "_user")
	user := CreateUser(t, env, name, "Test User")
	CreateUserIdentifier(t, env, user.ID, "email", name+"@example.com", true, true)
	CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

	project := CreateProject(t, env, "test-project", "https://github.com/name/test-project")
	commit := CreateCommit(t, env, project.ID, "abc123def456", "Initial commit")

	return user, project, commit
}

// CreateGameScenario posts a commit through the real webhook endpoint so the
// full pipeline runs: project upsert, language detection, game association,
// and board/player scoring.
func CreateGameScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Game, db.Commit) {
	t.Helper()
	ctx := context.Background()

	name := uniqueName(t, "_user")
	user := CreateUser(t, env, name, "Test User")
	CreateUserIdentifier(t, env, user.ID, "email", name+"@example.com", true, true)
	CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

	game := CreateActiveGame(t, env)

	hash := fmt.Sprintf("game-%s", db.NewID().String()[:8])
	WebhookCommit(t, env, hash, func(o *WebhookOpts) {
		o.RepoID = 11111
		o.RepoName = name + "-project"
		o.FullName = name + "/test-project"
		o.HTMLURL = "https://github.com/" + name + "/test-project"
		o.Author = webhooks.GitHubAuthor{Name: user.Name, Email: name + "@example.com"}
		o.Files = []string{"main.go", "app.py"}
		o.Message = "Initial game commit"
	})

	slug := fmt.Sprintf("gh-%s-test-project", name)
	project, err := env.Queries.GetProjectBySlug(ctx, slug)
	require.NoError(t, err, "project should have been created by webhook")

	commit, err := env.Queries.GetCommitByHashStr(ctx, hash)
	require.NoError(t, err, "commit should have been created by webhook")

	return user, project, game, commit
}

// CreateGameScenarioForUser adds a new participant to an existing game by
// posting through the webhook endpoint. The user's email identifier is
// registered first so the commit is linked to them automatically.
func CreateGameScenarioForUser(t *testing.T, env *TestEnv, game db.Game, username, name string) (db.User, db.Project) {
	t.Helper()
	ctx := context.Background()

	// username is now caller-controlled (test-name-based), so no collision.
	user := CreateUser(t, env, username, name)
	CreateUserIdentifier(t, env, user.ID, "email", username+"@example.com", true, true)

	repoName := username + "-repo"
	hash := fmt.Sprintf("%s-%s", username, db.NewID().String()[:8])
	WebhookCommit(t, env, hash, func(o *WebhookOpts) {
		o.RepoID = int64(db.NewID().ID())
		o.RepoName = repoName
		o.FullName = username + "/" + repoName
		o.HTMLURL = "https://github.com/" + username + "/" + repoName
		o.Author = webhooks.GitHubAuthor{Name: name, Email: username + "@example.com"}
		o.Files = []string{"main.go", "app.py"}
		o.Message = "Commit for " + username
	})

	slug := fmt.Sprintf("gh-%s-%s", username, repoName)
	project, err := env.Queries.GetProjectBySlug(ctx, slug)
	require.NoError(t, err, "project should have been created by webhook for %s", username)

	return user, project
}
