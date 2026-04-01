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
LIMIT GREATEST(@limit_count, 1);

-- name: ListActiveProjects :many
SELECT * FROM projects
WHERE is_active = true
ORDER BY id DESC
LIMIT GREATEST(@limit_count, 1);

-- name: DeactivateProject :exec
UPDATE projects SET is_active = false WHERE id = @id;

-- name: ActivateProject :exec
UPDATE projects SET is_active = true WHERE id = @id;

-- name: UpsertAnalysisMetric :exec
-- Score always reflects latest scan. Level auto-transitions between 0 and 1
-- based on whether the score hits the threshold (10). L2/L3 are never
-- downgraded by a rescan — the AI grading endpoint owns those transitions.
INSERT INTO analysis_metrics (id, project_id, metric_type, level, score, data, sha, updated_by)
VALUES (@id, @project_id, @metric_type, CASE WHEN @score = 10 THEN 1 ELSE 0 END, @score, @data, @sha, @updated_by)
ON CONFLICT (project_id, metric_type) DO UPDATE SET
  sha        = EXCLUDED.sha,
  updated_at = now(),
  updated_by = EXCLUDED.updated_by,
  data       = EXCLUDED.data,
  score      = EXCLUDED.score,
  level      = CASE
    WHEN analysis_metrics.level >= 2 THEN analysis_metrics.level  -- preserve L2/L3
    WHEN EXCLUDED.score = 10         THEN 1                       -- promote to L1
    ELSE                                  0                       -- incomplete
  END;

-- name: UpdateAnalysisMetricLevel :exec
-- Called after AI grading for L2/L3 upgrades only.
-- Requires the metric to already be at L1 (enforced in the handler).
UPDATE analysis_metrics SET
  level      = @level,
  updated_at = now(),
  updated_by = @updated_by
WHERE project_id = @project_id
  AND metric_type = @metric_type;

-- name: GetAnalysisMetric :one
SELECT * FROM analysis_metrics
WHERE project_id = @project_id
  AND metric_type = @metric_type;

-- name: GetAnalysisMetricsByProject :many
SELECT * FROM analysis_metrics
WHERE project_id = @project_id
ORDER BY metric_type;

-- name: GetProjectTotalScore :one
SELECT COALESCE(SUM(score * level), 0)::int AS total_score
FROM analysis_metrics
WHERE project_id = @project_id;