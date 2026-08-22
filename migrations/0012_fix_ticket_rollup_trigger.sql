-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rollup_ticket_status() RETURNS trigger AS $$
BEGIN
    -- If the selection changed to LOST, mark the booking code as LOST
    IF NEW.status = 'LOST' AND OLD.status != 'LOST' THEN
        UPDATE booking_codes SET status = 'LOST' WHERE id = NEW.booking_code_id;
    END IF;

    -- If the selection changed to WON, check if ALL selections for this code are now WON
    IF NEW.status = 'WON' AND OLD.status != 'WON' THEN
        IF NOT EXISTS (
            SELECT 1 FROM booking_selections 
            WHERE booking_code_id = NEW.booking_code_id AND status != 'WON'
        ) THEN
            UPDATE booking_codes SET status = 'WON' WHERE id = NEW.booking_code_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rollup_ticket_status() RETURNS trigger AS $$
BEGIN
    IF NEW.status = 'LOST' AND OLD.status != 'LOST' THEN
        UPDATE booking_codes SET status = 'LOST' WHERE id = NEW.booking_code_id;
    END IF;

    IF NEW.status = 'WON' AND OLD.status != 'WON' THEN
        IF NOT EXISTS (
            SELECT 1 FROM booking_selections 
            WHERE booking_code_id = NEW.booking_code_id AND status != 'WON'
        ) THEN
            UPDATE booking_codes SET status = 'WON' WHERE id = NEW.booking_code_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
