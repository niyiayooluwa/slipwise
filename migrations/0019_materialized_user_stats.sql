-- +goose Up
CREATE TABLE user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_tickets INT NOT NULL DEFAULT 0,
    won_tickets INT NOT NULL DEFAULT 0,
    lost_tickets INT NOT NULL DEFAULT 0,
    pending_tickets INT NOT NULL DEFAULT 0,
    total_staked NUMERIC NOT NULL DEFAULT 0,
    total_returns NUMERIC NOT NULL DEFAULT 0
);

-- Trigger 1: When a user tracks a new ticket
CREATE OR REPLACE FUNCTION trigger_increment_new_ticket()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, total_tickets, pending_tickets, total_staked)
    VALUES (NEW.user_id, 1, 1, NEW.stake)
    ON CONFLICT (user_id) DO UPDATE SET 
        total_tickets = user_stats.total_tickets + 1,
        pending_tickets = user_stats.pending_tickets + 1,
        total_staked = user_stats.total_staked + NEW.stake;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_ticket_tracked
AFTER INSERT ON user_tickets
FOR EACH ROW EXECUTE FUNCTION trigger_increment_new_ticket();

-- Trigger 2: When the background worker settles a booking code
CREATE OR REPLACE FUNCTION trigger_settle_ticket_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status = 'PENDING' AND NEW.status = 'WON' THEN
        UPDATE user_stats us
        SET pending_tickets = us.pending_tickets - 1,
            won_tickets = us.won_tickets + 1,
            total_returns = us.total_returns + (ut.stake * NEW.total_odds)
        FROM user_tickets ut
        WHERE ut.booking_code_id = NEW.id AND us.user_id = ut.user_id;
    ELSIF OLD.status = 'PENDING' AND NEW.status = 'LOST' THEN
        UPDATE user_stats us
        SET pending_tickets = us.pending_tickets - 1,
            lost_tickets = us.lost_tickets + 1
        FROM user_tickets ut
        WHERE ut.booking_code_id = NEW.id AND us.user_id = ut.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_booking_code_settled
AFTER UPDATE OF status ON booking_codes
FOR EACH ROW EXECUTE FUNCTION trigger_settle_ticket_stats();

-- +goose Down
DROP TRIGGER IF EXISTS on_booking_code_settled ON booking_codes;
DROP FUNCTION IF EXISTS trigger_settle_ticket_stats;
DROP TRIGGER IF EXISTS on_ticket_tracked ON user_tickets;
DROP FUNCTION IF EXISTS trigger_increment_new_ticket;
DROP TABLE IF EXISTS user_stats;
