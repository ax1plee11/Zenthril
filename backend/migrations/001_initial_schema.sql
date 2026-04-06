-- Миграция 001: начальная схема базы данных Veltrix
-- Применять: psql $DB_URL -f 001_initial_schema.sql

-- Расширение для генерации UUID
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Пользователи
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(32) UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,          -- Argon2id
    public_key    TEXT        NOT NULL,           -- X25519 публичный ключ, base64
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Серверы (гильдии)
-- ============================================================
CREATE TABLE IF NOT EXISTS guilds (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    owner_id   UUID         NOT NULL REFERENCES users(id),
    node_id    VARCHAR(255) NOT NULL,             -- домен/IP узла-хозяина
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Каналы
-- ============================================================
CREATE TABLE IF NOT EXISTS channels (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    guild_id   UUID         NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    type       VARCHAR(10)  NOT NULL CHECK (type IN ('text', 'voice')),
    position   INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Роли
-- ============================================================
CREATE TABLE IF NOT EXISTS roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    guild_id    UUID        NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    name        VARCHAR(50) NOT NULL,
    level       INT         NOT NULL,             -- 0=участник, 10=модератор, 50=администратор, 100=владелец
    permissions BIGINT      NOT NULL DEFAULT 0    -- битовая маска прав
);

-- ============================================================
-- Участники серверов
-- ============================================================
CREATE TABLE IF NOT EXISTS guild_members (
    guild_id    UUID        NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID        REFERENCES roles(id),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    banned      BOOLEAN     NOT NULL DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    PRIMARY KEY (guild_id, user_id)
);

-- ============================================================
-- Пригласительные ссылки
-- ============================================================
CREATE TABLE IF NOT EXISTS invites (
    code       VARCHAR(16) PRIMARY KEY,
    guild_id   UUID        NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    created_by UUID        NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    max_uses   INT,
    uses       INT         NOT NULL DEFAULT 0
);

-- ============================================================
-- Сообщения (хранятся зашифрованными)
-- ============================================================
CREATE TABLE IF NOT EXISTS messages (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL REFERENCES users(id),
    ciphertext TEXT        NOT NULL,              -- зашифрованное содержимое (base64)
    iv         TEXT        NOT NULL,              -- вектор инициализации (base64)
    key_id     TEXT        NOT NULL,              -- версия ключа шифрования
    edited     BOOLEAN     NOT NULL DEFAULT FALSE,
    deleted    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_created
    ON messages(channel_id, created_at DESC);

-- ============================================================
-- Журнал безопасности
-- ============================================================
CREATE TABLE IF NOT EXISTS security_log (
    id         BIGSERIAL   PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,              -- 'auth_fail', 'ip_blocked', 'spam_detected', ...
    ip_address INET,
    user_id    UUID        REFERENCES users(id),
    details    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_log_created
    ON security_log(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_log_ip
    ON security_log(ip_address, created_at DESC);

-- ============================================================
-- Федеративные узлы
-- ============================================================
CREATE TABLE IF NOT EXISTS federation_nodes (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    domain     VARCHAR(255) UNIQUE NOT NULL,
    public_key TEXT         NOT NULL,             -- Ed25519 публичный ключ для верификации подписей
    last_seen  TIMESTAMPTZ,
    status     VARCHAR(20)  NOT NULL DEFAULT 'active'
);
