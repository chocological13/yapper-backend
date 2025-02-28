-- +goose Up
-- +goose StatementBegin
CREATE TABLE yaps(
  yap_id uuid PRIMARY KEY DEFAULT uuid_generate_v1mc(),
  user_id uuid NOT NULL REFERENCES users(user_id),
  content text NOT NULL CHECK (LENGTH(content) <= 140),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz,
  deleted_at timestamptz
);

CREATE INDEX yaps_user_id_idx ON yaps(user_id);
CREATE INDEX yaps_created_at_idx ON yaps(created_at);

SELECT trigger_updated_at('yaps');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS yaps;
-- +goose StatementEnd
