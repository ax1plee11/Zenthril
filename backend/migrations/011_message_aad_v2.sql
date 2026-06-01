ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS sender_device_id TEXT,
  ADD COLUMN IF NOT EXISTS session_id TEXT,
  ADD COLUMN IF NOT EXISTS client_message_id TEXT,
  ADD COLUMN IF NOT EXISTS cipher_suite TEXT;

COMMENT ON COLUMN messages.sender_device_id IS 'Client E2EE AAD v2 sender device id.';
COMMENT ON COLUMN messages.session_id IS 'Client E2EE AAD v2 session id.';
COMMENT ON COLUMN messages.client_message_id IS 'Client generated message id bound into AAD v2.';
COMMENT ON COLUMN messages.cipher_suite IS 'Client E2EE cipher suite identifier for AAD v2.';

