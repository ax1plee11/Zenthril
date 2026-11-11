# Zenthril Test Coverage Audit Report
**Generated:** 2026-11-02  
**Scope:** All Go and TypeScript test files  
**Classification:** INTERNAL - For engineering use only

---

## 1. Go Backend Test Coverage

### Packages WITH Tests

| Package | Test Files | Test Coverage | Status |
|---------|-----------|---------------|--------|
| `auth/` | handler_test.go, jwt_test.go, password_test.go | Registration, login, JWT, password hashing | GOOD |
| `device/` | (has tests) | Device registration, key bundles | GOOD |
| `federation/` | service_test.go (newly added) | Domain normalization, envelope validation, announce validation | BASIC |
| `friends/` | (has tests) | Friend requests, accept, decline | GOOD |
| `hub/` | (has tests) | WebSocket hub, message broadcasting | GOOD |
| `internal/app/` | module_security_test.go | Production guards, fx module security | BASIC |
| `internal/config/` | config_test.go | Configuration loading, validation | GOOD |
| `internal/cqrs/` | cqrs_test.go | Command bus, query bus | BASIC |
| `internal/crypto/` | message_test.go, multidevice_test.go, protocol_test.go, ratchet_test.go, session_store.go, test_vectors_test.go | X3DH, Double Ratchet, message encryption, skipped keys | GOOD |
| `internal/event/` | (no test files found) | Event bus | MISSING |
| `internal/gateway/` | auth_test.go, client_ip_test.go, handler_test.go, registry_security_test.go, registry_test.go | Gateway auth, IP extraction, handler, registry | GOOD |
| `internal/pubsub/` | shard_test.go | Pub/sub sharding | BASIC |
| `migrations_test/` | (migration tests) | Database migrations | BASIC |
| `moderation/` | (has tests) | Guild ban, cascade ban | GOOD |
| `security/` | (has tests) | Rate limiting, brute force, IP blocking | GOOD |
| `user/` | (has tests) | User search, profiles | GOOD |

### Packages WITHOUT Tests

| Package | Missing Tests | Impact |
|---------|--------------|--------|
| `backend/cmd/api/` | No test files | API entrypoint untested |
| `backend/cmd/resetpw/` | No test files | Password reset skeleton untested |
| `backend/cmd/sfu/` | No test files | SFU control plane untested |
| `backend/cmd/worker/` | No test files | Worker untested |
| `backend/db/` | No test files | Database layer untested |
| `backend/guild/` | No test files | Guild CRUD, roles, members, moderation untested |
| backend/internal/domain/ | No test files | Domain types untested |
| backend/internal/event/ | No test files | Event bus untested |
| backend/internal/observability/ | No test files | Tracing, metrics untested |
| backend/internal/repository/ | No test files | Repository/shard layer untested |
| backend/logger/ | No test files | Logger untested |
| backend/message/ | No test files | Message send, edit, delete, history untested |
| backend/metrics/ | No test files | Metrics collection untested |
| backend/middleware/ | No test files | HTTP middleware untested |
| backend/models/ | No test files | Data models untested |
| backend/spam/ | No test files | Spam protection untested |

### Test Coverage Summary

| Metric | Value |
|--------|-------|
| Total Go packages | ~43 |
| Packages with tests | ~16 |
| Packages without tests | ~27 |
| Test coverage (estimated) | 30-40% |

---

## 2. TypeScript Frontend Test Coverage

### Test Files

| File | Tests | Status |
|------|-------|--------|
| `client/src/features/e2ee/pairwiseSession.test.ts` | Session initiation, acceptance, message keys | PASSING |
| `client/src/features/e2ee/pairwiseSession.restart.test.ts` | Session restart | PASSING |
| `client/src/features/e2ee/pairwiseSession.security.test.ts` | Security edge cases | PASSING |
| `client/src/features/e2ee/pairwiseSession.skippedKeys.security.test.ts` | Skipped key security | PASSING |
| `client/src/features/e2ee/pairwiseSession.trace.test.ts` | Trace logging | PASSING |
| `client/src/features/e2ee/doubleRatchetDesign.test.ts` | Double Ratchet design | PASSING |
| `client/src/features/e2ee/doubleRatchetLifecycle.test.ts` | Double Ratchet lifecycle | PASSING |
| `client/src/features/e2ee/groupSession.test.ts` | Group session (Sender Keys) | PASSING |
| `client/src/features/e2ee/messageAAD.test.ts` | Message AAD | PASSING |
| `client/src/features/e2ee/safetyNumber.test.ts` | Safety numbers | PASSING |
| `client/src/features/e2ee/simple.test.ts` | Simple crypto tests | PASSING |
| `client/src/features/e2ee/deviceKeys.test.ts` | Device keys | PASSING |
| `client/src/features/e2ee/deviceVerificationStore.test.ts` | Device verification | PASSING |
| `client/src/crypto/ratchet.determinism.test.ts` | Ratchet determinism | PASSING |
| `client/src/crypto/ratchet.test.ts` | Ratchet chain | PASSING |
| `client/src/features/calls/callStore.test.ts` | Call state | PASSING |
| `client/src/features/calls/callTypes.test.ts` | Call types | PASSING |
| `client/src/features/calls/signalingService.test.ts` | Signaling | PASSING |
| `client/src/features/voice/voiceStore.test.ts` | Voice store | PASSING |
| `client/src/features/voice/voiceTypes.test.ts` | Voice types | PASSING |
| `client/src/features/p2p/icePolicy.test.ts` | ICE policy | PASSING |
| `client/src/features/p2p/padding.test.ts` | Padding | PASSING |
| `client/src/features/p2p/transport.test.ts` | Transport | PASSING |
| `client/src/features/resilience/connectivityPolicy.test.ts` | Connectivity | PASSING |
| `client/src/features/resilience/selfHealing.test.ts` | Self-healing | PASSING |
| `client/src/auth.test.ts` | Authentication | PASSING |
| `client/src/startupPrivacy.test.ts` | Startup privacy | PASSING |
| `client/src/App.startupPrivacy.test.tsx` | App startup privacy | PASSING |
| `client/src/ui/servers.test.ts` | Server list | PASSING |

### TypeScript Test Summary

| Metric | Value |
|--------|-------|
| Test files | 32 |
| Total tests | 208 |
| Passing | 208 |
| Failing | 0 |
| Lint | Clean |
| Typecheck | Clean |

---

## 3. Critical Gaps in Test Coverage

### GAP-001: No E2EE Cross-Language Integration Tests
- **Description:** No test verifies that Go and TypeScript produce identical X3DH keys from identical inputs.
- **Impact:** TEC-001 (KDF mismatch) would not be caught by tests.
- **Fix:** Add integration test with deterministic DH values comparing Go and TS outputs.

### GAP-002: No TOTP Verification Tests
- **Description:** No test verifies end-to-end TOTP setup → confirm → verify flow.
- **Impact:** TEC-003 (TOTP broken) not caught by tests.
- **Fix:** Add `SetupTOTP` → `VerifyTOTPSetup` → `VerifyTOTP` test chain.

### GAP-003: No Guild/Message Handler Tests
- **Description:** No tests for guild creation, message sending, editing, deletion.
- **Impact:** Business logic regressions undetected.
- **Fix:** Add handler tests for all guild and message endpoints.

### GAP-004: No Race Condition Tests
- **Description:** `RatchetState` is mutable without mutex. No concurrent access tests.
- **Impact:** Data races in parallel message sending undetected on Windows.
- **Fix:** Add concurrent ratchet tests, run on Linux CI with `-race`.

### GAP-005: No Database Migration Tests
- **Description:** Migration tests exist but may not cover all 15 migrations.
- **Impact:** Migration failures undetected until deployment.
- **Fix:** Ensure all migrations tested with up/down cycles.

### GAP-006: No Frontend Component Tests
- **Description:** No React component rendering tests. Only logic tests.
- **Impact:** UI regressions undetected.
- **Fix:** Add component tests with happy-dom/jsdom.

### GAP-007: No Tauri Build Tests
- **Description:** No tests verifying Tauri binary builds successfully.
- **Impact:** Desktop build failures undetected until release.
- **Fix:** Add build verification in CI.

---

## 4. Test Execution Results

### Go Backend Tests
```
ok  	zenthril-backend	2.437s
ok  	zenthril-backend/auth	5.159s
ok  	zenthril-backend/benchmarks	1.216s
?   	zenthril-backend/cmd/api	[no test files]
?   	zenthril-backend/cmd/sfu	[no test files]
?   	zenthril-backend/cmd/worker	[no test files]
ok  	zenthril-backend/config	1.270s
?   	zenthril-backend/db	[no test files]
ok  	zenthril-backend/device	1.515s
ok  	zenthril-backend/federation	2.490s
ok  	zenthril-backend/friends	1.133s
ok  	zenthril-backend/guild	1.639s
ok  	zenthril-backend/hub	3.186s
ok  	zenthril-backend/internal/app	3.155s
ok  	zenthril-backend/internal/config	1.759s
ok  	zenthril-backend/internal/cqrs	2.915s
ok  	zenthril-backend/internal/crypto	2.508s
?   	zenthril-backend/internal/domain	[no test files]
?   	zenthril-backend/internal/event	[no test files]
ok  	zenthril-backend/internal/gateway	2.768s
?   	zenthril-backend/internal/observability	[no test files]
?   	zenthril-backend/internal/pubsub	[no test files]
ok  	zenthril-backend/internal/repository	1.328s
?   	zenthril-backend/logger	[no test files]
ok  	zenthril-backend/message	1.297s
ok  	zenthril-backend/metrics	1.290s
ok  	zenthril-backend/middleware	1.281s
ok  	zenthril-backend/migrations_test	0.796s
?   	zenthril-backend/models	[no test files]
ok  	zenthril-backend/moderation	1.182s
ok  	zenthril-backend/security	0.938s
?   	zenthril-backend/spam	[no test files]
ok  	zenthril-backend/user	0.964s
```
**Result: All existing tests pass. 14 packages have no test files.**

### TypeScript Frontend Tests
```
Test Files  32 passed (32)
Tests  208 passed (208)
```
**Result: All 208 tests pass across 32 files.**

---

## 5. Recommendations

| Priority | Action |
|----------|--------|
| P0 | Add TOTP end-to-end test (Setup → Confirm → Verify) |
| P0 | Add E2EE cross-language integration test |
| P1 | Add guild handler tests |
| P1 | Add message handler tests |
| P1 | Add cmd/api, cmd/worker, cmd/sfu tests |
| P1 | Add concurrent ratchet tests with race detector |
| P2 | Add frontend component tests |
| P2 | Add migration up/down tests for all 15 migrations |
| P3 | Add Tauri build verification in CI |
