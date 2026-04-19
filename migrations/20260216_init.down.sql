-- Rollback complete Julython schema
-- Drops all tables, views, functions, and triggers

-- Drop triggers
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON oauth_tokens;
DROP TRIGGER IF EXISTS update_user_identifiers_updated_at ON user_identifiers;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS leaderboard;

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS language_boards;
DROP TABLE IF EXISTS languages;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS commits;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS user_identifiers;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;