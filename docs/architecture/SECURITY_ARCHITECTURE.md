# Zenthril Security Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Security Model

Zenthril follows a **defense-in-depth** security model with multiple layers of protection:

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Network (TLS, WAF, DDoS protection)               │
├─────────────────────────────────────────────────────────────┤
│ Layer 2: Transport (CORS, WebSocket Origin, Security Hdrs) │
├─────────────────────────────────────────────────────────────┤
│ Layer 3: Authentication (JWT, MFA/TOTP, Password Hashing) │
├─────────────────────────────────────────────────────────────┤
│ Layer 4: Authorization (RBAC, Ownership, IDOR Protection)   │
├─────────────────────────────────────────────────────────────┤
│ Layer 5: Rate Limiting (IP-based, Endpoint-based, Brute Force)│
├─────────────────────────────────────────────────────────────┤
│ Layer 6: Application Security (Input Validation, SQLi Prevention)│
├─────────────────────────────────────────────────────────────┤
│ Layer 7: Cryptography (E2EE, X3DH, Double Ratchet, AES-GCM)│
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Authentication Architecture

### JWT Token Design

| Token Type | TTL | Storage | Transmission |
|------------|-----|---------|--------------|
| Access Token | 15 minutes | HttpOnly Cookie | Authorization header |
| Refresh Token | 30 days | HttpOnly Cookie | Request body or cookie |

**Token Structure:**
```json
{
  "user_id": "uuid",
  "token_type": "access|refresh",
  "iat": "timestamp",
  "exp": "timestamp",
  "jti": "uuid"
}
```

**Security Properties:**
- HS256 signing with minimum 32-byte secret
- `alg=none` rejected
- Token type validation (access vs refresh)
- Expiration required (`jwt.WithExpirationRequired()`)
- Valid methods restricted to HS256

### Refresh Token Rotation

1. Client sends refresh token
2. Server atomically consumes old token (Redis `GETDEL`)
3. Server validates token hash against stored record
4. Server issues new token pair
5. New refresh token hash stored in Redis
6. Old token ID added to "used" set for replay detection
7. On replay detection: revoke ALL user refresh tokens

### Password Hashing

- Algorithm: Argon2id
- Parameters: Configurable (memory, iterations, parallelism)
- Salt: Random 16 bytes per password
- Pepper: Application-level secret (JWT_SECRET)

### MFA/TOTP

- Algorithm: HMAC-SHA1 (RFC 6238)
- Time step: 30 seconds
- Code digits: 6
- Look-ahead: 1 time step (±30 seconds clock skew)
- Secret: 160 bits (20 bytes), base32-encoded
- Issuer: "Zenthril"

---

## 3. Authorization Architecture

### Role-Based Access Control (RBAC)

**Roles:**
- `user` - Authenticated user
- `admin` - Global administrator
- `guild_owner` - Guild creator
- `guild_admin` - Guild administrator
- `guild_moderator` - Guild moderator
- `guild_member` - Guild member

**Permission Model:**
- Resource-based: `/api/v1/guilds/{guildId}/members/{userId}`
- Action-based: `read`, `write`, `delete`, `moderate`
- Ownership check: User must own resource or have appropriate role

### IDOR Protection

All mutation endpoints must validate:
1. User is authenticated
2. User owns the resource OR has appropriate role
3. Resource exists

**Example:** `DELETE /api/v1/messages/{messageId}` must verify `message.sender_id == user_id` or user is guild admin.

---

## 4. Rate Limiting Architecture

| Limiter | Scope | Limit | Window | Backend |
|---------|-------|-------|--------|---------|
| IP Rate Limit | Per IP | 1000 req/sec | 1 second | Redis |
| Brute Force | Per IP + endpoint | 10 failures | 1 minute | Redis |
| Auth Rate Limit | Per IP + endpoint | Varies | Varies | Redis |
| WS Ticket | Per user | 30/min | 1 minute | Redis |

**Implementation:** Redis fixed-window counters with atomic `INCR` + `EXPIRE`.

---

## 5. WebSocket Security

### Connection Flow
1. Client obtains one-time WebSocket ticket via authenticated HTTP endpoint
2. Client connects with ticket in query parameter
3. Server validates ticket atomically (Redis `GETDEL`)
4. Server upgrades connection, binds to user ID
5. Server validates Origin header against allowlist

### Security Properties
- Tickets are single-use (atomic consume)
- Origin validation prevents cross-site WebSocket hijacking
- Message size limits (64KB max body)
- Connection rate limiting per IP
- Graceful connection draining on shutdown

---

## 6. Cryptography Architecture

### E2EE Protocol Stack

| Component | Algorithm | Purpose |
|-----------|-----------|---------|
| Key Exchange | X25519 | Diffie-Hellman key agreement |
| Key Derivation | HKDF-SHA256 | Derive keys from shared secrets |
| Session Init | X3DH | Extended Triple Diffie-Hellman |
| Symmetric Ratchet | HKDF-SHA256 | Chain key derivation |
| DH Ratchet | X25519 + HKDF | Asymmetric ratchet turns |
| Message Encryption | AES-256-GCM | Authenticated encryption |
| Identity Signing | Ed25519 | Signed prekey signatures |
| Safety Numbers | SHA-256 | Key fingerprint comparison |

### Key Hierarchy

```
Identity Key (X25519, long-term)
├── Signed Prekey (X25519, medium-term, signed by Identity)
└── One-Time Prekeys (X25519, single-use)

Session Keys (derived via X3DH):
├── Root Key (32 bytes)
├── Send Chain Key (32 bytes)
└── Recv Chain Key (32 bytes)

Message Keys (derived via Double Ratchet):
├── AES-256 Key (32 bytes)
├── AES-GCM Nonce (12 bytes)
└── Counter (4 bytes)
```

### Key Storage

| Key Type | Location | Protection |
|----------|----------|------------|
| Identity Private Key | Client local storage | Planned: password-derived encryption |
| Signed Prekey Private Key | Client local storage | Planned: password-derived encryption |
| One-Time Prekey Private Key | Client local storage | Planned: password-derived encryption |
| Session Root/Chain Keys | Client local storage | Planned: password-derived encryption |
| Device Public Keys | Server PostgreSQL | None (public) |
| Signed Prekey Public Key | Server PostgreSQL | None (public) |
| One-Time Prekey Public Key | Server PostgreSQL | None (public, consumed on use) |

---

## 7. Security Headers

| Header | Value | Purpose |
|--------|-------|---------|
| `Content-Security-Policy` | Restrictive | Prevent XSS, data injection |
| `Strict-Transport-Security` | max-age=31536000 | Force HTTPS |
| `X-Frame-Options` | DENY | Prevent clickjacking |
| `X-Content-Type-Options` | nosniff | Prevent MIME sniffing |
| `Referrer-Policy` | strict-origin-when-cross-origin | Limit referrer info |
| `Permissions-Policy` | Restrictive | Limit browser features |

---

## 8. Secrets Management

| Secret | Storage | Rotation |
|--------|---------|----------|
| JWT_SECRET | Environment variable | Manual |
| PostgreSQL password | Environment variable | Manual |
| Redis password | Environment variable | Manual |
| Operational token | Environment variable | Manual |
| Metrics token | Environment variable | Manual |
| Federation token | Environment variable | Manual |
| TOTP secrets | PostgreSQL (hashed) | User-initiated |
| Refresh tokens | Redis (hashed) | Automatic on rotation |

**Issue:** `backend/.env` contains plaintext secrets and is not in `.gitignore` properly. Must be excluded from version control.

---

## 9. Security Logging

All security events are logged to PostgreSQL `security_log` table:

| Event Type | Description |
|------------|-------------|
| `auth_fail` | Authentication failure |
| `brute_force_detected` | Brute force pattern detected |
| `ip_blocked` | IP temporarily blocked |
| `rate_limited` | Rate limit exceeded |
| `refresh_token_replay` | Refresh token replay detected |
| `totp_setup` | TOTP setup initiated |
| `totp_confirm` | TOTP confirmation |
| `global_ban` | User globally banned |
| `device_revoked` | Device key revoked |

**Issue:** No rotation/retention policy for security_log table.

---

## 10. Threat Model Summary

| Threat | Asset | Mitigation | Residual Risk |
|--------|-------|------------|---------------|
| Server compromise | Message ciphertext | E2EE (server can't decrypt) | Low |
| Client compromise | E2EE private keys | Planned: password-derived encryption | Medium |
| Database breach | User password hashes | Argon2id | Low |
| Replay attack | Refresh tokens | Redis GETDEL + used token set | Low |
| MITM attack | Key exchange | X3DH + Ed25519 signatures | Medium (no safety numbers UI) |
| Brute force | TOTP codes | Rate limiting (planned fix) | High (until fixed) |
| CSRF | State-changing requests | SameSite cookies + Origin validation | Low |
| XSS | Client data | CSP + HttpOnly cookies | Low |

---

## 11. Known Security Gaps (Must Fix Before Release)

1. **TOTP verification broken** - SHA-256 hash stored instead of base32 secret
2. **No rate limiting on TOTP endpoints** - Brute force vulnerable
3. **No safety numbers UI** - MITM undetectable by users
4. **Local key storage unencrypted** - Device compromise exposes keys
5. **backend/.env contains plaintext secrets** - Secret exposure risk

---
