# Security Policy

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
- AES-GCM associated data binding the payload to `protocol_version` and `key_id`.
- Device key registration and revocation foundations.
- Safety number foundations.

Known limitations:

- Full X3DH session setup is not integrated into all real message flows.
- Full Double Ratchet, skipped-message-key handling, and session healing are not complete.
- Multi-device recovery and key backup are not production-ready.
- Messages created before the protocol-v1 envelope stored an incomplete server-side payload and may not be decryptable from history.
- No independent cryptographic audit has been completed.

Do not use Zenthril for highly sensitive communication at this stage.

## Key Storage Status

Production web builds intentionally refuse to store private E2EE material in
`localStorage` unless `VITE_ALLOW_INSECURE_KEY_STORAGE=true` is explicitly set.

The current Tauri desktop client uses `tauri-plugin-store` as a temporary local
storage mechanism. This is **not equivalent to OS keychain storage**. A future
hardening step should move private key material to Windows Credential Manager,
macOS Keychain, Linux Secret Service/KWallet, or Tauri Stronghold.

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
