-- Speed up the leaderboard JOIN filtering on users.is_active.
-- Combined with ix_player_game_id, this lets the query
-- find active users in each game without a full table scan.
CREATE INDEX IF NOT EXISTS ix_user_is_active_id
    ON users (is_active, id);
