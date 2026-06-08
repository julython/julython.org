-- ============================================
-- Audit Logs
-- ============================================

-- name: CreateAuditLog :one
INSERT INTO audit_logs (id, moderator_id, action, target_type, target_id, reason)
VALUES (@id, @moderator_id, @action, @target_type, @target_id, @reason)
RETURNING *;

-- name: ListAuditLogs :many
SELECT al.*, u.username AS moderator_username
FROM audit_logs al
JOIN users u ON u.id = al.moderator_id
ORDER BY al.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: ListAuditLogsByTarget :many
SELECT al.*, u.username AS moderator_username
FROM audit_logs al
JOIN users u ON u.id = al.moderator_id
WHERE al.target_type = @target_type AND al.target_id = @target_id
ORDER BY al.created_at DESC;