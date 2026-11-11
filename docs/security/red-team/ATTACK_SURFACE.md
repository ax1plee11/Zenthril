# Zenthril Red Team - Attack Surface Map
**Generated:** 2026-11-02  
**Classification:** INTERNAL - Security Team Only  
**Scope:** `C:\Users\User\Desktop\дс аналонг` (Zenthril backend + client)

---

## 1. Entry Points Summary

### HTTP REST API (`/api/v1`)

| Path | Method | Auth | Rate Limit | Description |
|------|--------|------|------------|-------------|
| `/api/v1/auth/register` | POST | No | 10/hr per IP | User registration |
| `/api/v1/auth/login` | POST | No | BruteForce (10/min) | User login |
| `/api/v1/auth/logout` | POST | Cookie | 30/min per IP | User logout |
| `/api/v1/auth/refresh` | POST | Cookie | 30/min per IP | Token refresh |
| `/api/v1/auth/mfa/totp/start` | POST | JWT | 5/min per IP | TOTP setup |
| `/api/v1/auth/mfa/totp/confirm` | POST | JWT | 5/min per IP | TOTP confirm |
| `/api/v1/auth/mfa/totp/disable` | POST | JWT | 5/min per IP | TOTP disable |
| `/api/v1/auth/password-reset/request` | POST | No | 3/hr per IP | Password reset request |
| `/api/v1/auth/password-reset/confirm` | POST | No | 10/hr per IP | Password reset confirm |
| `/api/v1/auth/ws-ticket` | POST | JWT | WSTicket rate limit | WebSocket ticket |
| `/api/v1/auth/logout-all` | POST | JWT | None | Logout all sessions |
| `/api/v1/auth/me` | GET | JWT | None | Current user profile |
| `/api/v1/devices/` | GET | JWT | None | List own devices |
| `/api/v1/devices/register` | POST | JWT | None | Register device |
| `/api/v1/devices/{deviceId}` | DELETE | JWT | None | Revoke own device |
| `/api/v1/key-bundles/claim` | POST | JWT | None | Claim key bundle |
| `/api/v1/guilds/` | GET | JWT | None | List user's guilds |
| `/api/v1/guilds/` | POST | JWT | None | Create guild |
| `/api/v1/guilds/{guildId}/invites` | POST | JWT | None | Create invite |
| `/api/v1/guilds/{guildId}/roles` | GET | JWT | None | List roles |
| `/api/v1/guilds/{guildId}/roles` | POST | JWT | None | Create role |
| `/api/v1/guilds/{guildId}/roles/{roleId}` | PATCH | JWT | None | Update role |
| `/api/v1/guilds/{guildId}/roles/{roleId}` | DELETE | JWT | None | Delete role |
| `/api/v1/guilds/{guildId}/members` | GET | JWT | None | List members |
| `/api/v1/guilds/{guildId}/members/{userId}` | DELETE | JWT | None | Remove member |
| `/api/v1/guilds/{guildId}/members/{userId}/role` | PATCH | JWT | None | Assign role |
| `/api/v1/guilds/{guildId}/members/{userId}/roles/{roleId}` | PUT | JWT | None | Add member role |
| `/api/v1/guilds/{guildId}/members/{userId}/roles/{roleId}` | DELETE | JWT | None | Remove member role |
| `/api/v1/guilds/{guildId}/members/{userId}/mute` | POST | JWT | None | Mute member |
| `/api/v1/guilds/{guildId}/members/{userId}/ban` | POST | JWT | None | Ban member |
| `/api/v1/guilds/{guildId}/channels` | GET | JWT | None | List channels |
| `/api/v1/guilds/{guildId}/channels` | POST | JWT | None | Create channel |
| `/api/v1/invites/{code}/join` | POST | JWT | None | Join by invite |
| `/api/v1/channels/{channelId}/e2ee-recipients` | GET | JWT | None | List recipient devices |
| `/api/v1/channels/{channelId}/messages` | GET | JWT | None | Get message history |
| `/api/v1/channels/{channelId}/messages` | POST | JWT + Spam | None | Send message |
| `/api/v1/messages/{messageId}` | PATCH | JWT | None | Edit message |
| `/api/v1/messages/{messageId}` | DELETE | JWT | None | Delete message |
| `/api/v1/users/{userId}/devices` | GET | JWT | None | List user devices |
| `/api/v1/users/search` | GET | JWT | None | Search users |
| `/api/v1/friends/` | GET | JWT | None | List friends |
| `/api/v1/friends/request` | POST | JWT | None | Send friend request |
| `/api/v1/friends/{userId}/accept` | POST | JWT | None | Accept friend request |
| `/api/v1/friends/{userId}` | DELETE | JWT | None | Decline/remove friend |
| `/api/v1/admin/users/{userId}/ban` | POST | JWT + Admin | None | Global ban |
| `/api/v1/admin/users/{userId}/ban` | DELETE | JWT + Admin | None | Global unban |

### WebSocket (`/ws`)

| Path | Auth | Description |
|------|------|-------------|
| `/ws` | JWT (via ticket) | Real-time messaging |

### Federation (`/federation/v1`)

| Path | Method | Auth | Description |
|------|--------|------|-------------|
| `/federation/v1/announce` | POST | Federation token | Announce node |
| `/federation/v1/peers` | GET | Federation token | List peers |
| `/federation/v1/inbox` | POST | Federation token | Receive messages |

### Operational (`/`)

| Path | Method | Auth | Description |
|------|--------|------|-------------|
| `/health` | GET | Operational token | Liveness |
| `/healthz` | GET | Operational token | Liveness |
| `/ready` | GET | Operational token | Readiness |
| `/readyz` | GET | Operational token | Readiness |
| `/livez` | GET | None | Liveness (public) |
| `/metrics` | GET | Metrics token/Admin JWT | Prometheus metrics |
| `/metrics/prometheus` | GET | Metrics token/Admin JWT | Prometheus metrics |

### Static SPA

| Path | Method | Auth | Description |
|------|--------|------|-------------|
| `/*` | GET | None | Frontend SPA (if STATIC_DIR set) |

---

## 2. Attack Surface by Component

### Authentication Module (`backend/auth/`)
- **Registration:** Password complexity enforced, Argon2id hashing
- **Login:** Brute force protection per IP, rate limited
- **JWT:** HS256, 15 min access TTL, 30 day refresh TTL, algorithm pinning
- **Refresh Token:** Redis GETDEL atomic rotation, hash stored, replay detection
- **TOTP/MFA:** HMAC-SHA1, 30-second step, 6 digits, base32 secret (FIXED)
- **Password Reset:** SHA-256 hashed tokens, 1-hour expiry, single-use (**ISSUE: token returned in response body**)
- **Rate Limiting:** Per-IP, per-scope, Redis-backed, fail-closed

### Device Module (`backend/device/`)
- **Registration:** X25519 key bundles, device limits
- **Listing (own):** Full device info for authenticated user
- **Listing (cross-user):** Minimized projection, requires shared guild (**FIXED**)
- **Revocation:** Own devices only
- **Key Bundle Claim:** Requires shared guild, returns public key material only

### Message Module (`backend/message/`)
- **Send:** Channel membership required, recipient envelope validation
- **History:** Channel membership required, recipient-specific envelopes
- **Edit:** Author ownership required, channel membership required
- **Delete:** Author ownership required, channel membership required
- **E2EE:** Client-side encryption, server never sees plaintext

### Guild Module (`backend/guild/`)
- **Membership:** Required for all guild/channel operations
- **Roles:** Hierarchical permissions
- **Moderation:** Ban, mute, remove member (with permission checks)
- **Invites:** Expirable, single/multi-use

### Security Module (`backend/security/`)
- **IP Rate Limit:** 1000 req/sec per IP, 60-second block
- **Brute Force:** 10 failures/min per IP, 15-minute block
- **Auth Rate Limit:** Per-endpoint per-IP budgets
- **Security Logging:** PostgreSQL `security_log` table (no rotation)
- **Trusted Proxies:** X-Forwarded-For only when explicitly configured

---

## 3. Trust Boundaries

| Boundary | Protection | Residual Risk |
|----------|-----------|---------------|
| Client ↔ Server | TLS 1.3, JWT cookies (HttpOnly, SameSite) | Token theft via XSS mitigated by HttpOnly |
| Server ↔ PostgreSQL | Parameterized queries, internal network | SQL injection mitigated by pgx |
| Server ↔ Redis | Password auth, internal network | Redis compromise exposes sessions/rate limits |
| Client ↔ Client | E2EE (X3DH + Double Ratchet, AES-256-GCM) | Server cannot decrypt messages |
| Server ↔ Federation | Federation token, fail-closed | Federation disabled by default |

---

## 4. Data Flow - Sensitive Data

| Data | Stored On | Encryption | Access Control |
|------|-----------|------------|----------------|
| User credentials | PostgreSQL | Argon2id hash | Auth service only |
| E2EE private keys | Client only | Planned: password-derived | Client device |
| E2EE public keys | PostgreSQL | None (public) | Authenticated users |
| Message ciphertext | PostgreSQL | AES-256-GCM | Channel members |
| Message plaintext | Client only | None (decrypted locally) | Client device |
| Refresh tokens | Redis | SHA-256 hash | Auth service |
| Access tokens | Client (cookie) | JWT signature | Auth service |
| TOTP secrets | PostgreSQL | Plain base32 (**ISSUE**) | Auth service |
| Security logs | PostgreSQL | None | Admin only |

---

## 5. Known Attack Vectors (Pre-Remediation)

| ID | Vector | Severity | Status |
|----|--------|----------|--------|
| RS-001 | Password reset token returned in HTTP response body | CRITICAL | OPEN |
| RS-002 | TOTP secret stored as plain base32 (not encrypted at rest) | MEDIUM | OPEN |
| RS-003 | Security log table without rotation policy | MEDIUM | OPEN |
| RS-004 | No MFA recovery codes for lost devices | HIGH | OPEN |
| RS-005 | Backend-2.0 empty skeleton (unmaintained codebase) | HIGH | OPEN |
| RS-006 | ZenthrilDB not integrated (C++ code, no Go bindings) | MEDIUM | OPEN |
