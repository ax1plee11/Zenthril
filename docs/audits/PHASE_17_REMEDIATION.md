# PHASE 17 REMEDIATION AUDIT REPORT

**Date:** 2026-10-17  
**Repository:** Zenthril  
**Branch:** `main`  
**HEAD:** `2476e9b`  
**Auditor:** Kilo (lead software engineer / security specialist)

---

## 1. Initial repository state

Zenthril — alpha-stage self-hosted messenger:
- **Backend:** Go 1.26.5, Chi router, PostgreSQL, Redis, gorilla/websocket
- **Frontend:** Tauri 2, React, TypeScript, Tailwind CSS, vitest
- **Research:** ZenthrilDB (C++23, separate, untracked, `.obj` artifacts present)

Git status at start: branch `main`, uncommitted user changes in
`pairwiseSession.security.test.ts`, `pairwiseSession.trace.test.ts`, `docker-compose.yml`.
Untracked: `.kilo/`, `INTEGRATION_ANALYSIS.md`, `Zenthril_ZenthrilDB_Technical_Analysis.docx`,
`client/src/features/e2ee/doubleRatchetDesign.test.ts`, `generate_dissertation.py`, `zenthrildb-push/`.

All baseline checks green: `go build`, `go vet`, `golangci-lint` (0 issues), `go test ./...`
(18 packages `ok`), frontend lint (0), tsc (0), vitest (29 files / 185 tests).

## 2. Discovered defects (with IDs and priorities)

| ID | Priority | Component | Summary | Status |
|---|---|---|---|---|
| CRY-101 / F-09 | CRITICAL | E2EE client ↔ Go | Client and Go ratchet use different salt/info/layout and session init material; cross-implementation E2EE does not work | Blocked — owner decision required |
| PKG-201 | HIGH | `moderation` | Package had zero tests; access-control logic unguarded | FIXED — tests added |
| MFA-203 | HIGH | auth | MFA handlers return 501; not registered in production | Open — implementation or doc removal |
| PROD-204 | HIGH | product | No direct messages, no attachments | Open — product design required |
| DEV-301 | MEDIUM | devices | Per-user device cap missing | FIXED — `MaxDevicesPerUser = 10` with `FOR UPDATE` |
| WS-302 | MEDIUM | WebSocket | Ticket not bound to IP/User-Agent | Open — threat-model analysis needed |
| CI-303 | MEDIUM | CI | Trivy `continue-on-error: true` — not a real gate | FIXED — gate enforced |
| RACE-304 | MEDIUM | env | Race detector unavailable locally (windows/386, no cgo) | Open — CI covers |
| REPO-305 | MEDIUM | frontend | Uncommitted WIP test with lint errors and disabled `expect` | FIXED — test passes lint + vitest |
| LOW-401 | LOW | `internal/crypto` | `panic()` in test helper | FIXED — `t.Fatalf` |
| LOW-403 | LOW | migrations | No migration tests | FIXED — schema validation tests added |
| CRY-202 | LOW | crypto | No official test vectors (RFC/NIST) | FIXED — vectors + negative AES-GCM tests |
| DOC-306 | LOW | docs | ZenthrilDB README referenced nonexistent docs, omitted implemented subsystems | FIXED — README rewritten |
| BENCH-307 | LOW | research | Benchmark numbers unverified, µs vs ms mismatch, "production-ready" claim contradicts Alpha status | FIXED — marked unverified/historical, units corrected |
| LOW-402 | LOW | — | No fuzz tests | Open |

## 3. Fixes and changed files

### LOW-401 — `panic()` → `t.Fatalf`
- `backend/internal/crypto/test_vectors_test.go` — `mustDecodeHex` now takes `*testing.T` and calls `t.Fatalf` instead of `panic`. All 18 call sites updated to pass `t`.

### PKG-201 — moderation tests
- `backend/moderation/moderation_test.go` (new) — 9 unit tests covering `GuildStateManager` (trust, lock/unlock, owner join/leave, status), `ConnectionMonitor` (`ExtractRealIP` header parsing, `isValidIP` private/loopback rejection), UUID validation helpers.

### LOW-403 — migration tests
- `backend/migrations_test/migration_test.go` (new) — validates 15 migrations are numbered sequentially (001–015), non-empty, contain SQL statements; live DB application skipped without `ZENTHRIL_TEST_DB_DSN`.

### CRY-202 — crypto test vectors + negative tests
- `backend/internal/crypto/test_vectors_test.go` — added `TestAESGCMRejectsWrongKey`, `TestAESGCMRejectsTamperedCiphertext`, `TestAESGCMEncryptDecryptEmptyPlaintext`. Existing vectors: RFC 7748 X25519 (2), RFC 5869 HKDF, RFC 8032 Ed25519, root ratchet, X3DH shared secret.

### CI-303 — Trivy gate
- `.github/workflows/ci.yml` — removed `continue-on-error: true` from Trivy step so CRITICAL/HIGH findings fail the job.

### DOC-306 — ZenthrilDB README
- `zenthrildb-push/README.md` — rewritten: marks component as research/not integrated, lists implemented subsystems (WAL, recovery, transaction, lock), clarifies PostgreSQL is the Go backend DB, no integration claimed.

### BENCH-307 — benchmark numbers
- `research/results/RESEARCH_REPORT.md` — added UNVERIFIED/HISTORICAL banner, corrected 0.83ms → 0.83 µs, removed "production-ready" claim, softened concurrency claims.

### Pre-existing (already fixed in recent commits)
- `backend/friends/service.go` — `parseFriendshipIDs` rejects malformed IDs; `backend/friends/service_test.go` added.
- `backend/device/service.go` — `MaxDevicesPerUser = 10` with `FOR UPDATE` row lock; `backend/device/service_test.go` (5 tests, BOLA fix).
- `backend/middleware/cors.go` — comma-split header fix; `backend/middleware/cors_test.go` (6 tests).
- `backend/security/guard.go` — rate limit on `/register`, `/refresh`, `/logout`; `guard_ratelimit_test.go` (3 tests).
- Client E2EE fixes: IV from dedicated nonce output, functional chain key, counter validation, DH key copy.

## 4. Added tests

- `backend/moderation/moderation_test.go` — 9 tests
- `backend/migrations_test/migration_test.go` — 4 tests (1 skipped without DB)
- `backend/internal/crypto/test_vectors_test.go` — 3 negative AES-GCM tests
- Existing: friends (3), device (5), cors (6), guard (3), crypto vectors

## 5. Commands and actual results

```
go build ./...                    -> exit 0
go vet ./...                      -> exit 0
golangci-lint run                 -> 0 issues, exit 0
go test ./...                     -> exit 0, 18 packages ok
go test -race ./...               -> BLOCKED locally (windows/386, no cgo); covered by CI
npm run lint (client)             -> exit 0
npx tsc --noEmit (client)         -> exit 0
npm test (client)                 -> 32 files / 208 tests passed
```

Race detector: CI runs `go test -race` on `ubuntu-latest` (`ci.yml:74`).

## 6. Remaining critical and high issues

- **CRY-101** — client ↔ Go cryptographic incompatibility. Requires owner decision on canonical schema before any protocol change. Do not modify ratchet code without approval.
- **MFA-203** — MFA advertised but stubbed (501). Either implement TOTP fully or remove from docs/routes.
- **PROD-204** — no DMs or attachments; blocks any real messaging scenario.
- **WS-302** — WebSocket ticket not bound to IP/User-Agent; requires threat-model analysis before binding (mobile/proxy considerations).

## 7. Changes requiring owner decision

- CRY-101: select canonical ratchet schema (recommend Go as reference since server holds session state), add cross-implementation test vector, version sessions for migration.
- MFA-203: implement TOTP or remove endpoints/docs.
- PROD-204 / PROD-205: design DMs, attachments, avatar upload endpoint.
- ZenthrilDB integration: do not connect automatically; separate architectural decision needed.

## 8. Unimplemented product functions

- Direct messages (DM) — no routes
- Attachments / file upload — no routes, no path-traversal protection
- Avatar server storage — currently local base64 only; `QuotaExceededError` risk
- MFA / TOTP — stubbed handlers
- Federation — scaffolded, disabled by default, no tests

## 9. Compatibility and migration risks

- Changing ratchet derivation (CRY-101) breaks all existing sessions; requires versioning and migration path.
- ZenthrilDB is not a PostgreSQL replacement; any adoption requires full data migration and audit.
- Device cap (MaxDevicesPerUser = 10) may affect existing users with >10 active devices.

## 10. Real readiness status

**Alpha / not production-ready.** Automated checks pass, but:
- E2EE cross-implementation compatibility is broken (CRY-101)
- MFA, DMs, attachments missing
- 15 of 30 Go packages lacked tests (now moderation covered; friends covered)
- No external security audit
- Benchmark numbers unverified

Do not use for sensitive or high-risk communication.

---

**End of report.**
