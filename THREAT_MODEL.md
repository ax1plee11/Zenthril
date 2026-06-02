# Zenthril Threat Model

## Status

This threat model describes the current alpha implementation. It is not a claim
of production readiness or audited security.

## Assets

- User credentials and refresh tokens.
- Access tokens and WebSocket tickets.
- E2EE private identity keys, signed prekeys, one-time prekeys, root keys, and message keys.
- Message ciphertext, IVs, authentication tags, and protocol metadata.
- Guild, channel, membership, device, and moderation metadata.
- Operational tokens, metrics, database credentials, and Redis credentials.

## Attackers

- Network attackers able to observe or modify traffic without valid TLS termination.
- Malicious websites attempting CORS abuse or Cross-Site WebSocket Hijacking.
- Malicious authenticated users trying to access unauthorized channels or flood realtime paths.
- Compromised refresh tokens or access tokens.
- Compromised clients or devices.
- A compromised or malicious self-hosted server operator.
- Supply-chain attackers targeting dependencies or CI/CD.

## Trust Boundaries

- Browser/Tauri client to backend HTTP API.
- Browser/Tauri client to WebSocket gateway.
- Backend to PostgreSQL.
- Backend to Redis.
- Local client runtime to local key storage.
- Future federation boundary between independent nodes.
- Future object storage/media boundary.

## Current Mitigations

- Strict CORS and WebSocket Origin allowlists.
- One-time WebSocket tickets instead of long-lived JWTs in query strings.
- JWT signing method pinned to HS256.
- Refresh-token rotation with Redis `GETDEL` and replay detection.
- Access-token blacklist on logout.
- Argon2id password hashing.
- Request body and WebSocket message size limits.
- Message envelope validation for ciphertext, IV, tag, and protocol version.
- AES-GCM uses HKDF-derived keys and AAD for protocol-version/key-id binding.
- Production config validation fails closed for missing secrets and origins.

## Server Compromise Assumptions

Zenthril does not currently provide complete Signal-grade protection against a
malicious or compromised server. The server should not receive plaintext message
content, but it does observe metadata such as users, devices, guilds, channels,
message timing, and ciphertext sizes.

The current alpha E2EE layer does not yet provide complete X3DH/Double Ratchet
session semantics, robust compromise recovery, or mature multi-device recovery.

## Compromised Device Assumptions

If a device is compromised, local private key material and message plaintext on
that device may be compromised. Current device revocation helps stop future use
of a revoked device identity, but it does not retroactively protect plaintext
already available on that device.

## Current Limitations

- No external security audit has been completed.
- Tauri desktop storage currently uses `tauri-plugin-store`, not OS keychain.
- Production web builds refuse `localStorage` private-key persistence unless it
  is explicitly enabled for testing, and locally loaded device key bundles must
  match the active user id. This reduces accidental cross-account key loading,
  but it does not protect against a compromised local machine or XSS in an
  insecure development build.
- Old alpha messages without stored authentication tags may not decrypt from history.
- Federation, voice, and multi-node fan-out are not production-ready.
- Full X3DH and Double Ratchet are roadmap items, not complete guarantees.
