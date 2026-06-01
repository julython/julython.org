-- Extract owner from project URLs (https://github.com/OWNER/REPO or similar)
-- and set the owner field for all existing projects.
UPDATE projects
SET owner = SUBSTRING(url FROM '^https?://[^/]+/([^/]+)/[^/]+$')
WHERE owner = '' OR owner IS NULL;
