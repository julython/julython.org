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
ORDER BY id DESC
LIMIT GREATEST(@limit_count, 1) OFFSET @offset_count;

-- name: GetCommitsByUserAndGame :many
SELECT * FROM commits
WHERE user_id = @user_id AND game_id = @game_id
ORDER BY id DESC
LIMIT GREATEST(@limit_count, 1) OFFSET @offset_count;

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
LIMIT GREATEST(@limit_count, 1);

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
LIMIT GREATEST(@limit_count, 1);

-- name: SetCommitGame :exec
UPDATE commits
SET game_id = @game_id
WHERE id = @id;

-- name: GetProjectGameActivityAggregates :one
WITH month_commits AS (
    SELECT m.files, m.timestamp
    FROM commits m
    WHERE m.project_id = @project_id
      AND m.game_id = @game_id
      AND m.timestamp >= @month_start
)
SELECT
    (SELECT COUNT(*)::int FROM month_commits) AS commits_this_month,
    (
        SELECT COUNT(*)::int
        FROM commits w
        WHERE w.project_id = @project_id
          AND w.game_id = @game_id
          AND w.timestamp >= @week_start
    ) AS commits_this_week,
    CAST(
        COALESCE(
            (
                SELECT SUM(
                    CASE
                        WHEN jsonb_typeof(mc.files) = 'array' THEN jsonb_array_length(mc.files)
                        ELSE 0
                    END
                )
                FROM month_commits mc
            ),
            0
        ) AS int
    ) AS file_touch_count,
    CAST(
        COALESCE(
            (
                SELECT COUNT(DISTINCT dir_path)
                FROM (
                    SELECT
                        CASE
                            WHEN elem->>'file' IS NULL OR elem->>'file' = '' THEN NULL
                            WHEN position('/' IN elem->>'file') = 0 THEN '.'
                            ELSE regexp_replace(elem->>'file', '/[^/]+$', '')
                        END AS dir_path
                    FROM month_commits c
                    CROSS JOIN LATERAL jsonb_array_elements(
                        CASE
                            WHEN jsonb_typeof(c.files) = 'array' THEN c.files
                            ELSE '[]'::jsonb
                        END
                    ) AS elem
                ) paths
                WHERE dir_path IS NOT NULL
            ),
            0
        ) AS int
    ) AS unique_dirs;