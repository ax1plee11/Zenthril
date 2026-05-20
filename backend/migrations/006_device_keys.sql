-- Migration 006: device key management foundation for E2EE.
-- Apply with: psql $DB_URL -f backend/migrations/006_device_keys.sql

CREATE TABLE IF NOT EXISTS devices (
    id                       UUID        PRIMARY KEY,
    user_id                  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                     VARCHAR(100) NOT NULL DEFAULT '',
    identity_public_key       TEXT        NOT NULL,
    signed_pre_key_id         INT         NOT NULL,
    signed_pre_key            TEXT        NOT NULL,
    signed_pre_key_signature  TEXT        NOT NULL,
    fingerprint               TEXT        NOT NULL,
    trust_state               VARCHAR(20) NOT NULL DEFAULT 'unverified'
        CHECK (trust_state IN ('unverified', 'verified', 'revoked')),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at                TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_devices_user_id
    ON devices(user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_identity_public_key
    ON devices(identity_public_key);

CREATE TABLE IF NOT EXISTS device_one_time_prekeys (
    device_id    UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    key_id       INT         NOT NULL,
    public_key   TEXT        NOT NULL,
    consumed_at  TIMESTAMPTZ,
    consumed_by  UUID        REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, key_id)
);

CREATE INDEX IF NOT EXISTS idx_device_one_time_prekeys_available
    ON device_one_time_prekeys(device_id, key_id)
    WHERE consumed_at IS NULL;
