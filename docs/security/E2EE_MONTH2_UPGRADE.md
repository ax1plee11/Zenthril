# E2EE Month 2 Upgrade Plan

Zenthril is still alpha software. This document defines the next security
milestones without claiming Signal-grade security before the protocol is fully
implemented, tested, and audited.

## Implemented In This Batch

- Safety number QR payload format for device identity verification.
- Existing safety number generation remains deterministic across participant order.
- Existing protocol-v1 message envelope continues using HKDF-derived AES-GCM keys and AAD.
- Existing symmetric ratchet helper remains a foundation, not a complete Double Ratchet session.

## X3DH Work Required

- Persist identity key, signed prekey, signed prekey signature, and one-time prekeys per device.
- Verify signed prekey signatures before session bootstrap.
- Consume one-time prekeys atomically on the server.
- Derive the initial root key with X3DH DH1/DH2/DH3/DH4 and HKDF domain separation.
- Bind session metadata into associated data: protocol version, local device, remote device, and prekey IDs.
- Add test vectors for successful setup, missing one-time prekey fallback, and invalid signature rejection.

## Double Ratchet Work Required

- Store per-peer-device ratchet state locally.
- Advance sending chain for every outbound message.
- Advance receiving chain for every inbound message.
- Store skipped message keys with strict limits and expiry.
- Delete message keys immediately after use.
- Rotate DH ratchet keys when a new remote ratchet public key is observed.
- Add post-compromise recovery tests and out-of-order message tests.

## Key Storage Work Required

- Replace Tauri Store for private E2EE material with OS keychain or Tauri Stronghold.
- Keep production web builds fail-closed when secure key storage is unavailable.
- Add explicit migration UX for existing alpha keys.
- Document that old alpha key material may need re-registration.

## Verification UX

- Show safety numbers in groups for manual comparison.
- Encode safety number payloads into QR codes in the UI.
- Require explicit user confirmation before marking a device verified.
- Warn when a known device identity key changes.

## Non-Goals For This Batch

- No claim of full Signal Protocol compatibility.
- No external audit claim.
- No silent trust on first use promotion to verified state.
