-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
-- +goose StatementEnd
