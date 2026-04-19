-- ============================================
-- Reports
-- ============================================

-- name: CreateReport :one
INSERT INTO reports (id, reported_user_id, report_type, reason, status)
VALUES (@id, @reported_user_id, @report_type, @reason, 'pending')
RETURNING *;

-- name: GetReportByID :one
SELECT * FROM reports WHERE id = @id;

-- name: ListPendingReports :many
SELECT r.*, u.username AS reported_username
FROM reports r
LEFT JOIN users u ON u.id = r.reported_user_id
WHERE r.status = 'pending'
ORDER BY r.created_at ASC
LIMIT @limit_count;

-- name: ReviewReport :exec
UPDATE reports SET
    status = @status,
    reviewed_by = @reviewed_by,
    reviewed_at = now(),
    moderator_notes = @moderator_notes
WHERE id = @id;