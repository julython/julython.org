-- Revert projects.repo_id from BIGINT back to INTEGER
-- NOTE: Will fail if any existing repo_id values exceed 2,147,483,647

ALTER TABLE projects ALTER COLUMN repo_id TYPE INTEGER;