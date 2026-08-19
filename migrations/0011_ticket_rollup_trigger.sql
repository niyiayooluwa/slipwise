-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rollup_ticket_status() RETURNS trigger AS $$
BEGIN
    -- If the selection changed to LOST, mark the booking code and user tickets as LOST
    IF NEW.status = 'LOST' AND OLD.status != 'LOST' THEN
        UPDATE booking_codes SET status = 'LOST' WHERE id = NEW.booking_code_id;
        UPDATE user_tickets SET status = 'LOST' WHERE booking_code_id = NEW.booking_code_id;
    END IF;

    -- If the selection changed to WON, check if ALL selections for this code are now WON
    IF NEW.status = 'WON' AND OLD.status != 'WON' THEN
        IF NOT EXISTS (
            SELECT 1 FROM booking_selections 
            WHERE booking_code_id = NEW.booking_code_id AND status != 'WON'
        ) THEN
            UPDATE booking_codes SET status = 'WON' WHERE id = NEW.booking_code_id;
            UPDATE user_tickets SET status = 'WON' WHERE booking_code_id = NEW.booking_code_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_rollup_ticket_status
AFTER UPDATE OF status ON booking_selections
FOR EACH ROW
EXECUTE FUNCTION rollup_ticket_status();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_rollup_ticket_status ON booking_selections;
DROP FUNCTION IF EXISTS rollup_ticket_status();
