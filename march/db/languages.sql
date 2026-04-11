-- ============================================
-- Languages
-- ============================================

-- name: GetOrCreateLanguage :one
INSERT INTO languages (id, name)
VALUES (@id, @name)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetLanguageByName :one
SELECT * FROM languages WHERE name = @name;

-- name: UpsertLanguageBoard :one
INSERT INTO language_boards (id, game_id, language_id, points, commit_count)
VALUES (@id, @game_id, @language_id, @points, @commit_count)
ON CONFLICT ON CONSTRAINT uq_language_game
DO UPDATE SET
    points = language_boards.points + EXCLUDED.points,
    commit_count = language_boards.commit_count + EXCLUDED.commit_count
RETURNING *;

-- name: GetLanguageLeaderboard :many
SELECT lb.*, l.name AS language_name
FROM language_boards lb
JOIN languages l ON l.id = lb.language_id
WHERE lb.game_id = @game_id
ORDER BY lb.points DESC
LIMIT @limit_count;