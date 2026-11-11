# Zenthril Backend Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Backend Structure

The backend is written in Go 1.26.5 using the Chi router. It follows a modular monolith structure with the following packages:

```
backend/
├── main.go                    # Legacy HTTP server entrypoint
├── cmd/
│   ├── api/                   # Next-gen API (fx DI container, experimental)
│   ├── worker/                # Background worker skeleton
│   ├── sfu/                   # SFU control plane skeleton
│   └── resetpw/               # Password reset skeleton
├── auth/                      # Authentication (JWT, TOTP, refresh tokens)
├── device/                    # Device management (key bundles, registration)
├── federation/                # Inter-server federation (fail-closed)
├── friends/                   # Friend relationships
├── guild/                     # Guilds, channels, roles, moderation
├── hub/                       # WebSocket hub
├── message/                   # Message send, edit, delete, history
├── metrics/                   # Prometheus metrics
├── middleware/                # HTTP middleware (security headers, CORS)
├── models/                    # Data models
├── moderation/                # Moderation actions (bans, mutes)
├── security/                  # Rate limiting, brute force, IP blocking
├── spam/                      # Spam protection
├── user/                      # User profiles, search
├── internal/
│   ├── app/                   # fx DI container, module wiring
│   ├── config/                # Configuration loading and validation
│   ├── cqrs/                  # Command/query bus, event store
│   ├── crypto/                # X3DH, Double Ratchet, key derivation
│   ├── domain/                # Domain types
│   ├── event/                 # Event bus (memory, kafka)
│   ├── gateway/               # WebSocket gateway, session validation
│   ├── observability/         # Tracing, metrics
│   ├── pubsub/                # Pub/sub sharding
│   └── repository/            # Repository pattern (planned)
├── migrations/                # SQL database migrations (001-015)
├── migrations_test/           # Migration tests
├── static/                    # Static files
├── go.mod                     # Go module definition
├── Dockerfile                 # Backend Docker image
└── .env                       # Local environment (secrets - NOT for commit)
```

---

## 2. Package Responsibilities

| Package | Layer | Responsibility |
|---------|-------|---------------|
| `auth` | App/Security | Registration, login, JWT, refresh rotation, TOTP/MFA |
| `device` | Domain/App | Device registration, key bundle management, revocation |
| `federation` | App/Transport | Inter-server announcement, peer discovery, inbox |
| `friends` | Domain/App | Friend requests, acceptance, decline, listing |
| `guild` | Domain/App | Guild CRUD, channels, roles, members, invites |
| `hub` | Realtime | WebSocket connection management, message broadcasting |
| `message` | App/Transport | Message send, edit, delete, history, recipient envelopes |
| `metrics` | Observability | Prometheus metrics collection |
| `middleware` | Transport | Security headers, CORS, request logging |
| `models` | Domain | Data models (User, Device, Message, Guild, etc.) |
| `moderation` | App/Security | Guild bans, cascade bans, member management |
| `security` | Security | IP rate limiting, brute-force protection, security logging |
| `spam` | Security | Spam protection, message rate limiting |
| `user` | Domain/App | User profiles, search |
| `internal/app` | Infrastructure | fx DI container, lifecycle management |
| `internal/config` | Infrastructure | Configuration loading, validation |
| `internal/cqrs` | Application | Command bus, query bus, event store |
| `internal/crypto` | Domain/Security | X3DH, Double Ratchet, HKDF, AES-GCM |
| `internal/domain` | Domain | Domain types and value objects |
| `internal/event` | Infrastructure | Event bus implementations |
| `internal/gateway` | Transport/Security | WebSocket gateway, session validation, JWT auth |
| `internal/observability` | Observability | Tracing, structured logging |
| `internal/pubsub` | Infrastructure | Pub/sub sharding |
| `internal/repository` | Persistence | Repository pattern (planned, not yet implemented) |

---

## 3. Current Issues

### Issue 1: No Repository Pattern
Handlers and services access `pgxpool.Pool` directly. No repository abstraction exists.
**Fix:** Introduce `internal/repository/` with interfaces for each entity.

### Issue 2: Two Entry Points
`backend/main.go` (legacy) and `backend/cmd/api/main.go` (fx experimental) both exist.
**Fix:** Consolidate into single entry point or clearly deprecate legacy.

### Issue 3: backend-2.0 Empty
Promised clean architecture backend is not implemented.
**Fix:** Complete backend-2.0 or consolidate into backend 1.x.

### Issue 4: Missing Tests
15+ packages have no test files.
**Fix:** Add unit tests for all packages.

---

## 4. API Endpoints

### Authentication
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/auth/register | No | User registration |
| POST | /api/v1/auth/login | No | User login |
| POST | /api/v1/auth/logout | Cookie | User logout |
| POST | /api/v1/auth/refresh | Cookie | Token refresh |
| POST | /api/v1/auth/logout-all | JWT | Logout all sessions |
| POST | /api/v1/mfa/totp/start | JWT | Start TOTP setup |
| POST | /api/v1/mfa/totp/confirm | JWT | Confirm TOTP setup |
| POST | /api/v1/mfa/totp/disable | JWT | Disable TOTP |
| GET | /api/v1/auth/me | JWT | Get current user |

### Devices
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/v1/devices/ | JWT | List own devices |
| POST | /api/v1/devices/register | JWT | Register new device |
| DELETE | /api/v1/devices/{deviceId} | JWT | Revoke device |
| POST | /api/v1/key-bundles/claim | JWT | Claim key bundle |
| GET | /api/v1/users/{userId}/devices | JWT | List user's devices |

### Guilds
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/v1/guilds/ | JWT | List user's guilds |
| POST | /api/v1/guilds/ | JWT | Create guild |
| POST | /api/v1/guilds/{guildId}/invites | JWT | Create invite |
| GET | /api/v1/guilds/{guildId}/roles | JWT | List roles |
| POST | /api/v1/guilds/{guildId}/roles | JWT | Create role |
| PATCH | /api/v1/guilds/{guildId}/roles/{roleId} | JWT | Update role |
| DELETE | /api/v1/guilds/{guildId}/roles/{roleId} | JWT | Delete role |
| GET | /api/v1/guilds/{guildId}/members | JWT | List members |
| DELETE | /api/v1/guilds/{guildId}/members/{userId} | JWT | Remove member |
| PATCH | /api/v1/guilds/{guildId}/members/{userId}/role | JWT | Assign role |
| PUT | /api/v1/guilds/{guildId}/members/{userId}/roles/{roleId} | JWT | Add role |
| DELETE | /api/v1/guilds/{guildId}/members/{userId}/roles/{roleId} | JWT | Remove role |
| POST | /api/v1/guilds/{guildId}/members/{userId}/mute | JWT | Mute member |
| POST | /api/v1/guilds/{guildId}/members/{userId}/ban | JWT | Ban member |
| POST | /api/v1/guilds/{guildId}/channels | JWT | Create channel |
| GET | /api/v1/guilds/{guildId}/channels | JWT | List channels |

### Messages
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/v1/channels/{channelId}/e2ee-recipients | JWT | List recipient devices |
| POST | /api/v1/channels/{channelId}/messages | JWT + spam | Send message |
| GET | /api/v1/channels/{channelId}/messages | JWT | Get message history |
| PATCH | /api/v1/messages/{messageId} | JWT | Edit message |
| DELETE | /api/v1/messages/{messageId} | JWT | Delete message |

### Users
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/v1/users/{userId}/devices | JWT | List user devices |
| GET | /api/v1/users/search | JWT | Search users |

### Friends
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/v1/friends/ | JWT | List friends |
| POST | /api/v1/friends/request | JWT | Send friend request |
| POST | /api/v1/friends/{userId}/accept | JWT | Accept friend request |
| DELETE | /api/v1/friends/{userId} | JWT | Decline/remove friend |

### Admin
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/admin/users/{userId}/ban | JWT + Admin | Global ban user |
| DELETE | /api/v1/admin/users/{userId}/ban | JWT + Admin | Global unban user |

### Federation (disabled by default)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /federation/v1/announce | Federation token | Announce node |
| GET | /federation/v1/peers | Federation token | List peers |
| POST | /federation/v1/inbox | Federation token | Receive messages |

### Operational
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | Operational token | Liveness |
| GET | /healthz | Operational token | Liveness |
| GET | /ready | Operational token | Readiness |
| GET | /readyz | Operational token | Readiness |
| GET | /livez | None | Liveness (public) |
| GET | /metrics | Metrics token or Admin JWT | Prometheus metrics |
| GET | /metrics/prometheus | Metrics token or Admin JWT | Prometheus metrics |

### WebSocket
| Path | Auth | Description |
|------|------|-------------|
| /ws | JWT (via ticket) | WebSocket connection |

---

## 5. WebSocket Protocol

### Connection Flow
1. Client obtains WebSocket ticket via `POST /api/v1/auth/ws-ticket`
2. Client connects to `wss://host/ws?ticket=<ticket>`
3. Server validates ticket (one-time use, Redis GetDel)
4. Server upgrades connection, associates with user ID
5. Client and server exchange messages via JSON frames

### Message Format (Client → Server)
```json
{
  "type": "message",
  "channel_id": "uuid",
  "payload": "base64-encrypted-data",
  "ratchet_counter": 123
}
```

### Message Format (Server → Client)
```json
{
  "type": "message",
  "message_id": "uuid",
  "sender_id": "uuid",
  "channel_id": "uuid",
  "payload": "base64-encrypted-data",
  "timestamp": "ISO8601"
}
```

---

## 6. Database Schema (Migrations 001-015)

| Migration | Tables Created |
|-----------|---------------|
| 001_initial_schema.sql | users |
| 002_global_bans.sql | global_bans |
| 003_friends.sql | friendships |
| 004_admin_audit_log.sql | admin_audit_log |
| 005_ip_tracking_and_guild_bans.sql | ip_tracking, banned_guilds |
| 006_device_keys.sql | device_keys |
| 007_add_mfa_totp.sql | user_mfa_totp |
| 008_message_crypto_envelope.sql | message_crypto_envelopes |
| 009_federation_messages.sql | federation_messages |
| 010_rbac_v2.sql | guild_roles, guild_member_roles |
| 011_message_aad_v2.sql | message_aad |
| 012_device_identity_dh_key.sql | device_identity_dh_keys |
| 013_message_recipient_envelopes.sql | message_recipient_envelopes |
| 014_device_sessions.sql | device_sessions |
| 015_recipient_envelope_dh_public_key.sql | recipient_envelope_dh_public_keys |

---

## 7. Security Controls

| Control | Implementation | Location |
|---------|---------------|----------|
| Password hashing | Argon2id | `backend/auth/password.go` |
| JWT signing | HS256 | `backend/auth/jwt.go` |
| Refresh token rotation | Redis GETDEL + hash | `backend/auth/service.go` |
| Access token blacklist | Redis with TTL | `backend/auth/service.go` |
| Rate limiting | Redis fixed window | `backend/security/guard.go` |
| Brute-force protection | Redis counter + block | `backend/security/guard.go` |
| IP-based blocking | Redis block key | `backend/security/guard.go` |
| Trusted proxies | X-Forwarded-For with hop count | `backend/security/guard.go` |
| CORS | Strict allowlist | `backend/middleware/` |
| WebSocket Origin | Strict allowlist | `backend/main.go` |
| Security headers | CSP, HSTS, etc. | `backend/middleware/` |
| Input validation | Type checking + length limits | `backend/auth/handler.go` |

---

## 8. Planned Improvements

1. **Repository Pattern** - Create `internal/repository/` with interfaces
2. **Consolidate Entry Points** - Merge cmd/api into main.go or deprecate legacy
3. **Add Missing Tests** - guild, message, cmd packages
4. **Complete backend-2.0** - Or remove and consolidate
5. **Add CQRS Event Sourcing** - Wire cqrs package into legacy path
6. **Add Distributed Tracing** - OpenTelemetry in legacy main.go

---
