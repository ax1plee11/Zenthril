# Red Team Audit Report

Project: Zenthril alpha secure messenger.

Date: 2026-05-31.

Scope: repository review plus defensive regression coverage for WebSocket, JWT,
CORS, E2EE envelope, auth/session, access-control, client rendering, WebRTC
metadata, and supply-chain risk.

## Executive Summary

Zenthril already contains several important hardening controls:

- WebSocket Origin allowlist and missing-Origin rejection.
- One-time WebSocket ticket flow.
- Per-connection and per-user WebSocket rate limiting.
- JWT validation pinned to HS256 with expiration required.
- Refresh token rotation, Redis tracking, and replay detection.
- Strict CORS allowlist and preflight validation.
- Security headers middleware.
- Message envelope validation for `ciphertext`, `iv`, `tag`, and `protocol_version`.
- HKDF-based E2EE message key derivation foundation.

The project remains alpha. The largest residual risks are incomplete Signal-grade E2EE, metadata exposure, partial device verification, no external security audit, and limited end-to-end security regression coverage.

## Findings

| ID | Area | Risk | Current Result | Main Recommendation |
|---|---|---:|---|---|
| RT-01 | CSWSH | Critical | Mitigated by strict WS Origin allowlist and one-time tickets | Keep fail-closed config and test in CI |
| RT-02 | JWT `alg:none` / confusion | Critical | Mitigated by HS256 pinning and valid-method check | Add negative tests for malformed token families |
| RT-03 | WS flooding / resource exhaustion | High | Partially mitigated by message size and rate limits | Add ping/pong deadlines and connection caps |
| RT-04 | Missing WS Origin | Critical | Mitigated by empty-Origin rejection | Keep non-browser tooling documented |
| RT-05 | E2EE MITM / ratchet bypass | Critical | Not fully solved; HKDF envelope foundation only | Implement X3DH, Double Ratchet, safety numbers, skipped keys |
| RT-06 | Rate-limit bypass | High | Partial: IP, spam, WS limits exist | Add distributed identity-aware limits and tests |
| RT-07 | IDOR | High | Many handlers check membership/author/admin | Add integration tests for cross-user access |
| RT-08 | Message injection / XSS | High | E2EE ciphertext reduces server-side content inspection | Enforce safe React rendering and sanitize rich previews |
| RT-09 | Refresh-token replay | High | Mitigated by Redis `GETDEL` rotation and used-token markers | Add concurrency replay test |
| RT-10 | CORS misconfiguration | High | Mitigated by exact-origin allowlist and no wildcard | Test prod config fail-closed |
| RT-11 | Supply chain | Medium | Lockfiles exist; CI should audit deps | Add govulncheck/npm audit/CodeQL/Dependabot |
| RT-12 | Metadata leakage | Medium | Still exposed: sender/channel/timing/ciphertext size | Document clearly; consider padding/batching |
| RT-13 | WebRTC ICE leak | Medium | Voice/P2P can reveal IPs unless TURN/Tor-aware policy is used | Add relay-only mode and UI warning |
| RT-14 | Admin privilege escalation | High | Admin routes use `ADMIN_USER_IDS` allowlist | Add negative tests with normal user tokens |
| RT-15 | Brute force / stuffing | High | Login route has guard; no MFA yet | Implement TOTP/WebAuthn and lockout telemetry |

## Production Blockers

- E2EE is not Signal-grade and not externally audited.
- Key storage is still not mature enough for high-risk production use.
- WebRTC direct connectivity can leak IP metadata.
- Federation remains alpha and disabled by default.
- Some security checks are unit-tested, but not all are covered by integration tests.

## Verification Strategy

The public repository intentionally avoids shipping executable attack PoCs.
Security checks should be represented as ordinary unit/integration tests and CI
jobs. A secure result is usually rejection with `401`, `403`, close frames, or
safe validation errors.

Add these checks to CI gradually:

- JWT negative tests.
- WS Origin negative tests.
- Message envelope negative tests.
- Refresh replay tests.
- Admin/IDOR integration tests.
- Dependency audit jobs.
