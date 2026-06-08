-- ============================================
-- Games
-- ============================================

-- name: CreateGame :one
INSERT INTO games (id, name, starts_at, ends_at, commit_points, project_points, is_active)
VALUES (@id, @name, @starts_at, @ends_at, @commit_points, @project_points, @is_active)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM games WHERE id = @id;

-- name: GetActiveGame :one
SELECT * FROM games
WHERE is_active = true AND starts_at <= @now AND ends_at >= @now
ORDER BY starts_at DESC
LIMIT 1;

-- name: GetActiveGameAtTime :one
SELECT * FROM games
WHERE is_active = true AND starts_at <= @timestamp AND ends_at >= @timestamp
ORDER BY starts_at DESC
LIMIT 1;

-- name: GetLatestGame :one
SELECT * FROM games
WHERE ends_at <= @now
ORDER BY ends_at DESC
LIMIT 1;

-- name: ListGames :many
SELECT * FROM games
ORDER BY starts_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: DeactivateAllGames :exec
UPDATE games SET is_active = false WHERE is_active = true;

-- name: ActivateGame :exec
UPDATE games SET is_active = true WHERE id = @id;

-- name: DeactivateGame :exec
UPDATE games SET is_active = false WHERE id = @id;
