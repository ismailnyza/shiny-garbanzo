CREATE TABLE themes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id   UUID NOT NULL UNIQUE REFERENCES restaurants(id) ON DELETE CASCADE,
    primary_color   TEXT NOT NULL DEFAULT '#FF6B35',
    secondary_color TEXT NOT NULL DEFAULT '#F7C59F',
    logo_url        TEXT,
    banner_url      TEXT,
    font_style      TEXT NOT NULL DEFAULT 'inter',
    dark_mode       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
