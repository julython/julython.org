package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
)

func TestText(t *testing.T) {
	t.Run("valid text", func(t *testing.T) {
		t.Parallel()
		got := db.Text("hello")
		assert.True(t, got.Valid)
		assert.Equal(t, "hello", got.String)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		got := db.Text("")
		assert.True(t, got.Valid)
		assert.Equal(t, "", got.String)
	})
}

func TestNullText(t *testing.T) {
	t.Parallel()
	got := db.NullText()
	assert.False(t, got.Valid)
}

func TestBigInt(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		got := db.BigInt(42)
		assert.True(t, got.Valid)
		assert.Equal(t, int64(42), got.Int64)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		got := db.BigInt(-100)
		assert.True(t, got.Valid)
		assert.Equal(t, int64(-100), got.Int64)
	})
}

func TestUUID(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	got := db.UUID(id)
	assert.True(t, got.Valid)
	assert.Equal(t, id[:], got.Bytes[:])
}

func TestNullUUID(t *testing.T) {
	t.Parallel()
	got := db.NullUUID()
	assert.False(t, got.Valid)
}

func TestStringFromNull(t *testing.T) {
	t.Run("valid returns string", func(t *testing.T) {
		t.Parallel()
		text := pgtype.Text{String: "hello", Valid: true}
		assert.Equal(t, "hello", db.StringFromNull(text))
	})

	t.Run("invalid returns empty", func(t *testing.T) {
		t.Parallel()
		text := pgtype.Text{Valid: false}
		assert.Equal(t, "", db.StringFromNull(text))
	})
}

func TestGetCommitByHashStr(t *testing.T) {
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)

	hash := "abc123def456"

	t.Run("returns error for missing commit", func(t *testing.T) {
		t.Parallel()
		_, err := env.Queries.GetCommitByHashStr(ctx, hash)
		assert.Error(t, err)
	})

	t.Run("inserts and retrieves commit", func(t *testing.T) {
		t.Parallel()
		project := testutil.CreateProject(t, env, "commit-test-project", "https://github.com/test/commit-test")
		user := testutil.CreateUser(t, env, "commit-testuser", "Commit Test User")

		commitSha := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
		_, err := env.Queries.CreateCommit(ctx, db.CreateCommitParams{
			ID:       uuid.New(),
			Hash:     db.Text(commitSha),
			ProjectID: project.ID,
			UserID:   db.UUID(user.ID),
			Author:   db.Text("test"),
			Email:    db.Text("test@test.com"),
			Message:  "test commit",
			Url:      "https://github.com/test/commit/" + commitSha,
			Timestamp: time.Now(),
		})
		require.NoError(t, err)

		commit, err := env.Queries.GetCommitByHashStr(ctx, commitSha)
		require.NoError(t, err)
		assert.Equal(t, commitSha, commit.Hash.String)
		assert.Equal(t, project.ID, commit.ProjectID)
	})
}
