-- Миграция 002: глобальные баны пользователей
-- Применять: psql $DB_URL -f 002_global_bans.sql

CREATE TABLE IF NOT EXISTS global_bans (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by  UUID        REFERENCES users(id),
    reason     TEXT,
    expires_at TIMESTAMPTZ,                       -- NULL = перманентный бан
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_global_bans_user ON global_bans(user_id);
