-- +goose Up
-- Add gamification metrics to user_stats
ALTER TABLE user_stats 
    ADD COLUMN current_win_streak INT NOT NULL DEFAULT 0,
    ADD COLUMN longest_win_streak INT NOT NULL DEFAULT 0,
    ADD COLUMN total_points NUMERIC NOT NULL DEFAULT 0;

-- Add notification state tracking to booking_selections
ALTER TABLE booking_selections 
    ADD COLUMN notified_early_win BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN notified_ht BOOLEAN NOT NULL DEFAULT FALSE;

-- Update the existing settlement trigger to handle streaks and points
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trigger_settle_ticket_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status = 'PENDING' AND NEW.status = 'WON' THEN
        UPDATE user_stats us
        SET pending_tickets = us.pending_tickets - 1,
            won_tickets = us.won_tickets + 1,
            total_returns = us.total_returns + (ut.stake * NEW.total_odds),
            current_win_streak = us.current_win_streak + 1,
            longest_win_streak = GREATEST(us.longest_win_streak, us.current_win_streak + 1),
            total_points = us.total_points + 10
        FROM user_tickets ut
        WHERE ut.booking_code_id = NEW.id AND us.user_id = ut.user_id;
    ELSIF OLD.status = 'PENDING' AND NEW.status = 'LOST' THEN
        UPDATE user_stats us
        SET pending_tickets = us.pending_tickets - 1,
            lost_tickets = us.lost_tickets + 1,
            current_win_streak = 0
        FROM user_tickets ut
        WHERE ut.booking_code_id = NEW.id AND us.user_id = ut.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE user_stats 
    DROP COLUMN current_win_streak,
    DROP COLUMN longest_win_streak,
    DROP COLUMN total_points;

ALTER TABLE booking_selections 
    DROP COLUMN notified_early_win,
    DROP COLUMN notified_ht;
