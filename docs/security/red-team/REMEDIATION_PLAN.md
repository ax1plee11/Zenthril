# Zenthril Red Team - Remediation Plan
**Generated:** 2026-11-02  
**Classification:** INTERNAL - Security Team Only

---

## 1. Remediation Priority

| Priority | Level | Count | Action Timeline |
|----------|-------|-------|-----------------|
| P0 | Critical | 1 | Fix immediately (VULN-001) |
| P1 | High | 3 | Fix before release (VULN-002, VULN-003, VULN-005) |
| P2 | Medium | 2 | Fix before release (VULN-004, VULN-006) |

---

## 2. Remediation Items

### P0 - Critical

#### VULN-001: Password Reset Token in Response Body
- **Fix:** Remove `reset_token` from `RequestPasswordReset` response. Return `{"status":"ok"}` only.
- **Files:** `backend/auth/handler.go`
- **Effort:** Small (~30 minutes)
- **Dependencies:** None
- **Status:** FIXED

### P1 - High

#### VULN-002: TOTP Secret Not Encrypted at Rest
- **Fix:** Encrypt TOTP secret using AES-256-GCM with server-side key before storing in `secret_enc` column. Decrypt on verification.
- **Files:** `backend/auth/service.go`, `backend/migrations/` (may need column rename or new column)
- **Effort:** Large (~1-2 days)
- **Dependencies:** Key management infrastructure (server-side encryption key)
- **Status:** OPEN

#### VULN-003: No MFA Recovery Codes
- **Fix:** Generate recovery codes during TOTP setup, store hashed, allow verification for MFA disable/reset.
- **Files:** `backend/auth/service.go`, `backend/auth/handler.go`, `backend/migrations/`
- **Effort:** Medium (~4-6 hours)
- **Dependencies:** None
- **Status:** OPEN

#### VULN-005: Backend-2.0 Empty Skeleton
- **Fix:** Either complete backend-2.0 implementation with clean architecture OR consolidate into backend 1.x and remove backend-2.0.
- **Files:** `backend-2.0/` directory
- **Effort:** Large (weeks)
- **Dependencies:** Architecture decision
- **Status:** OPEN

### P2 - Medium

#### VULN-004: Security Log Without Rotation
- **Fix:** Add retention policy (delete entries older than 90 days) or archival mechanism.
- **Files:** `backend/security/guard.go`, `backend/migrations/`
- **Effort:** Small (~2 hours)
- **Dependencies:** None
- **Status:** OPEN

#### VULN-006: ZenthrilDB Not Integrated
- **Fix:** Create Go CGo bindings to ZenthrilDB C++ code OR remove ZenthrilDB dependency and documentation.
- **Files:** `zenthrildb-push/`, backend integration code
- **Effort:** Large (days to weeks)
- **Dependencies:** C++ code review, Go CGo expertise
- **Status:** OPEN

---

## 3. Remediation Execution Order

1. **VULN-001** (CRITICAL) - Fix password reset token exposure (DONE)
2. **VULN-003** (HIGH) - Add MFA recovery codes
3. **VULN-002** (HIGH) - Encrypt TOTP secrets at rest
4. **VULN-004** (MEDIUM) - Add security log retention
5. **VULN-006** (MEDIUM) - Integrate or remove ZenthrilDB
6. **VULN-005** (HIGH) - Resolve backend-2.0 architectural debt

---

## 4. Verification Criteria

For each fix:
- [ ] Build succeeds (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Regression test added and passes
- [ ] Original attack scenario no longer works
- [ ] No new vulnerabilities introduced

---
