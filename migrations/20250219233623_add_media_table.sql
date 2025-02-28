-- +goose Up
-- +goose StatementBegin
-- Create media table
CREATE TABLE IF NOT EXISTS public.media (
    media_id UUID DEFAULT uuid_generate_v1mc() NOT NULL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('image', 'video')),
    url TEXT NOT NULL,
    content_id UUID,
    content_type TEXT CHECK (content_type IN ('yap', 'reply')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS media_content_idx ON public.media(content_id, content_type);
CREATE INDEX IF NOT EXISTS media_cleanup_idx ON public.media(created_at) WHERE content_id IS NULL;

-- Modify yaps table
ALTER TABLE public.yaps DROP COLUMN media;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE public.yaps ADD COLUMN media JSONB DEFAULT '[]'::jsonb;
DROP TABLE IF EXISTS public.media;
-- +goose StatementEnd
