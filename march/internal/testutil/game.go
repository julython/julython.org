package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"july/internal/db"
)

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
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

func CreateCommit(t *testing.T, env *TestEnv, projectID uuid.UUID, hash, message string) db.Commit {
	t.Helper()
	commit, err := env.Queries.CreateCommit(context.Background(), db.CreateCommitParams{
		ID:        db.NewID(),
		Hash:      db.Text(hash),
		ProjectID: projectID,
		UserID:    db.NullUUID(),
		GameID:    db.NullUUID(),
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

// AddCommit creates a commit and runs it through GameService.AddCommit so that
// boards, players, and language_boards are all updated exactly as production does.
func AddCommit(t *testing.T, env *TestEnv, projectID uuid.UUID, userID uuid.UUID, hash, message string, timestamp time.Time, languages []string) db.Commit {
	t.Helper()
	ctx := context.Background()

	commit := CreateCommit(t, env, projectID, hash, message)
	// Run through the service so boards/players/language_boards are populated.
	if err := env.GameService.AddCommit(ctx, commit); err != nil {
		t.Fatalf("game service failed to process commit %s: %v", hash, err)
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
	start := now.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)
	return CreateGame(t, env,
		fmt.Sprintf("Test Game %s", db.NewID().String()[:8]),
		start, end, true,
	)
}

// CreateTestScenario sets up a user with a project and a commit processed
// through the game service so all derived tables are populated.
func CreateTestScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Commit) {
	t.Helper()

	user := CreateUser(t, env, "testuser", "Test User")
	CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
	CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

	project := CreateProject(t, env, "test-project", "https://github.com/testuser/test-project")
	commit := AddCommit(t, env, project.ID, user.ID, "abc123def456", "Initial commit", time.Now(), []string{})

	return user, project, commit
}

// CreateGameScenario sets up a full game scenario. Commits are routed through
// GameService.AddCommit so that players, boards, and language_boards are all
// populated exactly as they would be in production.
func CreateGameScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Game, db.Commit) {
	t.Helper()

	user := CreateUser(t, env, "testuser", "Test User")
	CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
	CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

	project := CreateOwnedProject(t, env, user, "test-project", "https://github.com/testuser/test-project")
	game := CreateActiveGame(t, env)

	// Timestamp inside the active game window so AddCommit picks up the game.
	commit := AddCommit(t, env, project.ID, user.ID,
		fmt.Sprintf("game-%s", db.NewID().String()[:8]),
		"Game commit",
		time.Now(),
		[]string{"Go", "Python"},
	)

	return user, project, game, commit
}

// CreateGameScenarioForUser adds a participant to an existing game.
func CreateGameScenarioForUser(t *testing.T, env *TestEnv, game db.Game, username, name string) (db.User, db.Project) {
	t.Helper()

	user := CreateUser(t, env, username, name)
	project := CreateOwnedProject(t, env, user, "repo", "https://github.com/"+username+"/repo")

	AddCommit(t, env, project.ID, user.ID,
		fmt.Sprintf("%s-%s", username, db.NewID().String()[:8]),
		"commit for "+username,
		time.Now(),
		[]string{"Go", "Python"},
	)

	return user, project
}
