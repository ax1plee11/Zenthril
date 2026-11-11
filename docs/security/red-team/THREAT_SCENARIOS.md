# Zenthril Red Team - Threat Scenarios
**Generated:** 2026-11-02  
**Classification:** INTERNAL - Security Team Only

---

## 1. Attacker Scenarios

### Scenario A: Unauthorized External Attacker
**Initial privileges:** None  
**Resources:** Network access to the Zenthril instance  
**Goals:** Account takeover, data exfiltration, service disruption  
**Attack paths:**
1. Brute-force login (mitigated by rate limiting)
2. Password reset token interception (if token exposed in response)
3. Registration spam (mitigated by rate limiting)
4. WebSocket ticket abuse (mitigated by rate limiting)
5. XSS via static SPA (mitigated by CSP)

### Scenario B: Registered User (No Special Rights)
**Initial privileges:** Authenticated, owns devices, can create guilds/channels  
**Resources:** Own JWT access token, refresh token, device keys  
**Goals:** Read other users' messages, escalate privileges, access other guilds  
**Attack paths:**
1. IDOR on message edit/delete (mitigated by author ownership checks)
2. IDOR on guild member management (mitigated by permission checks)
3. Device key bundle claim without shared guild (mitigated by guild check)
4. Cross-user device enumeration (mitigated by shared guild requirement)

### Scenario C: Attacker with Stolen Access Token
**Initial privileges:** Valid JWT access token for victim user  
**Resources:** Stolen access token (15-minute TTL)  
**Goals:** Impersonate victim, send messages, access guilds  
**Attack paths:**
1. Use token until expiry (15 minutes)
2. Refresh token not accessible (HttpOnly cookie)
3. WebSocket ticket issuance with stolen token
4. Cannot escalate to admin without admin user ID

### Scenario D: User with Device Access
**Initial privileges:** Physical access to own device  
**Resources:** Device local storage, E2EE private keys  
**Goals:** Extract keys, read message history, impersonate user  
**Attack paths:**
1. Extract unencrypted E2EE keys from local storage (**OPEN**)
2. Read cached message history
3. Replay messages using stored session state

### Scenario E: Malicious Guild Member
**Initial privileges:** Member of a target guild  
**Resources:** Guild access, channel access  
**Goals:** Read messages from other channels, escalate roles  
**Attack paths:**
1. Access channels without permission (mitigated by channel membership checks)
2. Assign roles without permission (mitigated by role assignment checks)
3. Ban/kick other members (mitigated by permission checks)

### Scenario F: Compromised Client
**Initial privileges:** Full control of client application  
**Resources:** All user data, E2EE keys, session state  
**Goals:** Decrypt messages, exfiltrate keys, impersonate user  
**Attack paths:**
1. Extract E2EE private keys from local storage (**OPEN**)
2. Read all decrypted message history
3. Send messages as the user

### Scenario G: Limited Server Infrastructure Access
**Initial privileges:** Access to server filesystem or database  
**Resources:** PostgreSQL, Redis, server logs  
**Goals:** Read message ciphertext, extract user credentials, forge tokens  
**Attack paths:**
1. Read PostgreSQL message table (ciphertext only, E2EE protects content)
2. Read PostgreSQL user table (Argon2id hashes, not reversible)
3. Read Redis refresh tokens (hashed, single-use)
4. Forge JWT tokens (requires JWT_SECRET, not in database)
5. Read security logs (may contain IP addresses, event types)

---

## 2. Threat Model by Asset

| Asset | Threat | Likelihood | Impact | Risk Level | Mitigation |
|-------|--------|------------|--------|------------|------------|
| User credentials | Argon2id hash cracking | Low | High | Medium | Strong hashing, rate limiting |
| E2EE private keys (client) | Local storage extraction | Medium | Critical | High | Password-derived encryption (**OPEN**) |
| E2EE public keys (server) | Key substitution | Low | High | Medium | Ed25519 signed prekeys |
| Message ciphertext | Server decryption | Low | Critical | Low | Client-side E2EE |
| Refresh tokens | Redis compromise | Low | High | Medium | SHA-256 hashing, rotation |
| Access tokens | Token theft via XSS | Low | High | Medium | HttpOnly cookies |
| TOTP secrets | Server database read | Medium | High | High | Plain base32 storage (**OPEN**) |
| Security logs | Log reading | Low | Medium | Medium | No rotation (**OPEN**) |
| Password reset tokens | Token interception | High | Critical | Critical | Returned in response body (**OPEN**) |

---

## 3. STRIDE Analysis

| Category | Threat | Asset | Mitigation |
|----------|--------|-------|------------|
| **Spoofing** | Impersonate user with stolen JWT | Sessions | HttpOnly cookies, 15-min TTL |
| **Tampering** | Modify stored messages | Messages | E2EE, author ownership checks |
| **Repudiation** | Deny sending messages | Messages | Server stores ciphertext with author ID |
| **Information Disclosure** | Read other users' messages | Messages | Channel membership checks, E2EE |
| **Denial of Service** | Flood rate limits | API | IP rate limiting, Redis blocks |
| **Elevation of Privilege** | Access admin endpoints | Admin | `adminOnly` middleware, configured admin IDs |

---

## 4. OWASP Top 10 2021 Mapping

| OWASP Category | Status | Evidence |
|----------------|--------|----------|
| A01: Broken Access Control | PARTIALLY MITIGATED | Ownership checks on messages, guild permission checks, cross-user device guild requirement |
| A02: Cryptographic Failures | PARTIALLY MITIGATED | Argon2id passwords, HS256 JWT, X25519/AES-GCM E2EE, but TOTP secrets plain (**OPEN**) |
| A03: Injection | MITIGATED | pgx parameterized queries, input validation |
| A04: Insecure Design | PARTIALLY MITIGATED | Defense in depth, but password reset token in response (**OPEN**) |
| A05: Security Misconfiguration | PARTIALLY MITIGATED | CORS allowlist, security headers, trusted proxies default 0 |
| A06: Vulnerable Components | UNKNOWN | No dependency scanning in CI |
| A07: Identification/Auth Failures | PARTIALLY MITIGATED | JWT rotation, refresh token hashing, MFA (**FIXED**) |
| A08: Software/Data Failures | PARTIALLY MITIGATED | Error handling, no stack traces in responses |
| A09: Logging Failures | PARTIALLY MITIGATED | Security logging, but no rotation (**OPEN**) |
| A10: SSRF | MITIGATED | No server-side URL fetching |

---

## 5. Attack Tree - Account Takeover

```
ROOT: Account Takeover
├── Brute-force password
│   └── Rate limited (10/min IP, 15-min block) → MITIGATED
├── Password reset token theft
│   └── Token returned in response body → VULNERABLE (RS-001)
│   └── CSRF on /password-reset/request → Possible if no SameSite enforcement
├── Refresh token theft
│   └── HttpOnly cookie → MITIGATED
│   └── XSS on client → Possible
├── TOTP bypass
│   └── No rate limit on confirm beyond IP limit → PARTIAL (5/min IP limit added)
└── Session hijacking
    └── Access token 15-min TTL → MITIGATED (short window)
```

---

## 6. Attack Tree - Message Interception

```
ROOT: Read Other Users' Messages
├── Direct API access to /messages
│   └── Channel membership check → MITIGATED
├── Cross-user device key claim
│   └── Shared guild requirement → MITIGATED
├── Server database read
│   └── Ciphertext only, E2EE → MITIGATED
└── Client-side key extraction
    └── Local storage unencrypted → VULNERABLE (RS-005 equivalent)
```

---
