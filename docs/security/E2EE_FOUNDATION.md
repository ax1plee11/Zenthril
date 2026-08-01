# Zenthril E2EE Foundation

Languages: [Русский](E2EE_FOUNDATION.ru.md) | [English](E2EE_FOUNDATION.md) | [Українська](E2EE_FOUNDATION.uk.md)

This document tracks the production path toward Signal-style E2EE. The current
implementation is a foundation for device keys and X3DH key bundles. It is not
yet a complete audited end-to-end encryption protocol.

## Current Phase

Implemented backend primitives:

- per-user device registry;
- separate Ed25519 identity signing key and X25519 identity-DH public key;
- signed prekey and signed prekey signature;
- one-time prekey upload and consumption;
- device fingerprint for safety verification UX;
- authenticated endpoint to claim an X3DH-style key bundle;
- initial Double Ratchet state model with HKDF-SHA256 root, chain, and
  message-key derivation boundaries.

Implemented client primitives:

- local device key bundle generation;
- Ed25519 identity signing key for signed prekey verification and a separate
  X25519 identity-DH key for X3DH;
- X25519 signed prekey and one-time prekeys;
- best-effort device registration after login/register;
- device management UI for active devices and remote device revocation;
- deterministic safety number generation for pairwise device verification;
- HKDF-SHA256 derivation before importing X25519 shared secrets as AES-GCM keys;
- protocol-v1 message envelope with AES-GCM authentication tag persistence;
- protocol-v2 message envelope with AAD context binding channel, sender user,
  sender device, session, client message id, key id, and cipher suite;
- backend protocol-v2 validation now requires the full AAD context, including
  `channel_id` and `sender_user_id`, so new ciphertext cannot omit route/user
  binding fields;
- AES-GCM associated data binding `protocol_version` and `key_id` for legacy
  v1 payloads;
- private device material stays client-side and is not included in backend
  registration requests.
- a client pairwise X3DH bootstrap that verifies the remote signed prekey,
  derives a root plus directional chain keys with HKDF-SHA256, and consumes the
  selected one-time prekey on acceptance.

## API Surface

Authenticated routes:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices/`
- `GET /api/v1/users/{userId}/devices`
- `DELETE /api/v1/devices/{deviceId}`
- `POST /api/v1/key-bundles/claim`

`POST /api/v1/devices/register` stores or rotates a device key bundle for the
current user. The client should generate keys locally and send only public key
material.

`POST /api/v1/key-bundles/claim` returns a target device bundle and atomically
consumes at most one one-time prekey. This endpoint is intentionally `POST`
because claiming a bundle mutates server state.

`DELETE /api/v1/devices/{deviceId}` revokes one of the current user's devices
and removes its one-time prekeys so new sessions cannot be established with it.

## Next Steps

1. Connect the verified X3DH bootstrap to recipient-device message envelopes.
2. Persist pairwise ratchet state through OS-backed storage for direct messages.
3. Add skipped-message-key handling, DH ratchet turns, and audited X25519 test vectors.
4. Design a reviewed sender-key or MLS-based protocol for group channels.
5. Add product threat model and protocol limitations section.

## Security Notes

- The server must never receive private keys.
- One-time prekeys are consumed with row-level locking to avoid double issue.
- Device fingerprints are verification aids, not authentication by themselves.
- Tauri desktop now prefers OS-backed storage via Rust `keyring`; the previous
  store-plugin data is an alpha migration fallback only. This is still not a
  substitute for secure recovery, device verification, or external audit.
- Safety numbers are local verification UX primitives. They do not replace
  signature verification, trust decisions, or future per-user trust records.
- The client pairwise session foundation is not yet a complete message protocol:
  it has no skipped message keys, header encryption, full DH ratchet turns, or
  production recipient-device envelope distribution.
- Alpha messages created before the protocol-v1 envelope may not decrypt from
  history because the backend did not persist the AES-GCM authentication tag.
- Protocol-v1 payloads remain accepted as a temporary alpha compatibility path;
  the client `encrypt()` API now requires protocol-v2 AAD context for new sends.
- The protocol must remain marked experimental until reviewed and tested with
  reproducible vectors.
