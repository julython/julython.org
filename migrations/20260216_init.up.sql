-- Complete Julython Database Schema
-- Aligned with Python SQLModel definitions

-- ============================================
-- Core Tables
-- ============================================

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    username      VARCHAR(25) NOT NULL,
    avatar_url    VARCHAR,
    role          VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    is_banned     BOOLEAN NOT NULL DEFAULT false,
    banned_reason VARCHAR,
    banned_at     TIMESTAMPTZ,
    last_seen     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS "unique-user-username" ON users(username);
CREATE INDEX IF NOT EXISTS ix_user_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS ix_user_created_at ON users(created_at DESC);

CREATE TABLE IF NOT EXISTS user_identifiers (
    value      VARCHAR PRIMARY KEY,
    type       VARCHAR(10) NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    verified   BOOLEAN NOT NULL DEFAULT false,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    data       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_useridentifier_user_id ON user_identifiers(user_id);
CREATE INDEX IF NOT EXISTS ix_useridentifier_type ON user_identifiers(type);

-- ============================================
-- Games System
-- ============================================

CREATE TABLE IF NOT EXISTS games (
    id             UUID PRIMARY KEY,
    name           VARCHAR(25) NOT NULL,
    starts_at      TIMESTAMPTZ NOT NULL,
    ends_at        TIMESTAMPTZ NOT NULL,
    commit_points  INT NOT NULL DEFAULT 1,
    project_points INT NOT NULL DEFAULT 10,
    is_active      BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_game_active ON games(is_active, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS ix_game_dates ON games(starts_at, ends_at);

CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY,
    url         VARCHAR NOT NULL,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR NOT NULL,
    description VARCHAR,
    repo_id     INTEGER,
    service     VARCHAR(20) NOT NULL DEFAULT 'github',
    forked      BOOLEAN NOT NULL DEFAULT false,
    forks       INT NOT NULL DEFAULT 0,
    watchers    INT NOT NULL DEFAULT 0,
    parent_url  VARCHAR,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_project_service_repo ON projects(service, repo_id);
CREATE UNIQUE INDEX IF NOT EXISTS ix_project_url ON projects(url);
CREATE UNIQUE INDEX IF NOT EXISTS ix_project_slug ON projects(slug);
CREATE INDEX IF NOT EXISTS ix_project_repo_id ON projects(repo_id);
CREATE INDEX IF NOT EXISTS ix_project_is_active ON projects(is_active);

CREATE TABLE IF NOT EXISTS commits (
    id          UUID PRIMARY KEY,
    hash        VARCHAR(255) UNIQUE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    game_id     UUID REFERENCES games(id),
    author      VARCHAR(100),
    email       VARCHAR(320),
    message     VARCHAR NOT NULL,
    url         VARCHAR NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    languages   VARCHAR[],
    files       JSONB,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_flagged  BOOLEAN NOT NULL DEFAULT false,
    flag_reason VARCHAR,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ix_commit_hash ON commits(hash);
CREATE INDEX IF NOT EXISTS ix_commit_project_id ON commits(project_id);
CREATE INDEX IF NOT EXISTS ix_commit_user_id ON commits(user_id);
CREATE INDEX IF NOT EXISTS ix_commit_game_id ON commits(game_id);
CREATE INDEX IF NOT EXISTS ix_commit_timestamp ON commits(timestamp DESC);
CREATE INDEX IF NOT EXISTS ix_commit_game_timestamp ON commits(game_id, timestamp) WHERE game_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_commit_languages ON commits USING GIN(languages) WHERE languages IS NOT NULL;

CREATE TABLE IF NOT EXISTS players (
    id               UUID PRIMARY KEY,
    game_id          UUID NOT NULL REFERENCES games(id),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    points           INT NOT NULL DEFAULT 0,
    potential_points INT NOT NULL DEFAULT 0,
    verified_points  INT NOT NULL DEFAULT 0,
    commit_count     INT NOT NULL DEFAULT 0,
    project_count    INT NOT NULL DEFAULT 0,
    analysis_status  VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_analyzed_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_player_user_game UNIQUE (game_id, user_id)
);

CREATE INDEX IF NOT EXISTS ix_player_game_id ON players(game_id);
CREATE INDEX IF NOT EXISTS ix_player_user_id ON players(user_id);
CREATE INDEX IF NOT EXISTS ix_player_game_points ON players(game_id, points DESC);

CREATE TABLE IF NOT EXISTS boards (
    id                UUID PRIMARY KEY,
    game_id           UUID NOT NULL REFERENCES games(id),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    points            INT NOT NULL DEFAULT 0,
    potential_points  INT NOT NULL DEFAULT 0,
    verified_points   INT NOT NULL DEFAULT 0,
    commit_count      INT NOT NULL DEFAULT 0,
    contributor_count INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_board_project_game UNIQUE (game_id, project_id)
);

CREATE INDEX IF NOT EXISTS ix_board_game_id ON boards(game_id);
CREATE INDEX IF NOT EXISTS ix_board_project_id ON boards(project_id);
CREATE INDEX IF NOT EXISTS ix_board_game_points ON boards(game_id, points DESC);

CREATE TABLE IF NOT EXISTS languages (
    id         UUID PRIMARY KEY,
    name       VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ix_language_name ON languages(name);

CREATE TABLE IF NOT EXISTS language_boards (
    id           UUID PRIMARY KEY,
    game_id      UUID NOT NULL REFERENCES games(id),
    language_id  UUID NOT NULL REFERENCES languages(id),
    points       INT NOT NULL DEFAULT 0,
    commit_count INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_language_game UNIQUE (game_id, language_id)
);

CREATE INDEX IF NOT EXISTS ix_languageboard_game_id ON language_boards(game_id);
CREATE INDEX IF NOT EXISTS ix_languageboard_language_id ON language_boards(language_id);
CREATE INDEX IF NOT EXISTS ix_languageboard_game_points ON language_boards(game_id, points DESC);

-- ============================================
-- Teams
-- ============================================

CREATE TABLE IF NOT EXISTS teams (
    id           UUID PRIMARY KEY,
    name         VARCHAR NOT NULL,
    slug         VARCHAR NOT NULL,
    description  VARCHAR,
    avatar_url   VARCHAR,
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_public    BOOLEAN NOT NULL DEFAULT true,
    member_count INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ix_team_name ON teams(name);
CREATE UNIQUE INDEX IF NOT EXISTS ix_team_slug ON teams(slug);
CREATE INDEX IF NOT EXISTS ix_team_created_by ON teams(created_by);

CREATE TABLE IF NOT EXISTS team_members (
    id         UUID PRIMARY KEY,
    team_id    UUID NOT NULL REFERENCES teams(id),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_teammember_team_id ON team_members(team_id);
CREATE INDEX IF NOT EXISTS ix_teammember_user_id ON team_members(user_id);

CREATE TABLE IF NOT EXISTS team_boards (
    id           UUID PRIMARY KEY,
    game_id      UUID NOT NULL REFERENCES games(id),
    team_id      UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    points       INT NOT NULL DEFAULT 0,
    member_count INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_teamboard_team_game UNIQUE (game_id, team_id)
);

CREATE INDEX IF NOT EXISTS ix_teamboard_game_id ON team_boards(game_id);
CREATE INDEX IF NOT EXISTS ix_teamboard_team_id ON team_boards(team_id);
CREATE INDEX IF NOT EXISTS ix_teamboard_game_points ON team_boards(game_id, points DESC);

-- ============================================
-- Moderation
-- ============================================

CREATE TABLE IF NOT EXISTS reports (
    id               UUID PRIMARY KEY,
    reported_user_id UUID REFERENCES users(id),
    report_type      VARCHAR(20) NOT NULL DEFAULT 'spam',
    reason           VARCHAR(100),
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by      UUID REFERENCES users(id),
    reviewed_at      TIMESTAMPTZ,
    moderator_notes  VARCHAR,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_report_reported_user_id ON reports(reported_user_id);
CREATE INDEX IF NOT EXISTS ix_report_status ON reports(status) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS audit_logs (
    id           UUID PRIMARY KEY,
    moderator_id UUID NOT NULL REFERENCES users(id),
    action       VARCHAR NOT NULL,
    target_type  VARCHAR NOT NULL,
    target_id    VARCHAR NOT NULL,
    reason       VARCHAR,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_auditlog_moderator_id ON audit_logs(moderator_id);
CREATE INDEX IF NOT EXISTS ix_auditlog_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS ix_auditlog_target ON audit_logs(target_type, target_id);

-- ============================================
-- Functions and Triggers
-- ============================================

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_user_identifiers_updated_at BEFORE UPDATE ON user_identifiers
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_games_updated_at BEFORE UPDATE ON games
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_projects_updated_at BEFORE UPDATE ON projects
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_commits_updated_at BEFORE UPDATE ON commits
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_players_updated_at BEFORE UPDATE ON players
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_boards_updated_at BEFORE UPDATE ON boards
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_languages_updated_at BEFORE UPDATE ON languages
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_language_boards_updated_at BEFORE UPDATE ON language_boards
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_teams_updated_at BEFORE UPDATE ON teams
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_team_members_updated_at BEFORE UPDATE ON team_members
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_team_boards_updated_at BEFORE UPDATE ON team_boards
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_reports_updated_at BEFORE UPDATE ON reports
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_audit_logs_updated_at BEFORE UPDATE ON audit_logs
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;