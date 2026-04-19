-- ============================================
-- Teams
-- ============================================

-- name: CreateTeam :one
INSERT INTO teams (id, name, slug, description, avatar_url, created_by, is_public)
VALUES (@id, @name, @slug, @description, @avatar_url, @created_by, @is_public)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM teams WHERE id = @id;

-- name: GetTeamBySlug :one
SELECT * FROM teams WHERE slug = @slug;

-- name: UpdateTeam :exec
UPDATE teams SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    is_public = COALESCE(sqlc.narg('is_public'), is_public)
WHERE id = @id;

-- name: UpdateTeamMemberCount :exec
UPDATE teams SET member_count = (
    SELECT COUNT(*) FROM team_members WHERE team_id = @id
) WHERE id = @id;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = @id;

-- name: ListPublicTeams :many
SELECT * FROM teams
WHERE is_public = true
ORDER BY member_count DESC, created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: AddTeamMember :one
INSERT INTO team_members (id, team_id, user_id, role)
VALUES (@id, @team_id, @user_id, @role)
ON CONFLICT ON CONSTRAINT uq_team_member DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = @team_id AND user_id = @user_id;

-- name: GetTeamMember :one
SELECT * FROM team_members WHERE team_id = @team_id AND user_id = @user_id;

-- name: ListTeamMembers :many
SELECT tm.*, u.name, u.username, u.avatar_url
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = @team_id
ORDER BY tm.role DESC, tm.created_at;

-- name: ListUserTeams :many
SELECT t.*, tm.role AS member_role
FROM teams t
JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = @user_id
ORDER BY t.name;

-- name: UpsertTeamBoard :one
INSERT INTO team_boards (id, game_id, team_id, points, member_count)
VALUES (@id, @game_id, @team_id, @points, @member_count)
ON CONFLICT ON CONSTRAINT uq_team_board
DO UPDATE SET
    points = EXCLUDED.points,
    member_count = EXCLUDED.member_count
RETURNING *;

-- name: GetTeamLeaderboard :many
SELECT tb.*, t.name AS team_name, t.slug, t.avatar_url
FROM team_boards tb
JOIN teams t ON t.id = tb.team_id
WHERE tb.game_id = @game_id
ORDER BY tb.points DESC
LIMIT @limit_count;