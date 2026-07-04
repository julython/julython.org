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

-- name: GetPlayerWithBoards :many
-- Fetches a single player's info along with their up-to-3 boards and
-- project details in one query via a lateral join.  Returns 0-3 rows
-- (one per board), with user columns repeated across rows.
SELECT
    u.username, u.name, u.avatar_url,
    b.id, b.project_id, COALESCE(b.points, 0), COALESCE(b.verified_points, 0), COALESCE(b.commit_count, 0),
    COALESCE(b.project_name, ''), COALESCE(b.slug, '')
FROM players p
JOIN users u ON u.id = p.user_id
  AND p.game_id = @game_id
  AND u.username = @username
  AND u.is_active = true
LEFT JOIN LATERAL (
    SELECT boards.id, boards.project_id, boards.points, boards.verified_points, boards.commit_count,
           projects.name AS project_name, projects.slug
    FROM boards
    JOIN projects ON projects.id = boards.project_id
    WHERE boards.id = ANY(ARRAY[p.board_1_id, p.board_2_id, p.board_3_id])
) b ON true
WHERE b.id IS NOT NULL;

-- name: UpdatePlayerAnalysis :exec
UPDATE players SET
    verified_points = @verified_points,
    analysis_status = @analysis_status,
    last_analyzed_at = now()
WHERE id = @id;

-- name: GetLeaderboard :many
SELECT
    p.id, p.game_id, p.user_id,
    p.points, p.potential_points, p.verified_points,
    p.commit_count, p.project_count, p.analysis_status,
    p.last_analyzed_at, p.created_at, p.updated_at,
    p.board_1_id, p.board_2_id, p.board_3_id,
    u.name, u.username, u.avatar_url,
    COALESCE(board_total.total, 0) AS board_total
FROM players p
JOIN users u ON u.id = p.user_id AND p.game_id = @game_id AND u.is_active = true
LEFT JOIN LATERAL (
    SELECT COALESCE(
        SUM(CASE WHEN b.verified_points > 0 THEN b.verified_points ELSE b.points END),
        0
    )::int AS total
    FROM boards b
    WHERE b.id = ANY(ARRAY[p.board_1_id, p.board_2_id, p.board_3_id])
) AS board_total ON true
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

-- ============================================
-- Player Boards (up to 3 active boards)
-- ============================================

-- name: AssignBoards :one
-- Update a player's up-to-3 board slots.  NULL arguments leave existing
-- columns untouched, so callers can update 1, 2, or all 3 boards in one
-- query without re-reading the player row first.
UPDATE players
    SET board_1_id = COALESCE(@board_1_id, board_1_id),
        board_2_id = COALESCE(@board_2_id, board_2_id),
        board_3_id = COALESCE(@board_3_id, board_3_id)
WHERE id = @player_id
RETURNING *;

-- name: UnlinkBoard :one
-- Remove a specific board from one of a player's board slots.
-- Handles NULL comparisons using IS NOT DISTINCT FROM (SQL standard).
-- Returns 0 rows if the board isn't in any of the player's slots.
UPDATE players
    SET board_1_id = CASE WHEN board_1_id IS NOT DISTINCT FROM @delete_board THEN NULL ELSE board_1_id END,
        board_2_id = CASE WHEN board_2_id IS NOT DISTINCT FROM @delete_board THEN NULL ELSE board_2_id END,
        board_3_id = CASE WHEN board_3_id IS NOT DISTINCT FROM @delete_board THEN NULL ELSE board_3_id END
WHERE id = @player_id
  AND @delete_board IS NOT DISTINCT FROM board_1_id
    OR @delete_board IS NOT DISTINCT FROM board_2_id
    OR @delete_board IS NOT DISTINCT FROM board_3_id
RETURNING *;

-- name: GetPlayerBoardIds :one
-- Return the 3 board IDs for a player.  Callers can join boards on
-- these IDs when displaying the leaderboard.
SELECT p.board_1_id, p.board_2_id, p.board_3_id
FROM players p
WHERE p.id = @player_id;

-- name: GetBoardByID :one
SELECT * FROM boards WHERE id = @id;

-- name: GetPlayerBoardTotal :one
-- Return the sum of L1-scanned verified_points for all boards assigned
-- to a player, falling back to initial points when unverified.
SELECT COALESCE(
    SUM(
        CASE WHEN b.verified_points > 0 THEN b.verified_points ELSE b.points END
    ), 0)::int AS total
FROM boards b
WHERE b.id = @board_1_id OR b.id = @board_2_id OR b.id = @board_3_id;
