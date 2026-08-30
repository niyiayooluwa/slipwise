-- +goose Up
ALTER TABLE matches ADD COLUMN home_score INT NOT NULL DEFAULT 0;
ALTER TABLE matches ADD COLUMN away_score INT NOT NULL DEFAULT 0;
ALTER TABLE matches ADD COLUMN live_time TEXT;

-- +goose Down
ALTER TABLE matches DROP COLUMN home_score;
ALTER TABLE matches DROP COLUMN away_score;
ALTER TABLE matches DROP COLUMN live_time;
