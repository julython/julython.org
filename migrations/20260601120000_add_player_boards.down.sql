-- Remove 3 board FK columns from players table
ALTER TABLE players
    DROP COLUMN IF EXISTS board_1_id,
    DROP COLUMN IF EXISTS board_2_id,
    DROP COLUMN IF EXISTS board_3_id;
