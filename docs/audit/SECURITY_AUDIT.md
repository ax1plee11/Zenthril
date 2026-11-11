# Zenthril Security Audit Report
**Generated:** 2026-11-02  
**Scope:** Authentication, authorization, cryptography, infrastructure security  
**Classification:** INTERNAL - For engineering use only

---

## 1. Executive Summary

Zenthril has a solid security foundation with JWT authentication, refresh token rotation, Argon2id password hashing, rate limiting, and E2EE building blocks. However, **three critical security defects** prevent production deployment:

1. TOTP/MFA verification is non-functional (stores hash, tries base32-decode)
2. No rate limiting on TOTP endpoints (brute-force vulnerable)
3. E2EE cross-language key derivation mismatch (Go vs TypeScript)

**Overall security status:** Alpha / Not production ready.

---

## 2. Authentication Security

### CRITICAL

#### SEC-001: TOTP Secret Storage and Verification Mismatch
- **File:** `backend/auth/service.go:452` and `:483`
- **Description:** `SetupTOTP()` stores `hashToken(secret)` = SHA-256(base32 secret) as hex string. `VerifyTOTPSetup()` calls `base32Decode(secretHash)` on the SHA-256 hex output, which always fails.
- **Impact:** TOTP setup and confirmation always fail. MFA cannot be enabled by any user.
- **Fix:** Store raw base32-encoded TOTP secret. Encrypt at rest if required by compliance.

#### SEC-002: No Rate Limiting on TOTP Endpoints
- **File:** `backend/main.go:284-286`
- **Description:** Routes `/mfa/totp/start`, `/mfa/totp/confirm`, `/mfa/totp/disable` lack `AuthRateLimit` middleware.
- **Impact:** 6-digit TOTP codes (1,000,000 space) can be brute-forced without restriction.
- **Fix:** Apply `secGuard.AuthRateLimit("totp_confirm", 5, time.Minute)` to all TOTP routes.

### HIGH

#### SEC-003: No Password Reset / Account Recovery
- **File:** `backend/auth/handler.go`, `backend/auth/service.go`
- **Description:** No password reset endpoints exist. No MFA reset procedure for lost devices.
- **Impact:** Permanent account lockout on forgotten password or lost MFA device.
- **Fix:** Implement time-limited password reset tokens and MFA recovery codes.

#### SEC-004: No Account Enumeration Protection on Login
- **File:** `backend/auth/service.go:95-125`
- **Description:** Login returns `ErrInvalidCredentials` for both wrong password and non-existent user, but timing differences may exist between `pgx.ErrNoRows` path and password verification path.
- **Impact:** Potential user enumeration via timing analysis.
- **Fix:** Ensure constant-time response regardless of user existence.

### MEDIUM

#### SEC-005: JWT Secret Length Not Enforced in Non-Production
- **File:** `backend/internal/config/config.go:182`
- **Description:** JWT_SECRET minimum 32-byte check only applies in production. Development accepts any length.
- **Impact:** Weak JWT secrets in development could be brute-forced.
- **Fix:** Enforce minimum length in all environments.

#### SEC-006: Refresh Token Replay Detection Revokes All User Tokens
- **File:** `backend/auth/service.go:184-188`
- **Description:** On refresh token replay detection, `revokeUserRefreshTokens()` revokes ALL refresh tokens for the user, forcing re-login.
- **Impact:** Legitimate user inconvenienced by attacker's replay attempt.
- **Fix:** This is acceptable security behavior (revoke all on suspected compromise).

#### SEC-007: Access Token Blacklist on Logout
- **File:** `backend/auth/service.go:226-236`
- **Description:** Logout blacklists valid access tokens until natural expiry.
- **Impact:** Short-lived tokens (15 min) mean blacklist is effective only for remaining TTL.
- **Fix:** Acceptable for current design. Consider shorter access TTL for higher security.

---

## 3. Authorization Security

### MEDIUM

#### SEC-008: No Explicit IDOR Protection in Message Endpoints
- **File:** `backend/message/handler.go` (not fully audited - no test files)
- **Description:** Message edit/delete endpoints accept `messageId` path parameter. No visible ownership check in code reviewed.
- **Impact:** Potential IDOR - user could edit/delete another user's messages.
- **Fix:** Add ownership validation in all message mutation endpoints.

#### SEC-009: Federation Disabled by Default (Fail-Closed)
- **File:** `backend/main.go:114-119`
- **Description:** Federation endpoints return 501 unless `FEDERATION_ENABLED=true`.
- **Impact:** Federation is non-functional without explicit configuration.
- **Fix:** This is secure-by-default. Document configuration requirements.

---

## 4. Cryptography Security

### CRITICAL

#### SEC-010: X3DH KDF `info` Mismatch Between Go and TypeScript
- **File:** `backend/internal/crypto/protocol.go:142`
- **File:** `client/src/features/e2ee/pairwiseSession.ts:378`
- **Description:** Go uses `"zenthril.e2ee.x3dh.session.v1:" + userID + ":" + deviceID + "->" + peerUserID + ":" + peerDeviceID`. TypeScript uses `"zenthril.e2ee.x3dh.session.v1|" + JSON.stringify(canonicalHeader(header))`.
- **Impact:** Different keys derived on each side. E2EE decryption will fail. Complete E2EE breakdown.
- **Fix:** Unify `info` string format across both implementations.

#### SEC-011: `deriveX3DHSharedSecret` Is Dead Code
- **File:** `backend/internal/crypto/x25519.go:46-87`
- **Description:** Function not called from production code. Only tested in isolation.
- **Impact:** Production X3DH path bypasses this function. Tests validate unused code.
- **Fix:** Integrate into production path or remove.

### MEDIUM

#### SEC-012: No Safety Numbers UI Verification
- **File:** `client/src/features/e2ee/safetyNumber.ts`
- **Description:** Safety number implementation exists but no UI for comparing key fingerprints between users.
- **Impact:** MITM attacks during initial key exchange undetectable by users.
- **Fix:** Add safety number comparison UI (QR + numeric).

#### SEC-013: Session Restoration Loses DH Private Key
- **File:** `client/src/features/e2ee/pairwiseSession.ts:130-132`
- **Description:** `restorePairwiseSession()` sets `dhSendPrivate` to all zeros. `performDHRatchetTurn()` rejects zero keys.
- **Impact:** After restart, DH ratchet turns impossible. Must re-bootstrap via X3DH.
- **Fix:** Persist `dhSendPrivate` encrypted in session storage.

#### SEC-014: Local Key Storage Not Password-Encrypted
- **File:** `client/src/features/e2ee/` (session storage)
- **Description:** E2EE keys stored in Tauri local storage without user-password-derived encryption.
- **Impact:** Device compromise exposes all keys and message history.
- **Fix:** Encrypt local storage with key derived from user password via Argon2id.

---

## 5. Infrastructure Security

### MEDIUM

#### SEC-015: backend/.env Contains Plaintext Secrets
- **File:** `backend/.env`
- **Description:** `POSTGRES_PASSWORD`, `DB_URL` (with embedded password), `JWT_SECRET` in plaintext.
- **Impact:** Secrets exposed to anyone with file access.
- **Fix:** Remove from repository. Use environment injection or secret manager.

#### SEC-016: Security Log Without Rotation
- **File:** `backend/security/guard.go:182-207`
- **Description:** `security_log` table grows unbounded without retention policy.
- **Impact:** Database bloat, potential DoS via log flooding.
- **Fix:** Add retention/archival for security_log entries.

#### SEC-017: Docker Image Digests May Be Invalid
- **File:** `docker-compose.yml:5,39`
- **Description:** Images pinned to SHA256 digests that may not exist in public registries.
- **Impact:** Deployment may fail if digests are invalid or images removed.
- **Fix:** Verify digests or use versioned tags.

---

## 6. WebSocket Security

### MEDIUM

#### SEC-018: WebSocket Origin Validation
- **File:** `backend/main.go` (wsUpgrader), `backend/internal/gateway/`
- **Description:** WebSocket upgrader uses `wsAllowedOrigins(cfg)` from config. Origins validated.
- **Impact:** Properly secured. No finding.
- **Status:** OK

#### SEC-019: WebSocket Ticket One-Time Use
- **File:** `backend/auth/service.go:401-416`
- **Description:** `ConsumeWSTicket` uses `GetDel` for atomic one-time consumption.
- **Impact:** Properly secured. No finding.
- **Status:** OK

---

## 7. CORS and Headers Security

### OK

#### SEC-020: CORS Strict Allowlist
- **File:** `backend/internal/config/config.go:204-209`
- **Description:** Wildcard origins rejected. Exact origin validation enforced.
- **Impact:** Properly secured.
- **Status:** OK

#### SEC-021: Security Headers Middleware
- **File:** `backend/middleware/` (SecurityHeaders)
- **Description:** CSP, HSTS, frame protection, MIME sniffing, Referrer Policy, Permissions Policy.
- **Impact:** Properly secured.
- **Status:** OK

---

## 8. Dependency Security

### MEDIUM

#### SEC-022: No Secret Scanning in CI
- **File:** `.gitleaks.toml` (untracked, not in CI)
- **Description:** Gitleaks config exists but not tracked in git, not referenced in CI workflows.
- **Impact:** Secrets could be committed and not detected.
- **Fix:** Track `.gitleaks.toml` in git, add to CI workflow.

---

## 9. Summary Table

| ID | Severity | Component | Status |
|----|----------|-----------|--------|
| SEC-001 | CRITICAL | Auth (TOTP) | UNFIXED |
| SEC-002 | HIGH | Auth (rate limiting) | UNFIXED |
| SEC-003 | HIGH | Auth (password reset) | UNFIXED |
| SEC-004 | MEDIUM | Auth (enumeration) | UNFIXED |
| SEC-005 | MEDIUM | Config (JWT length) | UNFIXED |
| SEC-010 | CRITICAL | Crypto (X3DH KDF) | UNFIXED |
| SEC-011 | MEDIUM | Crypto (dead code) | UNFIXED |
| SEC-012 | MEDIUM | Frontend (safety numbers) | UNFIXED |
| SEC-013 | MEDIUM | Frontend (session restore) | UNFIXED |
| SEC-014 | MEDIUM | Frontend (key storage) | UNFIXED |
| SEC-015 | MEDIUM | Infrastructure (secrets) | UNFIXED |
| SEC-016 | MEDIUM | Infrastructure (log rotation) | UNFIXED |
| SEC-017 | MEDIUM | Infrastructure (docker digests) | UNFIXED |
| SEC-022 | MEDIUM | CI (secret scanning) | UNFIXED |

**Total:** 13 findings (2 CRITICAL, 3 HIGH, 8 MEDIUM)

**Critical security defects must be resolved before any production deployment.**
