CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id        UUID NOT NULL REFERENCES tables(id),
    restaurant_id   UUID NOT NULL REFERENCES restaurants(id),
    session_token   UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    status          TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','CLOSED')),
    payment_status  TEXT NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING', 'PARTIAL', 'PAID')),
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ
);
-- CRITICAL: Only ONE active session per table
CREATE UNIQUE INDEX idx_sessions_one_active_per_table
    ON sessions(table_id)
    WHERE status = 'ACTIVE';
CREATE INDEX idx_sessions_restaurant ON sessions(restaurant_id, status);
