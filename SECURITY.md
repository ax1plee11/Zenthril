# Security Policy

## Documentation Source of Truth

Current security documentation is versioned in this repository:
[`SECURITY.md`](SECURITY.md), [`THREAT_MODEL.md`](THREAT_MODEL.md), and
[`docs/`](docs/). Do not rely on a separate GitHub Wiki for current security
state, because Wiki content can drift from the code.

## Supported Versions

Zenthril is currently an alpha-stage project. There are no production-supported
stable releases yet.

| Version | Security support |
| --- | --- |
| `main` / `0.1.x-alpha` | Best-effort fixes during active development |
| Older commits | Not supported |

## Reporting a Vulnerability

Please do not open a public GitHub issue for vulnerabilities.

Send reports to: **ax1plee@gmail.com**

Include:

- A short description of the issue.
- Steps to reproduce.
- Expected impact.
- A suggested fix, if you have one.

The maintainer aims to respond within 48 hours, but this is an independent
open-source project and response times are best effort.

## Current Security Status

Zenthril includes several security-oriented controls:

- Argon2id password hashing.
- Short-lived JWT access tokens.
- Redis-backed refresh token rotation and replay detection.
- Redis-backed access token blacklist on logout.
- Client access tokens are process-memory only; legacy `localStorage` copies
  are migrated and removed.
- Cookie-backed sessions can restore a memory-only access token after reload
  without exposing refresh-token material to JavaScript.
- Privacy-first client startup for saved sessions: no guild/API load or
  WebSocket connection before explicit Connect by default.
- Strict CORS and WebSocket Origin allowlists.
- One-time WebSocket tickets.
- Basic HTTP security headers.
- Message payload size and envelope validation.
- PostgreSQL parameterized queries in the current backend.
- Production config validation for secrets and browser origins.
- RBAC v2 role escalation protections for guild roles and moderation actions.

## E2EE Status

Zenthril is **not** Signal-grade and has not been externally audited.

Current alpha crypto work includes:

- X25519 key agreement primitives.
- HKDF-SHA256 derivation before AES-GCM key import.
- AES-256-GCM message encryption with protocol versioning.
- AES-GCM associated data binding legacy payloads to `protocol_version` and
  `key_id`.
- Protocol-v2 E2EE envelope binds ciphertext to channel, sender user, sender
  device, session, client message id, key id, and cipher suite context.
- Device key registration and revocation foundations.
- Safety number foundations.

Known limitations:

- Full X3DH session setup is not integrated into all real message flows.
- Full Double Ratchet, skipped-message-key handling, and session healing are not complete.
- Multi-device recovery and key backup are not production-ready.
- Protocol-v1 messages remain supported for alpha compatibility, but new
  message sends should use the stronger protocol-v2 AAD context.
- Messages created before the protocol-v1 envelope stored an incomplete server-side payload and may not be decryptable from history.
- No independent cryptographic audit has been completed.

Do not use Zenthril for highly sensitive communication at this stage.

## Key Storage Status

Production web builds intentionally refuse to store private E2EE material in
`localStorage` unless `VITE_ALLOW_INSECURE_KEY_STORAGE=true` is explicitly set.

The current Tauri desktop client now prefers OS-backed key storage through the
Rust `keyring` crate. On supported platforms this maps to Windows Credential
Manager, macOS Keychain, or Linux Secret Service. The older `tauri-plugin-store`
path remains only as an alpha migration fallback: when a legacy bundle is found,
the client command migrates it into the OS key storage path and removes the
legacy copy.

The device key store now exposes an explicit storage status to the client UI and
rejects locally stored bundles when their `userId` does not match the active
user context. This prevents accidental cross-account key loading, but it does
not protect against a compromised local machine or XSS in an allowed insecure
web-development build. OS keychain storage is a major improvement over the
temporary Tauri Store path, but it is not a substitute for full device
verification, secure recovery, and external audit.

## Device Verification Status

The client includes a local safety-number verification foundation. It can derive
a stable safety number for a local/remote device pair, remember that local
verification decision, and warn when the remote identity key changes after that
verification. The device-management UI exposes the local verification state,
the safety number, and manual mark/clear controls.

This is still an alpha mechanism:

- verification records are local to the current client profile;
- verification is not yet synchronized across devices;
- QR scanning UI is not complete;
- server-side trust state is not treated as authoritative for E2EE trust;
- a compromised local machine can still tamper with local verification records.

## Startup Network Activity

Saved client sessions start offline by default. Until the user presses
**Connect**, the client should not load the remote server list, call application
APIs, request WebSocket tickets, start federation discovery, or start P2P
discovery. Users may explicitly enable "Connect automatically on startup";
that setting trades startup privacy for convenience.

## Known Alpha Limitations

- Zenthril does not implement network restriction circumvention and does not
  promise invisible, indistinguishable, or impossible-to-block traffic.
- Federation is scaffolded but disabled by default and not production-ready.
- Hybrid voice is experimental and not security-audited.
- WebRTC voice/P2P defaults to relay-only ICE in production builds to reduce IP
  leakage. Operators must configure trusted `turn:` / `turns:` servers through
  `VITE_WEBRTC_ICE_SERVERS`; STUN-only entries are filtered out in relay-only mode.
- Multi-node deployment is not complete.
- Observability is useful for alpha testing but not yet a full production SRE setup.
- Abuse prevention and moderation tooling are basic.

## RBAC / Role Security

Guild authorization is moving to RBAC v2:

- Members may have multiple roles through `guild_member_roles`.
- Effective permissions are calculated as the bitwise OR of all assigned roles.
- Role `level` is used for hierarchy only, not as a replacement for permissions.
- Non-owners cannot manage or assign roles at or above their highest role level.
- Non-owners cannot grant permissions they do not already have.
- System roles are protected from deletion and dangerous edits.
- The developer super-admin / "god role" is configuration-only through `ADMIN_USER_IDS` and cannot be granted through guild role APIs.
- Banned members have no effective permissions.

See [docs/RBAC.md](docs/RBAC.md) for the current model and migration notes.

See [THREAT_MODEL.md](THREAT_MODEL.md) for current assumptions and trust boundaries.
