# Zenthril Red Team - Attack Test Results
**Generated:** 2026-11-02  
**Classification:** INTERNAL - Security Team Only  
**Environment:** Local test instance, synthetic data only

---

## 1. Test Methodology

All attack simulations were performed against a local test instance of Zenthril using synthetic test accounts. No real user data was accessed. Each attack scenario was attempted, documented, and (where fixable) remediated.

---

## 2. Attack Test Results Summary

| Test ID | Attack Scenario | Severity | Result | Evidence |
|---------|----------------|----------|--------|----------|
| ATT-001 | Password reset token in response body | CRITICAL | CONFIRMED | Token returned in JSON response |
| ATT-002 | TOTP code brute force | HIGH | BLOCKED | Rate limiting (5/min IP) enforced |
| ATT-003 | Brute-force login | HIGH | BLOCKED | BruteForceProtect (10/min IP) |
| ATT-004 | IDOR on message edit | HIGH | BLOCKED | Author ownership check enforced |
| ATT-005 | IDOR on message delete | HIGH | BLOCKED | Author ownership check enforced |
| ATT-006 | Cross-user device key claim | MEDIUM | BLOCKED | Shared guild requirement enforced |
| ATT-007 | Cross-user device enumeration | MEDIUM | BLOCKED | Minimized projection + shared guild |
| ATT-008 | JWT algorithm confusion | CRITICAL | BLOCKED | HS256 pinned, alg=none rejected |
| ATT-009 | Refresh token replay | HIGH | BLOCKED | Redis GETDEL atomic rotation |
| ATT-010 | Access token theft via XSS | MEDIUM | MITIGATED | HttpOnly cookies |
| ATT-011 | WebSocket ticket reuse | MEDIUM | BLOCKED | One-time ticket (Redis GETDEL) |
| ATT-012 | E2EE key substitution | CRITICAL | BLOCKED | Ed25519 signed prekey verification |
| ATT-013 | SQL injection | HIGH | BLOCKED | pgx parameterized queries |
| ATT-014 | CORS bypass | MEDIUM | BLOCKED | Strict origin allowlist |
| ATT-015 | Rate limit bypass via X-Forwarded-For | MEDIUM | BLOCKED | Trusted proxies default 0 |
| ATT-016 | Password reset token exposure | CRITICAL | CONFIRMED | Token in response body (FIXED) |

---

## 3. Detailed Test Results

### ATT-001: Password Reset Token Exposure (CONFIRMED → FIXED)

**Attack:** Call `POST /api/v1/auth/password-reset/request` with victim username  
**Before fix:** Response contained `{"status":"ok","reset_token":"<plaintext>"}`  
**After fix:** Response contains `{"status":"ok"}` only  
**Re-attack:** Token no longer present in response  
**Status:** VERIFIED FIXED

### ATT-002: TOTP Code Brute Force (BLOCKED)

**Attack:** Send multiple TOTP confirm requests with different 6-digit codes  
**Protection:** `AuthRateLimit("totp_confirm", 5, time.Minute)` per IP  
**Result:** After 5 requests per minute, HTTP 429 returned  
**Status:** VERIFIED BLOCKED

### ATT-003: Brute-Force Login (BLOCKED)

**Attack:** Send multiple login attempts with wrong passwords  
**Protection:** `BruteForceProtect` middleware (10 failures per IP per minute, 15-minute block)  
**Result:** After 10 failures, IP blocked for 15 minutes  
**Status:** VERIFIED BLOCKED

### ATT-004: IDOR on Message Edit (BLOCKED)

**Attack:** User A attempts to edit message owned by User B  
**Protection:** `EditMessage` checks `author_id = $12` in UPDATE query  
**Result:** `ErrForbidden` returned  
**Status:** VERIFIED BLOCKED

### ATT-005: IDOR on Message Delete (BLOCKED)

**Attack:** User A attempts to delete message owned by User B  
**Protection:** `DeleteMessage` checks `author_id = $3` in UPDATE query  
**Result:** `ErrForbidden` returned  
**Status:** VERIFIED BLOCKED

### ATT-006: Cross-User Device Key Claim (BLOCKED)

**Attack:** User A claims key bundle for User B's device without shared guild  
**Protection:** `ClaimKeyBundle` checks shared guild membership  
**Result:** `ErrNoSharedGuild` returned  
**Status:** VERIFIED BLOCKED

### ATT-007: Cross-User Device Enumeration (BLOCKED)

**Attack:** User A lists devices for User B without shared guild  
**Protection:** `ListUserDevicesPublic` returns minimized projection, requires shared guild  
**Result:** 404 "No devices available"  
**Status:** VERIFIED BLOCKED

### ATT-008: JWT Algorithm Confusion (BLOCKED)

**Attack:** Send JWT with `alg: none` or different signing algorithm  
**Protection:** `jwt.ParseWithClaims` with `jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})` and method check in key callback  
**Result:** Token rejected with "unexpected signing method"  
**Status:** VERIFIED BLOCKED

### ATT-009: Refresh Token Replay (BLOCKED)

**Attack:** Reuse a refresh token after it has been consumed  
**Protection:** Redis `GETDEL` atomic consume + "used tokens" set for replay detection  
**Result:** Second use returns `ErrInvalidRefreshToken`  
**Status:** VERIFIED BLOCKED

### ATT-010: Access Token Theft via XSS (MITIGATED)

**Attack:** Steal access token via XSS in browser  
**Protection:** `HttpOnly` flag on access cookie prevents JavaScript access  
**Result:** Token not accessible via `document.cookie`  
**Status:** VERIFIED MITIGATED

### ATT-011: WebSocket Ticket Reuse (BLOCKED)

**Attack:** Reuse a WebSocket ticket after it has been consumed  
**Protection:** `ConsumeWSTicket` uses Redis `GETDEL` (atomic one-time consumption)  
**Result:** Second use fails with ticket not found  
**Status:** VERIFIED BLOCKED

### ATT-012: E2EE Key Substitution (BLOCKED)

**Attack:** MITM substitutes public keys during X3DH initiation  
**Protection:** `SignedPreKey` signed with `IdentitySigningPublicKey` (Ed25519), signature verified in `validatePeerBundle`  
**Result:** Tampered signature rejected with "peer signed prekey signature verification failed"  
**Status:** VERIFIED BLOCKED

### ATT-013: SQL Injection (BLOCKED)

**Attack:** Inject SQL via username, message content, or other parameters  
**Protection:** All queries use pgx parameterized statements (`$1`, `$2`, etc.)  
**Result:** Parameters treated as data, not SQL  
**Status:** VERIFIED BLOCKED

### ATT-014: CORS Bypass (BLOCKED)

**Attack:** Make cross-origin request from malicious domain  
**Protection:** `CORS` middleware with strict origin allowlist, wildcard rejected  
**Result:** Preflight rejected for non-allowed origins  
**Status:** VERIFIED BLOCKED

### ATT-015: Rate Limit Bypass via X-Forwarded-For (BLOCKED)

**Attack:** Rotate `X-Forwarded-For` header to evade per-IP rate limits  
**Protection:** `TRUSTED_PROXY_HOPS` defaults to 0, only `RemoteAddr` trusted  
**Result:** Header ignored, rate limit keyed on transport peer address  
**Status:** VERIFIED BLOCKED

### ATT-016: Password Reset Token Exposure (CONFIRMED → FIXED)

**Same as ATT-001.** See above.

---

## 4. Tests That Could Not Be Performed

| Test | Reason |
|------|--------|
| E2EE cross-language key mismatch | FIXED in previous phase (BLOCKER-001) |
| Race condition on RatchetState | Race detector not supported on Windows (windows/386) |
| ZenthrilDB integration | C++ code not integrated, no Go bindings |
| Penetration test against production | No production instance available |
| Dependency vulnerability scan | No secret/dependency scanning in CI |

---

## 5. False Positives Ruled Out

| Finding | Reason Ruled Out |
|---------|-----------------|
| Directory traversal | No file serving from user input (except static SPA with `os.Stat` check) |
| SSRF | No server-side URL fetching |
| Path traversal | No file path from user input |
| XML External Entity | No XML processing |

---
