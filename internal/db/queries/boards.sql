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

-- name: UpdateBoardVerifiedPoints :one
UPDATE boards
SET verified_points = @verified_points
WHERE id = @board_id
RETURNING *;

-- name: GetBoardByIDsAndGame :many
SELECT * FROM boards WHERE id = ANY(@board_ids::uuid[]) AND game_id = @game_id;

-- name: GetBoardByProjectAndGame :one
SELECT * FROM boards WHERE project_id = @project_id AND game_id = @game_id;

-- name: GetProjectLeaderboard :many
SELECT b.*, p.name AS project_name, p.url AS project_url, p.slug
FROM boards b
JOIN projects p ON p.id = b.project_id
WHERE b.game_id = @game_id AND p.is_active = true AND p.is_private = false
ORDER BY
    CASE WHEN b.verified_points > 0 THEN b.verified_points ELSE b.potential_points END DESC,
    b.points DESC
LIMIT @limit_count;

-- name: ValidateBoardOwnership :one
SELECT b.id, b.project_id
FROM boards b
JOIN projects p ON p.id = b.project_id
WHERE b.id = @board_id
  AND b.game_id = @game_id
  AND p.is_private = false
  AND p.owner = @owner
LIMIT 1;
