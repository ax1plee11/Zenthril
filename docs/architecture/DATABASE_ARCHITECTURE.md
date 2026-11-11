# Zenthril Database Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Database Overview

**Primary Database:** PostgreSQL 16  
**Connection Library:** jackc/pgx/v5  
**Migration Strategy:** SQL migration files (001-015)  
**Connection Pooling:** pgxpool.Pool

---

## 2. Schema Overview

### Core Tables

| Table | Description |
|-------|-------------|
| `users` | User accounts (id, username, password_hash, public_key, created_at) |
| `global_bans` | Global user bans (user_id, banned_by, reason, expires_at, created_at) |
| `friendships` | Friend relationships (id, user_id, friend_id, status, created_at) |
| `admin_audit_log` | Admin action audit trail |
| `ip_tracking` | IP address tracking for rate limiting |
| `banned_guilds` | Banned guilds (guild_id, banned_by, reason, banned_at) |
| `device_keys` | Device X25519 key bundles |
| `user_mfa_totp` | MFA/TOTP secrets (user_id, secret_enc, enabled, created_at) |
| `message_crypto_envelopes` | E2EE message encryption envelopes |
| `federation_messages` | Inter-server federation messages |
| `guild_roles` | RBAC roles (v2) |
| `guild_member_roles` | Member-role assignments |
| `message_aad` | Message Associated Data |
| `device_identity_dh_keys` | Device identity DH keys |
| `message_recipient_envelopes` | Per-recipient message envelopes |
| `device_sessions` | Device session tracking |
| `security_log` | Security event log (no rotation policy) |

---

## 3. Migration Files (001-015)

```
001_initial_schema.sql        → users table
002_global_bans.sql           → global_bans table
003_friends.sql               → friendships table
004_admin_audit_log.sql       → admin_audit_log table
005_ip_tracking_and_guild_bans.sql → ip_tracking, banned_guilds
006_device_keys.sql           → device_keys table
007_add_mfa_totp.sql          → user_mfa_totp table
008_message_crypto_envelope.sql → message_crypto_envelopes table
009_federation_messages.sql   → federation_messages table
010_rbac_v2.sql               → guild_roles, guild_member_roles
011_message_aad_v2.sql        → message_aad table
012_device_identity_dh_key.sql → device_identity_dh_keys table
013_message_recipient_envelopes.sql → message_recipient_envelopes table
014_device_sessions.sql       → device_sessions table
015_recipient_envelope_dh_public_key.sql → recipient_envelope_dh_public_keys table
```

---

## 4. Connection Configuration

```go
type PostgresConfig struct {
    Shards []PostgresShardConfig
}

type PostgresShardConfig struct {
    ID     string
    DSN    string
    Weight int
}
```

Connection string format: `postgres://user:password@host:port/dbname?sslmode=disable`

**Connection Pool Settings (recommended):**
- MaxConns: 25
- MinConns: 5
- MaxConnLifetime: 30 minutes
- MaxConnIdleTime: 5 minutes

---

## 5. Indexes (Planned)

| Table | Column(s) | Index Type | Reason |
|-------|-----------|------------|--------|
| `users` | `username` | UNIQUE | Lookup by username |
| `users` | `id` | PRIMARY KEY | User lookup |
| `device_keys` | `user_id` | INDEX | Device lookup by user |
| `device_keys` | `device_id` | UNIQUE | Device lookup by ID |
| `friendships` | `user_id, friend_id` | UNIQUE | Prevent duplicate friendships |
| `messages` | `channel_id, created_at` | INDEX | Message history queries |
| `messages` | `sender_id` | INDEX | Messages by sender |
| `security_log` | `created_at` | INDEX | Log cleanup queries |
| `security_log` | `event_type` | INDEX | Security event queries |
| `user_mfa_totp` | `user_id` | PRIMARY KEY | MFA lookup by user |
| `global_bans` | `user_id` | UNIQUE | Ban lookup |
| `federation_messages` | `created_at` | INDEX | Federation message cleanup |

---

## 6. Transactions

Critical operations that must be transactional:
- User registration (insert user + issue tokens)
- Message send (store message + create recipient envelopes + publish event)
- Friend request (create friendship + notification)
- Guild creation (create guild + add creator as member + assign admin role)
- Device registration (store keys + update user device list)
- TOTP setup (store secret + mark pending + audit log)

---

## 7. Retention Policies (Planned)

| Data Type | Retention | Policy |
|-----------|-----------|--------|
| Security logs | 90 days | Delete older entries |
| Rate limit counters | 24 hours | Redis TTL |
| Refresh tokens | 30 days | Redis TTL |
| WebSocket tickets | 2 minutes | Redis TTL |
| Access token blacklist | 15 minutes (access TTL) | Redis TTL |
| Message history | Indefinite | User-controlled deletion |
| Device sessions | 90 days inactive | Delete on logout or expiry |

---

## 8. Backup and Restore

**Planned:** Automated daily PostgreSQL backups with point-in-time recovery.

**Backup Command:**
```bash
pg_dump -h localhost -U zenthril -d zenthril > backup_$(date +%Y%m%d).sql
```

**Restore Command:**
```bash
psql -h localhost -U zenthril -d zenthril < backup_YYYYMMDD.sql
```

**Redis Backup:** RDB snapshots via Redis appendonly file.

---

## 9. Data Classification

| Data Type | Stored On Server | Stored On Client | Encrypted at Rest |
|-----------|-----------------|-----------------|-------------------|
| User credentials | Yes (hashed) | No | Yes (Argon2id hash) |
| E2EE private keys | No | Yes | No (planned: password-derived) |
| E2EE public keys | Yes | Yes | No |
| Message ciphertext | Yes | Yes | Yes (AES-256-GCM) |
| Message plaintext | No | Yes (decrypted) | N/A |
| User profile | Yes | Yes | No |
| Device info | Yes | Yes | No |
| MFA secret | Yes (hashed) | No (base32 in app) | No (planned) |

---

## 10. Planned Improvements

1. **Repository Pattern** - Abstract database access behind interfaces
2. **Migration Runner** - Programmatic migration execution with up/down support
3. **Connection Pool Tuning** - Production-appropriate pool settings
4. **Read Replicas** - Support for read-only replicas
5. **Sharding** - Horizontal sharding by user ID for scale
6. **Audit Triggers** - Database-level audit logging for sensitive operations
7. **Row-Level Security** - PostgreSQL RLS for tenant isolation

---
