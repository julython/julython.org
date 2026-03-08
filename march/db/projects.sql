-- ============================================
-- Projects
-- ============================================

-- name: CreateProject :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers, parent_url)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers, @parent_url)
RETURNING *;

-- name: UpsertProjectByRepoID :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers)
ON CONFLICT (service, repo_id) WHERE repo_id IS NOT NULL
DO UPDATE SET
    url = EXCLUDED.url,
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    description = EXCLUDED.description,
    forks = EXCLUDED.forks,
    watchers = EXCLUDED.watchers
RETURNING *;

-- name: UpsertProjectBySlug :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers)
ON CONFLICT (slug)
DO UPDATE SET
    url = EXCLUDED.url,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    forks = EXCLUDED.forks,
    watchers = EXCLUDED.watchers
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = @id;

-- name: GetProjectByURL :one
SELECT * FROM projects WHERE url = @url;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = @slug;

-- name: GetProjectByServiceAndRepoID :one
SELECT * FROM projects WHERE service = @service AND repo_id = @repo_id;

-- name: SearchActiveProjects :many
SELECT * FROM projects
WHERE is_active = true
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%' OR description ILIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('service')::text IS NULL OR service = sqlc.narg('service'))
  AND (sqlc.narg('cursor')::uuid IS NULL OR id < sqlc.narg('cursor')::uuid)
ORDER BY id DESC
LIMIT @limit_count;

-- name: ListActiveProjects :many
SELECT * FROM projects
WHERE is_active = true
ORDER BY updated_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: DeactivateProject :exec
UPDATE projects SET is_active = false WHERE id = @id;

-- name: ActivateProject :exec
UPDATE projects SET is_active = true WHERE id = @id;