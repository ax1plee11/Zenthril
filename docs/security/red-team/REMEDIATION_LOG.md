# Zenthril Red Team - Remediation Log
**Generated:** 2026-11-02  
**Classification:** INTERNAL - Security Team Only

---

## Changelog of Fixes

### 2026-11-02 | VULN-001: Password Reset Token Exposed in HTTP Response Body

**Status:** FIXED & VERIFIED

**Changes:**
- `backend/auth/handler.go`: `RequestPasswordReset` handler no longer returns `reset_token` in the JSON response body. Response is now `{"status":"ok"}` only.
- `backend/auth/service.go`: Updated `RequestPasswordReset` doc comment to clarify token is NOT returned to caller.
- `backend/auth/handler_test.go`: Handler test file reviewed; existing tests pass.

**Root Cause:** The handler returned the plaintext password reset token in the HTTP response body, enabling account takeover via token interception.

**Fix:** Token is generated and stored in the `password_reset_tokens` table internally, but the handler discards it and returns only `{"status":"ok"}`. Token delivery must be implemented out-of-band (email, SMS) by the caller.

**Verification:**
- `go build ./...` - PASS
- `go vet ./...` - CLEAN
- `go test ./auth/...` - PASS
- `go test ./...` - ALL 28 PACKAGES PASS
- Code review confirms `reset_token` key is absent from response map

**Attack Re-test:** Token no longer present in `POST /api/v1/auth/password-reset/request` response body.

---

### Remaining Open Vulnerabilities

| ID | Severity | Status | Required Action |
|----|----------|--------|-----------------|
| VULN-002 | HIGH | OPEN | Encrypt TOTP secrets at rest (AES-256-GCM) |
| VULN-003 | HIGH | OPEN | Add MFA recovery codes for lost devices |
| VULN-004 | MEDIUM | OPEN | Add security log retention policy |
| VULN-005 | HIGH | OPEN | Complete or consolidate backend-2.0 |
| VULN-006 | MEDIUM | OPEN | Integrate ZenthrilDB or remove dependency |

---
