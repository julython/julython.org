-- ============================================
-- Projects
-- ============================================

-- name: CreateProject :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers, parent_url, is_private)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers, @parent_url, @is_private)
RETURNING *;

-- name: UpsertProjectByRepoID :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers, is_private)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers, @is_private)
ON CONFLICT (service, repo_id) WHERE repo_id IS NOT NULL
DO UPDATE SET
    url = EXCLUDED.url,
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    description = EXCLUDED.description,
    forks = EXCLUDED.forks,
    watchers = EXCLUDED.watchers,
    is_private = EXCLUDED.is_private
RETURNING *;

-- name: UpsertProjectBySlug :one
INSERT INTO projects (id, url, name, slug, description, repo_id, service, forked, forks, watchers, is_private)
VALUES (@id, @url, @name, @slug, @description, @repo_id, @service, @forked, @forks, @watchers, @is_private)
ON CONFLICT (slug)
DO UPDATE SET
    url = EXCLUDED.url,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    forks = EXCLUDED.forks,
    watchers = EXCLUDED.watchers,
    is_private = EXCLUDED.is_private
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
  AND is_private = false
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%' OR description ILIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('service')::text IS NULL OR service = sqlc.narg('service'))
  AND (sqlc.narg('cursor')::uuid IS NULL OR id < sqlc.narg('cursor')::uuid)
ORDER BY id DESC
LIMIT GREATEST(@limit_count, 1);

-- name: ListActiveProjects :many
SELECT * FROM projects
WHERE is_active = true
  AND is_private = false
ORDER BY id DESC
LIMIT GREATEST(@limit_count, 1);

-- name: SetProjectIsPrivate :exec
UPDATE projects SET is_private = @is_private WHERE id = @id;

-- name: DeactivateProject :exec
UPDATE projects SET is_active = false WHERE id = @id;

-- name: ActivateProject :exec
UPDATE projects SET is_active = true WHERE id = @id;

-- name: UpsertAnalysisMetric :exec
-- Score always reflects latest scan. Level 1 (heuristic partial) when score > 0.
-- L2/L3 AI levels are never downgraded by a rescan — UpdateAnalysisMetricLevel owns AI tiers.
INSERT INTO analysis_metrics (id, project_id, metric_type, level, score, data, sha, updated_by)
VALUES (@id, @project_id, @metric_type, CASE WHEN @score > 0 THEN 1 ELSE 0 END, @score, @data, @sha, @updated_by)
ON CONFLICT (project_id, metric_type) DO UPDATE SET
  sha        = EXCLUDED.sha,
  updated_at = now(),
  updated_by = EXCLUDED.updated_by,
  data       = EXCLUDED.data,
  score      = EXCLUDED.score,
  level      = CASE
    WHEN analysis_metrics.level >= 2 THEN analysis_metrics.level  -- preserve L2/L3 AI
    WHEN EXCLUDED.score > 0          THEN 1                       -- L1 heuristic partial
    ELSE                                  0                       -- score 0, no L1 bar
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