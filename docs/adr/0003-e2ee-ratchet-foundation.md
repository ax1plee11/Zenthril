# ADR 0003: Build E2EE around a Double Ratchet foundation

## Status

Accepted

## Context

Zenthril already contains foundational cryptographic primitives and device management, but the project is not yet a complete Signal-grade E2EE system. To reach a stronger security model, the messaging protocol needs forward secrecy, post-compromise recovery properties, device-aware sessions, and clear testability.

## Decision

Use a Double Ratchet-style foundation for message key evolution and plan X3DH-style session setup with device pre-key bundles.

The current implementation starts with:

- Root key evolution with HKDF.
- Chain key derivation for sending and receiving chains.
- Per-message key derivation.
- State export/import for future persistence.
- Tests for key progression and skipped-message behavior.

Future protocol integration should add:

- X3DH-style identity keys, signed pre-keys, and one-time pre-keys.
- Authenticated device bundles.
- Safety number verification.
- Persistent session state per remote device.
- Replay protection and skipped message key limits.

## Rationale

Double Ratchet is a well-understood design pattern for asynchronous secure messaging. It gives the project a credible path toward forward secrecy and recovery after key compromise, while still allowing incremental implementation and academic evaluation.

## Consequences

Positive:

- E2EE work has a clear direction.
- Ratchet behavior can be benchmarked and tested independently.
- Documentation can honestly separate primitives, foundations, and complete protocol behavior.

Tradeoffs:

- The system must manage more client-side state.
- Multi-device support becomes more complex.
- A complete implementation requires careful review and interoperability-style test vectors.

## Follow-up

- Integrate ratchet sessions into client message flows.
- Add pre-key bundle APIs and storage.
- Add property-based tests and documented test vectors.
- Document metadata that remains visible to the server.
