ALTER TABLE sessions
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at)
    WHERE status = 'ACTIVE';
