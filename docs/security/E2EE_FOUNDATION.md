# Zenthril E2EE Foundation

This document tracks the production path toward Signal-style E2EE. The current
implementation is a foundation for device keys and X3DH key bundles. It is not
yet a complete audited end-to-end encryption protocol.

## Current Phase

Implemented backend primitives:

- per-user device registry;
- device identity public key;
- signed prekey and signed prekey signature;
- one-time prekey upload and consumption;
- device fingerprint for safety verification UX;
- authenticated endpoint to claim an X3DH-style key bundle.

Implemented client primitives:

- local device key bundle generation;
- Ed25519 identity signing key for signed prekey verification;
- X25519 signed prekey and one-time prekeys;
- best-effort device registration after login/register;
- private device material stays client-side and is not included in backend
  registration requests.

## API Surface

Authenticated routes:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices/`
- `GET /api/v1/users/{userId}/devices`
- `POST /api/v1/key-bundles/claim`

`POST /api/v1/devices/register` stores or rotates a device key bundle for the
current user. The client should generate keys locally and send only public key
material.

`POST /api/v1/key-bundles/claim` returns a target device bundle and atomically
consumes at most one one-time prekey. This endpoint is intentionally `POST`
because claiming a bundle mutates server state.

## Next Steps

1. Add client-side secure device key storage in Tauri.
2. Add safety number display and manual verification UX.
3. Implement X3DH shared secret derivation with test vectors.
4. Add Double Ratchet session state for direct messages.
5. Add academic threat model and protocol limitations section.

## Security Notes

- The server must never receive private keys.
- One-time prekeys are consumed with row-level locking to avoid double issue.
- Device fingerprints are verification aids, not authentication by themselves.
- The current Tauri storage adapter uses the Tauri store plugin as a local
  persistence layer. Replace it with OS keychain/Stronghold before claiming
  production-grade private key storage.
- The protocol must remain marked experimental until reviewed and tested with
  reproducible vectors.
