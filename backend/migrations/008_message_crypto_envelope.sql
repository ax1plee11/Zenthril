ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS tag TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS protocol_version INTEGER NOT NULL DEFAULT 1;

COMMENT ON COLUMN messages.tag IS 'AES-GCM authentication tag, base64 encoded. Empty for legacy alpha messages created before envelope v1.';
COMMENT ON COLUMN messages.protocol_version IS 'Client crypto envelope protocol version. Version 1 uses HKDF-derived AES-GCM keys with AAD.';
