
-- ============================================
-- Users
-- ============================================

-- name: CreateUser :one
INSERT INTO users (id, name, username, avatar_url, role)
VALUES (@id, @name, @username, @avatar_url, @role)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = @username;

-- name: UpdateUser :exec
UPDATE users SET
    name = COALESCE(sqlc.narg('name'), name),
    username = COALESCE(sqlc.narg('username'), username),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    last_seen = COALESCE(sqlc.narg('last_seen'), last_seen)
WHERE id = @id;

-- name: UpdateUserLastSeen :exec
UPDATE users SET last_seen = now() WHERE id = @id;

-- name: DeactivateUser :exec
UPDATE users SET
    is_active = false,
    is_banned = true,
    banned_at = now(),
    banned_reason = @reason
WHERE id = @id;

-- name: ReactivateUser :exec
UPDATE users SET
    is_active = true,
    is_banned = false,
    banned_at = NULL,
    banned_reason = NULL
WHERE id = @id;

-- name: ListUsers :many
SELECT * FROM users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: ListBannedUsers :many
SELECT * FROM users
WHERE is_banned = true
ORDER BY banned_at DESC
LIMIT @limit_count;

-- ============================================
-- User Identifiers
-- ============================================

-- name: UpsertUserIdentifier :one
INSERT INTO user_identifiers (value, type, user_id, verified, is_primary, data)
VALUES (@value, @type, @user_id, @verified, @is_primary, @data)
ON CONFLICT (value) DO UPDATE SET
    verified = EXCLUDED.verified,
    is_primary = EXCLUDED.is_primary,
    data = EXCLUDED.data
WHERE user_identifiers.user_id = EXCLUDED.user_id
RETURNING *;

-- name: GetUserIdentifier :one
SELECT * FROM user_identifiers WHERE value = @value;

-- name: GetUserIdentifiersByUserID :many
SELECT * FROM user_identifiers
WHERE user_id = @user_id
ORDER BY is_primary DESC, created_at;

-- name: GetUserIdentifiersByType :many
SELECT * FROM user_identifiers
WHERE user_id = @user_id AND type = @type
ORDER BY is_primary DESC, created_at;

-- name: GetVerifiedEmails :many
SELECT * FROM user_identifiers
WHERE user_id = @user_id AND type = 'email' AND verified = true
ORDER BY is_primary DESC;

-- name: FindUserByIdentifier :one
SELECT u.* FROM users u
JOIN user_identifiers ui ON ui.user_id = u.id
WHERE ui.value = @value;

-- name: DeleteUserIdentifier :exec
DELETE FROM user_identifiers WHERE value = @value;

-- name: DeleteUserIdentifiersByUser :exec
DELETE FROM user_identifiers WHERE user_id = @user_id;

-- name: GetUserIdentifierByUserAndType :one
SELECT * FROM user_identifiers
WHERE user_id = @user_id AND type = @type
LIMIT 1;


-- name: GetUserByPasswordIdentifier :one
-- Looks up a user by email identifier so the handler can verify
-- the bcrypt hash stored in data->>'password_hash'.
SELECT u.*
FROM users u
JOIN user_identifiers ui ON ui.user_id = u.id
WHERE ui.type  = 'email'
  AND ui.value = @value   -- format: "email:<address>"
  AND ui.data ? 'password_hash'
LIMIT 1;

-- name: GetPasswordHash :one
-- Returns just the identifier row so the handler can read data->>'password_hash'.
SELECT data
FROM user_identifiers
WHERE type  = 'email'
  AND value = @value
LIMIT 1;