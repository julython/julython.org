-- ============================================
-- Boards (Project Scores)
-- ============================================

-- name: UpsertBoard :one
INSERT INTO boards (id, game_id, project_id, points, potential_points, commit_count, contributor_count)
VALUES (@id, @game_id, @project_id, @points, @points, @commit_count, @contributor_count)
ON CONFLICT ON CONSTRAINT uq_board_project_game
DO UPDATE SET
    points = boards.points + EXCLUDED.points,
    potential_points = boards.potential_points + EXCLUDED.points,
    commit_count = boards.commit_count + EXCLUDED.commit_count
RETURNING *;

-- name: GetBoardByProjectAndGame :one
SELECT * FROM boards WHERE project_id = @project_id AND game_id = @game_id;

-- name: GetProjectLeaderboard :many
SELECT b.*, p.name AS project_name, p.url AS project_url, p.slug
FROM boards b
JOIN projects p ON p.id = b.project_id
WHERE b.game_id = @game_id AND p.is_active = true
ORDER BY
    CASE WHEN b.verified_points > 0 THEN b.verified_points ELSE b.potential_points END DESC,
    b.points DESC
LIMIT @limit_count;