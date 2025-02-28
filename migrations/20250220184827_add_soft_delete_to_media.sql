-- +goose Up
-- +goose StatementBegin
ALTER TABLE media ADD COLUMN deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS media_deleted_at_idx ON public.media(deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media DROP COLUMN deleted_at;
-- +goose StatementEnd
