# Zenthril Technical Audit Report
**Generated:** 2026-11-02  
**Scope:** Full codebase audit (backend, client, infrastructure)  
**Classification:** INTERNAL - For engineering use only

---

## 1. Executive Summary

Zenthril is an Alpha-stage self-hosted secure messaging platform. The codebase contains a working Go backend (Chi router, PostgreSQL, Redis), a Tauri 2 + React + TypeScript desktop client, and E2EE foundations (X3DH + Double Ratchet). However, several **critical defects** prevent production use:

1. **E2EE interoperability is broken** - Go and TypeScript X3DH KDF `info` strings do not match in production code paths
2. **TOTP/MFA is non-functional** - Setup stores SHA-256 hash of secret, verification tries base32-decoding the hash
3. **No password reset / account recovery** - Users who lose access are permanently locked out
4. **backend-2.0 is an empty skeleton** - Only 2 Go files, no implementation

**Overall readiness:** Alpha / Not production ready.

---

## 2. Backend Technical Findings

### CRITICAL

#### TEC-001: X3DH KDF `info` String Mismatch (Go vs TypeScript)
- **File:** `backend/internal/crypto/protocol.go:142`
- **File:** `client/src/features/e2ee/pairwiseSession.ts:378`
- **Description:** `StartSession()` in Go constructs `sessionInfo` as `"zenthril.e2ee.x3dh.session.v1:" + local.UserID + ":" + local.DeviceID + "->" + peer.UserID + ":" + peer.DeviceID`. TypeScript `deriveSessionState()` uses `"zenthril.e2ee.x3dh.session.v1|" + JSON.stringify(canonicalHeader(header))`. These produce different HKDF inputs, resulting in different root/send/recv keys.
- **Impact:** E2EE messages encrypted by one side cannot be decrypted by the other. End-to-end encryption is non-functional between client and server.
- **Reproduction:** Call `StartSession()` in Go and `initiatePairwiseSession()`/`acceptPairwiseSession()` in TypeScript with identical DH inputs; compare resulting root keys.
- **Fix:** Unify the `info` string format. Either both use JSON canonical header, or both use the simple colon-separated format.

#### TEC-002: `deriveX3DHSharedSecret` is Dead Code
- **File:** `backend/internal/crypto/x25519.go:46-87`
- **Description:** The function is only called from tests (`protocol_test.go:261`, `test_vectors_test.go:253`). Production code in `protocol.go:StartSession()` computes DH values inline and does not use this function.
- **Impact:** Tests validate code that is not used in production. False sense of security.
- **Fix:** Either integrate `deriveX3DHSharedSecret` into `StartSession()` or remove it.

### HIGH

#### TEC-003: TOTP Secret Storage Mismatch
- **File:** `backend/auth/service.go:452` (SetupTOTP) and `:483` (VerifyTOTPSetup)
- **Description:** `SetupTOTP` stores `hashToken(secret)` (SHA-256 hex of base32 secret) in `secret_enc` column. `VerifyTOTPSetup` calls `base32Decode(secretHash)` attempting to decode SHA-256 hex as base32, which always fails with `ErrInvalidTOTPSecret`.
- **Impact:** TOTP setup can never complete. MFA cannot be enabled.
- **Fix:** Store the raw base32-encoded TOTP secret (encrypted at rest if required), not its SHA-256 hash.

#### TEC-004: No Rate Limiting on TOTP Endpoints
- **File:** `backend/main.go:284-286`
- **Description:** Routes `/mfa/totp/start`, `/mfa/totp/confirm`, `/mfa/totp/disable` are registered without `AuthRateLimit` or `BruteForceProtect` middleware.
- **Impact:** Attacker can brute-force 6-digit TOTP codes (1,000,000 combinations).
- **Fix:** Apply `secGuard.AuthRateLimit("totp_confirm", 5, time.Minute)` to TOTP endpoints.

#### TEC-005: No Password Reset / Account Recovery
- **File:** `backend/auth/handler.go`, `backend/auth/service.go`
- **Description:** No endpoints exist for password reset, account recovery, or MFA reset when device is lost.
- **Impact:** Users who forget password or lose MFA device are permanently locked out.
- **Fix:** Implement password reset flow with time-limited reset tokens.

#### TEC-006: backend-2.0 Empty Skeleton
- **Files:** `backend-2.0/cmd/api/`, `backend-2.0/cmd/worker/`, `backend-2.0/internal/app/`, `backend-2.0/internal/config/`, `backend-2.0/internal/domains/`, `backend-2.0/internal/infrastructure/`, `backend-2.0/internal/platform/` (except logger.go), `backend-2.0/pkg/crypto/`
- **Description:** All directories are empty or contain only stub files. `go.mod` exists but no implementation.
- **Impact:** Promised next-generation backend architecture does not exist.
- **Fix:** Either complete backend-2.0 implementation or remove it and consolidate into backend 1.x.

### MEDIUM

#### TEC-007: Missing Tests for Critical Packages
- **Files:** `backend/guild/*_test.go` (missing), `backend/message/*_test.go` (missing), `backend/cmd/*_test.go` (missing), `backend/db/*_test.go` (missing), `backend/internal/domain/*_test.go` (missing)
- **Description:** 15+ Go packages have no unit tests.
- **Impact:** Regressions in guild/message/cmd logic will not be caught.
- **Fix:** Add unit tests for all untested packages.

#### TEC-008: Race Condition Testing Blocked on Windows
- **File:** `backend/internal/crypto/ratchet.go`
- **Description:** `RatchetState` is mutable and modified without mutexes. Race detector (`go test -race`) is not supported on `windows/386`.
- **Impact:** Potential data races in concurrent message sending not detectable locally.
- **Fix:** Add mutex protection to RatchetState, run race tests on Linux CI.

#### TEC-009: ZenthrilDB Not Integrated
- **File:** `zenthrildb-push/` (untracked C++ code)
- **Description:** C++ database code exists but has no Go bindings, no API integration.
- **Impact:** Promised ZenthrilDB integration is non-functional.
- **Fix:** Create Go CGo bindings or remove ZenthrilDB dependency.

### LOW

#### TEC-010: Documentation Not Synchronized with Code
- **File:** `backend/internal/crypto/test_vectors_test.go:562-575`
- **Description:** Test comment claims TypeScript uses `"zenthril.e2ee.x3dh.session.v1:" + sessionInfo` but actual TypeScript uses `"zenthril.e2ee.x3dh.session.v1|" + JSON.stringify(...)`.
- **Impact:** Developers reading tests will draw incorrect conclusions about cross-language compatibility.
- **Fix:** Update test comments to match actual implementation.

#### TEC-011: Security Log Without Rotation
- **File:** `backend/security/guard.go:182-207`
- **Description:** `LogSecurityEvent` writes to PostgreSQL `security_log` table without retention policy.
- **Impact:** Table grows unbounded, affecting long-term performance.
- **Fix:** Add retention policy or archival mechanism.

---

## 3. Frontend Technical Findings

### MEDIUM

#### TEC-012: No Safety Numbers UI Verification
- **File:** `client/src/features/e2ee/safetyNumber.ts`
- **Description:** Safety number implementation exists but no UI flow for comparing safety numbers between users.
- **Impact:** MITM attacks during key exchange are not detectable by users.
- **Fix:** Add safety number comparison UI with QR code and numeric comparison.

#### TEC-013: No Secure Local Key Storage Encryption
- **File:** `client/src/features/e2ee/` (session storage)
- **Description:** Pairwise session state and device keys are stored in Tauri local storage without additional encryption keyed to user password.
- **Impact:** Device compromise exposes all E2EE keys and message history.
- **Fix:** Encrypt local storage with key derived from user password.

#### TEC-014: Session Restoration Loses DH Private Key
- **File:** `client/src/features/e2ee/pairwiseSession.ts:130-132`
- **Description:** `restorePairwiseSession()` initializes `dhSendPrivate` as `new Uint8Array(32)` (all zeros). `performDHRatchetTurn()` rejects zero DH private keys, forcing re-bootstrap via X3DH.
- **Impact:** After app restart, sessions cannot perform DH ratchet turns; must re-initiate via X3DH.
- **Fix:** Persist `dhSendPrivate` in serialized session state (encrypted at rest).

---

## 4. Infrastructure Technical Findings

### MEDIUM

#### TEC-015: Docker Compose Uses Hardcoded Image Digests
- **File:** `docker-compose.yml:5,39`
- **Description:** Images pinned to specific SHA256 digests that may not exist in public registries.
- **Impact:** `docker compose up` may fail if digests are invalid.
- **Fix:** Verify digests or use version tags.

#### TEC-016: Backend .env Contains Plaintext Secrets
- **File:** `backend/.env`
- **Description:** `POSTGRES_PASSWORD`, `DB_URL` with password, and `JWT_SECRET` stored in plaintext.
- **Impact:** Secrets exposed if file is accessed.
- **Fix:** Remove from version control, use secret management or environment injection only.

#### TEC-017: No Backup/Restore Procedures Documented
- **Files:** `docker-compose.yml` (volumes only), no backup scripts
- **Description:** PostgreSQL and Redis volumes exist but no automated backup/restore procedures.
- **Impact:** Data loss on failure with no recovery path.
- **Fix:** Add backup scripts and restore documentation.

---

## 5. Cryptography Technical Findings

### CRITICAL

#### CRY-001: X3DH Cross-Language Incompatibility
- See TEC-001 above.

### MEDIUM

#### CRY-002: No Safety Numbers in UI
- See TEC-012 above.

#### CRY-003: Session State Persists Zeroed DH Private Key
- See TEC-014 above.

---

## 6. Summary Table

| ID | Severity | Component | Status |
|----|----------|-----------|--------|
| TEC-001 | CRITICAL | E2EE (Go-TS KDF) | UNFIXED |
| TEC-002 | CRITICAL | E2EE (dead code) | UNFIXED |
| TEC-003 | CRITICAL | Auth (TOTP) | UNFIXED |
| TEC-004 | HIGH | Auth (rate limiting) | UNFIXED |
| TEC-005 | HIGH | Auth (password reset) | UNFIXED |
| TEC-006 | HIGH | Architecture (backend-2.0) | UNFIXED |
| TEC-007 | MEDIUM | Testing (missing tests) | UNFIXED |
| TEC-008 | MEDIUM | Crypto (race conditions) | UNFIXED |
| TEC-009 | MEDIUM | Data (ZenthrilDB) | UNFIXED |
| TEC-010 | LOW | Docs (sync) | UNFIXED |
| TEC-011 | LOW | Infrastructure (log rotation) | UNFIXED |
| TEC-012 | MEDIUM | Frontend (safety numbers) | UNFIXED |
| TEC-013 | MEDIUM | Frontend (key storage) | UNFIXED |
| TEC-014 | MEDIUM | Frontend (session restore) | UNFIXED |
| TEC-015 | MEDIUM | Infrastructure (docker digests) | UNFIXED |
| TEC-016 | MEDIUM | Infrastructure (secrets) | UNFIXED |
| TEC-017 | MEDIUM | Infrastructure (backup) | UNFIXED |

**Total:** 17 findings (3 CRITICAL, 5 HIGH, 8 MEDIUM, 1 LOW)
