-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN deleted_at timestamptz;

-- Update existing queries to exclude deleted records
CREATE OR REPLACE VIEW active_users AS
SELECT *
FROM users
WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS active_users;
ALTER TABLE users
DROP COLUMN deleted_at;
-- +goose StatementEnd
