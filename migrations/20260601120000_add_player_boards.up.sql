-- Add 3 board FK columns to players table.
-- These hold up to 3 active project boards for each player.
-- Order does not matter — a player's leaderboard score is the
-- sum of their 3 boards (computed in the leaderboard query).

ALTER TABLE players
    ADD COLUMN board_1_id UUID REFERENCES boards(id),
    ADD COLUMN board_2_id UUID REFERENCES boards(id),
    ADD COLUMN board_3_id UUID REFERENCES boards(id);
