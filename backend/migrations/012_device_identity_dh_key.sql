-- Migration 012: split the Ed25519 signing identity from the X25519 X3DH identity.
-- Existing alpha device bundles have no X25519 identity key and must be
-- re-registered by an updated client before they may participate in X3DH.

ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS identity_dh_public_key TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_identity_dh_public_key
  ON devices(identity_dh_public_key)
  WHERE identity_dh_public_key <> '';

COMMENT ON COLUMN devices.identity_public_key IS
  'Ed25519 public signing identity used to verify signed prekeys and safety numbers.';
COMMENT ON COLUMN devices.identity_dh_public_key IS
  'X25519 public identity used only for X3DH DH operations. Empty marks an obsolete alpha device bundle.';
