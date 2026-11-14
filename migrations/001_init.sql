CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    error       TEXT,
    engine_job_id TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    CONSTRAINT sessions_status_check CHECK (
        status IN ('pending', 'running', 'completed', 'failed', 'cancelled')
    )
);

CREATE INDEX IF NOT EXISTS sessions_created_at_idx ON sessions (created_at DESC);

CREATE TABLE IF NOT EXISTS leads (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    place_id      TEXT,
    name          TEXT NOT NULL DEFAULT '',
    category      TEXT,
    categories    JSONB,
    address       TEXT,
    city          TEXT,
    state         TEXT,
    country       TEXT,
    postal_code   TEXT,
    latitude      DOUBLE PRECISION,
    longitude     DOUBLE PRECISION,
    phone         TEXT,
    website       TEXT,
    maps_url      TEXT,
    rating        DOUBLE PRECISION,
    review_count  INTEGER,
    open_hours    JSONB,
    status        TEXT,
    emails        JSONB,
    extracted_at  TIMESTAMPTZ,
    raw           JSONB
);

CREATE INDEX IF NOT EXISTS leads_session_id_idx ON leads (session_id);

CREATE UNIQUE INDEX IF NOT EXISTS leads_session_place_uidx
    ON leads (session_id, place_id)
    WHERE place_id IS NOT NULL AND place_id <> '';
