-- Migration 014: Double Ratchet session state storage
-- Stores pairwise E2EE session state between device pairs for Double Ratchet protocol.
-- SECURITY: Private DH keys and root/chain keys are encrypted at rest.
-- This table contains sensitive cryptographic material and must be protected.

CREATE TABLE IF NOT EXISTS device_sessions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    local_device_id      UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    remote_user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    remote_device_id     UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    
    -- Session metadata
    session_version      INT         NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at      TIMESTAMPTZ,
    
    -- Double Ratchet state (encrypted JSON)
    -- Contains: root_key, send_chain_key, recv_chain_key, counters, DH keys, skipped keys
    ratchet_state        JSONB       NOT NULL,
    
    -- Ephemeral public key from X3DH initialization
    ephemeral_public_key TEXT,
    
    -- Session lifecycle
    is_bootstrap         BOOLEAN     NOT NULL DEFAULT true,
    confirmed_at         TIMESTAMPTZ,
    
    UNIQUE (local_device_id, remote_user_id, remote_device_id),
    
    CHECK (session_version > 0)
);

-- Index for looking up sessions by local device
CREATE INDEX IF NOT EXISTS idx_device_sessions_local_device
    ON device_sessions(local_device_id, updated_at DESC);

-- Index for looking up sessions with specific remote device
CREATE INDEX IF NOT EXISTS idx_device_sessions_remote_device
    ON device_sessions(remote_device_id);

-- Index for cleanup of old/inactive sessions
CREATE INDEX IF NOT EXISTS idx_device_sessions_last_message
    ON device_sessions(last_message_at)
    WHERE last_message_at IS NOT NULL;

-- Table for storing skipped message keys separately to avoid JSONB bloat
-- and enable efficient cleanup of old skipped keys
CREATE TABLE IF NOT EXISTS device_session_skipped_keys (
    session_id           UUID        NOT NULL REFERENCES device_sessions(id) ON DELETE CASCADE,
    message_counter      BIGINT      NOT NULL,
    chain_index          INT         NOT NULL DEFAULT 0,
    
    -- Encrypted message key material
    key_data             JSONB       NOT NULL,
    
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at           TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    
    PRIMARY KEY (session_id, message_counter, chain_index),
    
    CHECK (message_counter >= 0)
);

-- Index for cleanup of expired skipped keys
CREATE INDEX IF NOT EXISTS idx_device_session_skipped_keys_expires
    ON device_session_skipped_keys(expires_at)
    WHERE expires_at < NOW() + INTERVAL '1 day';

-- Function to automatically clean up expired skipped keys
CREATE OR REPLACE FUNCTION cleanup_expired_skipped_keys()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM device_session_skipped_keys
    WHERE expires_at < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Optional: Create a scheduled job to run cleanup
-- This can be called periodically by a worker or cron job
COMMENT ON FUNCTION cleanup_expired_skipped_keys() IS 
    'Deletes expired skipped message keys. Should be called periodically (e.g., hourly).';

