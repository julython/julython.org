ALTER TABLE users ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user';
ALTER TABLE users ALTER COLUMN is_active SET DEFAULT true;
ALTER TABLE users ALTER COLUMN is_banned SET DEFAULT false;

ALTER TABLE user_identifiers ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE user_identifiers ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE user_identifiers ALTER COLUMN verified SET DEFAULT false;
ALTER TABLE user_identifiers ALTER COLUMN is_primary SET DEFAULT false;

ALTER TABLE games ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE games ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE games ALTER COLUMN commit_points SET DEFAULT 1;
ALTER TABLE games ALTER COLUMN project_points SET DEFAULT 10;
ALTER TABLE games ALTER COLUMN is_active SET DEFAULT false;

ALTER TABLE projects ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE projects ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE projects ALTER COLUMN service SET DEFAULT 'github';
ALTER TABLE projects ALTER COLUMN forked SET DEFAULT false;
ALTER TABLE projects ALTER COLUMN forks SET DEFAULT 0;
ALTER TABLE projects ALTER COLUMN watchers SET DEFAULT 0;
ALTER TABLE projects ALTER COLUMN is_active SET DEFAULT true;

ALTER TABLE commits ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE commits ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE commits ALTER COLUMN is_verified SET DEFAULT false;
ALTER TABLE commits ALTER COLUMN is_flagged SET DEFAULT false;

ALTER TABLE players ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE players ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE players ALTER COLUMN points SET DEFAULT 0;
ALTER TABLE players ALTER COLUMN potential_points SET DEFAULT 0;
ALTER TABLE players ALTER COLUMN verified_points SET DEFAULT 0;
ALTER TABLE players ALTER COLUMN commit_count SET DEFAULT 0;
ALTER TABLE players ALTER COLUMN project_count SET DEFAULT 0;
ALTER TABLE players ALTER COLUMN analysis_status SET DEFAULT 'pending';

ALTER TABLE boards ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE boards ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE boards ALTER COLUMN points SET DEFAULT 0;
ALTER TABLE boards ALTER COLUMN potential_points SET DEFAULT 0;
ALTER TABLE boards ALTER COLUMN verified_points SET DEFAULT 0;
ALTER TABLE boards ALTER COLUMN commit_count SET DEFAULT 0;
ALTER TABLE boards ALTER COLUMN contributor_count SET DEFAULT 0;

ALTER TABLE languages ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE languages ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE language_boards ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE language_boards ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE language_boards ALTER COLUMN points SET DEFAULT 0;
ALTER TABLE language_boards ALTER COLUMN commit_count SET DEFAULT 0;

ALTER TABLE teams ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE teams ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE teams ALTER COLUMN is_public SET DEFAULT true;
ALTER TABLE teams ALTER COLUMN member_count SET DEFAULT 0;

ALTER TABLE team_members ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE team_members ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE team_members ALTER COLUMN role SET DEFAULT 'member';

ALTER TABLE team_boards ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE team_boards ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE team_boards ALTER COLUMN points SET DEFAULT 0;
ALTER TABLE team_boards ALTER COLUMN member_count SET DEFAULT 0;

ALTER TABLE reports ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE reports ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE reports ALTER COLUMN report_type SET DEFAULT 'spam';
ALTER TABLE reports ALTER COLUMN status SET DEFAULT 'pending';

ALTER TABLE audit_logs ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE audit_logs ALTER COLUMN updated_at SET DEFAULT now();
