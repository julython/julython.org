-- ============================================
-- Commits
-- ============================================

-- name: CreateCommit :one
INSERT INTO commits (id, hash, project_id, user_id, game_id, author, email, message, url, timestamp, languages, files)
VALUES (@id, @hash, @project_id, @user_id, @game_id, @author, @email, @message, @url, @timestamp, @languages, @files)
ON CONFLICT (hash) DO NOTHING
RETURNING *;

-- name: GetCommitByHash :one
SELECT * FROM commits WHERE hash = @hash;

-- name: GetCommitByID :one
SELECT * FROM commits WHERE id = @id;

-- name: GetCommitsByProject :many
SELECT * FROM commits
WHERE project_id = @project_id
ORDER BY timestamp DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: GetCommitsByUserAndGame :many
SELECT * FROM commits
WHERE user_id = @user_id AND game_id = @game_id
ORDER BY timestamp DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: ClaimOrphanCommits :execrows
UPDATE commits SET user_id = @user_id
WHERE user_id IS NULL
  AND email = ANY(@emails::text[])
  AND game_id = @game_id;

-- name: CountUserCommitsForGame :one
SELECT
    COUNT(*)::int AS commit_count,
    COUNT(DISTINCT project_id)::int AS project_count
FROM commits
WHERE user_id = @user_id AND game_id = @game_id;

-- name: FlagCommit :exec
UPDATE commits SET
    is_flagged = true,
    flag_reason = @flag_reason
WHERE id = @id;

-- name: VerifyCommit :exec
UPDATE commits SET is_verified = true WHERE id = @id;

-- name: GetUnverifiedCommits :many
SELECT c.*, p.url AS project_url, p.name AS project_name
FROM commits c
JOIN projects p ON p.id = c.project_id
WHERE c.is_verified = false AND c.is_flagged = false
ORDER BY c.created_at ASC
LIMIT @limit_count;

-- name: GetCommitStats :one
SELECT
    COUNT(*) AS total_commits,
    COUNT(DISTINCT user_id) FILTER (WHERE user_id IS NOT NULL) AS unique_users,
    COUNT(DISTINCT project_id) AS unique_projects
FROM commits
WHERE game_id = @game_id;

-- name: GetDailyCommitCounts :many
SELECT
    DATE(timestamp) AS commit_date,
    COUNT(*)::int AS commit_count
FROM commits
WHERE game_id = @game_id
GROUP BY DATE(timestamp)
ORDER BY commit_date;

-- name: GetRecentCommits :many
SELECT
    c.id,
    c.hash,
    c.message,
    c.timestamp,
    c.author,
    u.username,
    u.avatar_url,
    p.slug AS project_slug,
    p.name AS project_name
FROM commits c
LEFT JOIN users u ON u.id = c.user_id
JOIN projects p ON p.id = c.project_id
WHERE c.game_id = @game_id
ORDER BY c.timestamp DESC
LIMIT @limit_count;
