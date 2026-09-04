-- E2EE: add sender DH public key to recipient envelopes for DH ratchet turns.
ALTER TABLE message_recipient_envelopes
    ADD COLUMN IF NOT EXISTS dh_public_key TEXT;

CREATE INDEX IF NOT EXISTS idx_message_recipient_envelopes_dh
    ON message_recipient_envelopes(dh_public_key);
