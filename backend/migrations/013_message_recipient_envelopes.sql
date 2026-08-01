-- Stores opaque per-device wrapped content keys for experimental E2EE delivery.
-- SECURITY: no private device keys or plaintext content keys are stored here.
CREATE TABLE IF NOT EXISTS message_recipient_envelopes (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    ratchet_counter BIGINT NOT NULL,
    bootstrap_header JSONB,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, recipient_device_id),
    CHECK (ratchet_counter >= 0)
);

CREATE INDEX IF NOT EXISTS idx_message_recipient_envelopes_recipient
    ON message_recipient_envelopes(recipient_user_id, recipient_device_id, message_id);
