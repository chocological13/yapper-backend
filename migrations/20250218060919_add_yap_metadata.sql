-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;

ALTER TABLE yaps
ADD COLUMN media jsonb DEFAULT '[]'::jsonb,
ADD COLUMN hashtags text[] DEFAULT ARRAY[]::text[],
ADD COLUMN mentions text[] DEFAULT ARRAY[]::text[],
ADD COLUMN location_point geometry(Point, 4326);

CREATE INDEX yaps_hashtags_gin_idx ON yaps USING gin(hashtags);
CREATE INDEX yaps_mentions_gin_idx ON yaps USING gin(mentions);
CREATE INDEX yaps_location_idx ON yaps USING GIST(location_point);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE yaps
DROP COLUMN media,
DROP COLUMN hashtags,
DROP COLUMN mentions,
DROP COLUMN location_point;

DROP INDEX IF EXISTS yaps_hashtags_gin_idx;
DROP INDEX IF EXISTS yaps_mentions_gin_idx;
DROP INDEX IF EXISTS yaps_location_idx;

DROP EXTENSION IF EXISTS postgis;
-- +goose StatementEnd
