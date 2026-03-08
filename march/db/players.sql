-- ============================================
-- Players
-- ============================================

-- name: UpsertPlayer :one
INSERT INTO players (id, game_id, user_id, points, potential_points, commit_count, project_count, analysis_status)
VALUES (@id, @game_id, @user_id, @points, @points, @commit_count, @project_count, @analysis_status)
ON CONFLICT ON CONSTRAINT uq_player_user_game
DO UPDATE SET
    points = EXCLUDED.points,
    potential_points = EXCLUDED.potential_points,
    commit_count = EXCLUDED.commit_count,
    project_count = EXCLUDED.project_count
RETURNING *;

-- name: GetPlayerByID :one
SELECT * FROM players WHERE id = @id;

-- name: GetPlayerByUserAndGame :one
SELECT * FROM players WHERE user_id = @user_id AND game_id = @game_id;

-- name: UpdatePlayerAnalysis :exec
UPDATE players SET
    verified_points = @verified_points,
    analysis_status = @analysis_status,
    last_analyzed_at = now()
WHERE id = @id;

-- name: GetLeaderboard :many
SELECT p.*, u.name, u.username, u.avatar_url
FROM players p
JOIN users u ON u.id = p.user_id
WHERE p.game_id = @game_id AND u.is_active = true
ORDER BY
    CASE WHEN p.verified_points > 0 THEN p.verified_points ELSE p.potential_points END DESC,
    p.points DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: GetPlayerRank :one
SELECT COUNT(*) + 1 AS rank
FROM players p
JOIN users u ON u.id = p.user_id
WHERE p.game_id = @game_id
  AND u.is_active = true
  AND (
    CASE WHEN p.verified_points > 0 THEN p.verified_points ELSE p.potential_points END
  ) > (
    SELECT CASE WHEN p2.verified_points > 0 THEN p2.verified_points ELSE p2.potential_points END
    FROM players p2 WHERE p2.id = @player_id
  );
