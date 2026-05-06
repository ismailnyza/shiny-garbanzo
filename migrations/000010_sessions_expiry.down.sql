DROP INDEX IF EXISTS idx_sessions_expires_at;
ALTER TABLE sessions
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS expires_at;
