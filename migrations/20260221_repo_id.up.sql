-- Expand projects.repo_id from INTEGER to BIGINT
-- GitHub repo IDs have exceeded the 32-bit integer range

ALTER TABLE projects ALTER COLUMN repo_id TYPE BIGINT;