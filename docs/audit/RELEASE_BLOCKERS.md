# Zenthril Release Blockers Report
**Generated:** 2026-11-02  
**Scope:** All defects that block Release 1.0.0  
**Classification:** INTERNAL - For engineering use only

---

## 1. P0 - Critical Release Blockers (Must Fix Before 1.0.0)

### BLOCKER-001: E2EE Cross-Language Key Derivation Mismatch
- **Severity:** CRITICAL
- **Component:** E2EE / Cryptography
- **Files:** `backend/internal/crypto/protocol.go:142`, `client/src/features/e2ee/pairwiseSession.ts:378`
- **Description:** Go `StartSession()` uses `sessionInfo = "zenthril.e2ee.x3dh.session.v1:" + userID + ":" + deviceID + "->" + peerUserID + ":" + peerDeviceID`. TypeScript `deriveSessionState()` uses `info = "zenthril.e2ee.x3dh.session.v1|" + JSON.stringify(canonicalHeader(header))`. Different HKDF `info` inputs produce different root/send/recv keys.
- **Impact:** E2EE messages cannot be decrypted between client and server. Core product feature is non-functional.
- **Reproduction:** Initiate X3DH session from Go and TypeScript with identical DH inputs; compare resulting root keys.
- **Fix Required:** Unify `info` string format. Both sides must use identical input to HKDF.
- **Verification:** Cross-language integration test with deterministic DH values proving key equality.

### BLOCKER-002: TOTP/MFA Verification Non-Functional
- **Severity:** CRITICAL
- **Component:** Authentication / MFA
- **Files:** `backend/auth/service.go:452`, `backend/auth/service.go:483`
- **Description:** `SetupTOTP()` stores `hashToken(secret)` = SHA-256 hex of base32 secret. `VerifyTOTPSetup()` calls `base32Decode(secretHash)` attempting to decode SHA-256 hex as base32, which always returns `ErrInvalidTOTPSecret`.
- **Impact:** TOTP setup cannot complete. MFA cannot be enabled. Two-factor authentication feature is broken.
- **Reproduction:** Call `SetupTOTP()`, then `VerifyTOTPSetup()` with valid 6-digit code. Verify returns error.
- **Fix Required:** Store raw base32-encoded TOTP secret (not its SHA-256 hash). Decode it in verification.
- **Verification:** End-to-end test: SetupTOTP → VerifyTOTPSetup → VerifyTOTP all succeed with valid code.

### BLOCKER-003: No Rate Limiting on TOTP Endpoints
- **Severity:** HIGH (blocker due to security)
- **Component:** Authentication / Rate Limiting
- **Files:** `backend/main.go:284-286`
- **Description:** Routes `/mfa/totp/start`, `/mfa/totp/confirm`, `/mfa/totp/disable` have no `AuthRateLimit` or `BruteForceProtect` middleware.
- **Impact:** 6-digit TOTP codes (1,000,000 combinations) can be brute-forced without restriction.
- **Fix Required:** Apply `AuthRateLimit("totp_confirm", 5, time.Minute)` to TOTP routes.
- **Verification:** Test that >5 TOTP confirm attempts per minute returns 429.

### BLOCKER-004: No Password Reset / Account Recovery
- **Severity:** HIGH
- **Component:** Authentication / Account Management
- **Files:** `backend/auth/handler.go`, `backend/auth/service.go`
- **Description:** No endpoints for password reset, account recovery, or MFA reset on lost device.
- **Impact:** Users who forget password or lose MFA device are permanently locked out. No recovery path exists.
- **Fix Required:** Implement `POST /api/v1/auth/password-reset/request` and `POST /api/v1/auth/password-reset/confirm` with time-limited tokens.
- **Verification:** Password reset flow test with token expiry.

### BLOCKER-005: backend-2.0 Is Empty Skeleton
- **Severity:** HIGH
- **Component:** Architecture
- **Files:** `backend-2.0/` (all directories empty except logger.go, errors.go)
- **Description:** Promised next-generation backend with clean architecture is not implemented. Only 2 Go files exist out of planned structure.
- **Impact:** Product ships with incomplete architecture. Maintenance burden of maintaining two codebases.
- **Fix Required:** Either complete backend-2.0 implementation or consolidate into backend 1.x.
- **Verification:** Backend-2.0 compiles and handles basic requests (if completed) OR backend-2.0 removed and consolidated.

### BLOCKER-006: ZenthrilDB Not Integrated
- **Severity:** HIGH
- **Component:** Data Layer
- **Files:** `zenthrildb-push/` (untracked C++ code)
- **Description:** ZenthrilDB C++ code exists but has no Go bindings, no API integration, no connection to main backend.
- **Impact:** Promised ZenthrilDB integration is non-functional. Product ships without claimed database feature.
- **Fix Required:** Create Go CGo bindings to ZenthrilDB OR remove ZenthrilDB dependency and documentation.
- **Verification:** Go code can query ZenthrilDB through bindings (if integrated).

---

## 2. P1 - High Priority Release Blockers

### BLOCKER-007: Missing Tests for guild, message, cmd Packages
- **Severity:** HIGH
- **Component:** Testing
- **Files:** `backend/guild/` (no tests), `backend/message/` (no tests), `backend/cmd/` (no tests)
- **Description:** 15+ Go packages have zero unit tests. Core business logic untested.
- **Impact:** Regressions undetected. Cannot guarantee correctness of guild/message operations.
- **Fix Required:** Add unit tests for all untested packages.
- **Verification:** All packages have test files, `go test ./...` passes with coverage > 60%.

### BLOCKER-008: No Race Condition Tests for RatchetState
- **Severity:** HIGH
- **Component:** Cryptography / Concurrency
- **Files:** `backend/internal/crypto/ratchet.go`
- **Description:** `RatchetState` is mutable and modified without mutex protection. Race detector not supported on Windows (`windows/386`).
- **Impact:** Data races in concurrent message sending could corrupt session state.
- **Fix Required:** Add `sync.RWMutex` to `RatchetState`. Add concurrent access tests. Run `-race` on Linux CI.
- **Verification:** `go test -race ./internal/crypto/...` passes on Linux.

### BLOCKER-009: No Safety Numbers UI Verification
- **Severity:** MEDIUM (security-relevant)
- **Component:** Frontend / E2EE
- **Files:** `client/src/features/e2ee/safetyNumber.ts`
- **Description:** Safety number implementation exists but no UI for users to compare key fingerprints.
- **Impact:** MITM attacks during initial key exchange undetectable by users.
- **Fix Required:** Add safety number comparison UI with QR code and numeric display.
- **Verification:** User can view and compare safety numbers in device management screen.

### BLOCKER-010: Local Key Storage Not Password-Encrypted
- **Severity:** MEDIUM
- **Component:** Frontend / Key Storage
- **Files:** `client/src/features/e2ee/` (session storage)
- **Description:** E2EE keys stored in Tauri local storage without encryption derived from user password.
- **Impact:** Device compromise exposes all E2EE keys and message history.
- **Fix Required:** Encrypt local storage with key derived from user password via Argon2id.
- **Verification:** Keys are encrypted at rest, require password to decrypt.

---

## 3. P2 - Medium Priority (Should Fix Before 1.0.0)

### BLOCKER-011: Documentation Not Synchronized with Code
- **Severity:** LOW
- **Component:** Documentation
- **Files:** `backend/internal/crypto/test_vectors_test.go:562-575`
- **Description:** Test comments claim TypeScript uses different `info` string format than actual implementation.
- **Impact:** Developers draw incorrect conclusions about cross-language compatibility.
- **Fix Required:** Update test comments to match actual TypeScript implementation.
- **Verification:** Documentation accurately describes code behavior.

### BLOCKER-012: Security Log Without Rotation
- **Severity:** LOW
- **Component:** Infrastructure
- **Files:** `backend/security/guard.go:182-207`
- **Description:** `security_log` table grows unbounded without retention policy.
- **Impact:** Database bloat over time.
- **Fix Required:** Add retention policy for security_log entries.
- **Verification:** Old security log entries are archived or deleted.

### BLOCKER-013: backend/.env Contains Plaintext Secrets
- **Severity:** MEDIUM
- **Component:** Infrastructure / Secrets
- **Files:** `backend/.env`
- **Description:** `POSTGRES_PASSWORD`, `DB_URL` (with password), `JWT_SECRET` in plaintext.
- **Impact:** Secrets exposed if repository is cloned or file accessed.
- **Fix Required:** Remove `.env` from repository. Use environment injection only.
- **Verification:** `.env` not tracked by git. CI does not contain hardcoded secrets.

---

## 4. Release Blocker Summary Table

| ID | Severity | Component | Status | Fix Effort |
|-----|----------|-----------|--------|------------|
| BLOCKER-001 | CRITICAL | E2EE KDF | UNFIXED | Medium |
| BLOCKER-002 | CRITICAL | TOTP/MFA | UNFIXED | Medium |
| BLOCKER-003 | HIGH | Rate Limiting | UNFIXED | Small |
| BLOCKER-004 | HIGH | Password Reset | UNFIXED | Medium |
| BLOCKER-005 | HIGH | backend-2.0 | UNFIXED | Large |
| BLOCKER-006 | HIGH | ZenthrilDB | UNFIXED | Large |
| BLOCKER-007 | HIGH | Testing | UNFIXED | Medium |
| BLOCKER-008 | HIGH | Race Conditions | UNFIXED | Medium |
| BLOCKER-009 | MEDIUM | Safety Numbers | UNFIXED | Medium |
| BLOCKER-010 | MEDIUM | Key Storage | UNFIXED | Medium |
| BLOCKER-011 | LOW | Documentation | UNFIXED | Small |
| BLOCKER-012 | LOW | Log Rotation | UNFIXED | Small |
| BLOCKER-013 | MEDIUM | Secrets | UNFIXED | Small |

**Total Blockers: 13 (2 CRITICAL, 6 HIGH, 5 MEDIUM)**

---

## 5. Definition of Done for Release 1.0.0

All of the following must be true:

- [ ] BLOCKER-001 through BLOCKER-008 resolved
- [ ] `go test ./...` passes for all Go packages
- [ ] `npm test` passes for all TypeScript tests
- [ ] `npm run lint` passes with zero errors
- [ ] `npx tsc --noEmit` passes with zero errors
- [ ] Cross-language E2EE integration test passes
- [ ] TOTP end-to-end test passes
- [ ] Password reset flow implemented and tested
- [ ] Rate limiting on TOTP endpoints verified
- [ ] backend-2.0 completed or consolidated
- [ ] ZenthrilDB integrated or removed
- [ ] All critical/high security defects fixed
- [ ] Documentation synchronized with code
- [ ] Release notes prepared
- [ ] GitHub repository updated

---
