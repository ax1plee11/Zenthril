# Zenthril Project Inventory
**Generated:** 2026-11-02  
**Working Directory:** `C:\Users\User\Desktop\дс аналонг`  
**Git Branch:** `main`  
**HEAD:** `2476e9b test(e2ee): document Double Ratchet lifecycle and post-compromise finding`  
**Status:** Uncommitted changes present (17 modified, 11 untracked)

---

## 1. Repository Structure

### Root Level
| Item | Type | Description |
|------|------|-------------|
| `.devcontainer/` | dir | Dev container configuration |
| `.github/` | dir | GitHub workflows, templates |
| `.kilo/` | dir | Kilo agent configuration |
| `.snapshots/` | dir | Project snapshots |
| `.venv/` | dir | Python virtual environment |
| `.vscode/` | dir | VS Code settings |
| `backend/` | dir | Go backend (main API server) |
| `backend-2.0/` | dir | Next-gen backend skeleton (empty) |
| `client/` | dir | Tauri 2 + React + TypeScript desktop client |
| `deployments/` | dir | Deployment manifests |
| `docs/` | dir | Documentation |
| `research/` | dir | Benchmark and research materials |
| `security/` | dir | Security documentation |
| `zenthril-audit/` | dir | Previous audit materials |
| `zenthrildb-push/` | dir | ZenthrilDB C++ integration (untracked) |
| `.env` | file | Local environment (contains secrets - **should not be committed**) |
| `.env.example` | file | Example environment template |
| `docker-compose.yml` | file | Docker Compose deployment |
| `docker-compose.debug.yml` | file | Debug Docker Compose |
| `Dockerfile` | file | Root Dockerfile |
| `README.md` | file | English README |
| `README.ru.md` | file | Russian README |
| `README.uk.md` | file | Ukrainian README |
| `SECURITY.md` | file | Security policy |
| `THREAT_MODEL.md` | file | Threat model |
| `CHANGELOG.md` | file | Changelog |
| `CONTRIBUTING.md` | file | Contribution guide |
| `CODE_OF_CONDUCT.md` | file | Code of conduct |
| `LICENSE` | file | License |
| `NOTICE` | file | Attribution notice |
| `PRIVACY_POLICY.md` | file | Privacy policy |
| `TERMS_OF_SERVICE.md` | file | Terms of service |
| `AUDIT_BASELINE.md` | file | Audit baseline |
| `CODEBASE_AUDIT.md` | file | Codebase audit |
| `INTEGRATION_ANALYSIS.md` | file | Integration analysis |
| `PROGRESS_STATUS.md` | file | Progress status |
| `PHASE1_PRODUCT_SECURITY.md` | file | Phase 1 security |
| `ZFDA_1.3_FINAL_SUMMARY.md` | file | ZFDA 1.3 summary |
| `ZENTHRIL_COMPLETION_PLAN.md` | file | Completion plan |
| `Zenthril_Architecture_Report.docx` | file | Architecture report |
| `Zenthril_ZenthrilDB_Technical_Analysis.docx` | file | ZenthrilDB analysis |
| `generate_dissertation.py` | file | Dissertation generator |
| `railway.toml` | file | Railway deployment config |
| `.admin_notes.txt` | file | Admin notes |
| `.commitlintrc.json` | file | Commit lint config |
| `.dockerignore` | file | Docker ignore |
| `.gitignore` | file | Git ignore |
| `.gitleaks.toml` | file | Gitleaks config (untracked) |

---

## 2. Backend (Go) Inventory

### go.mod
- **Module:** `zenthril-backend`
- **Go Version:** `1.26.5`
- **Direct Dependencies:** chi, golang-jwt/jwt, google/uuid, gorilla/websocket, jackc/pgx, joho/godotenv, redis/go-redis, segmentio/kafka-go, stretchr/testify, go.uber.org/fx, golang.org/x/crypto

### Backend Modules (packages)

| Package | Files | Tests | Status |
|---------|-------|-------|--------|
| `auth/` | audit.go, handler.go, handler_test.go, jwt.go, jwt_test.go, middleware.go, password.go, password_test.go, service.go, totp.go | handler_test, jwt_test, password_test | IMPLEMENTED (TOTP broken) |
| `cmd/api/` | main.go | none | EXPERIMENTAL (fx-based) |
| `cmd/resetpw/` | main.go | none | SKELETON (no Go files found) |
| `cmd/sfu/` | main.go | none | SKELETON |
| `cmd/worker/` | main.go | none | SKELETON |
| `config/` | (none - uses internal/config) | none | N/A |
| `db/` | (none) | none | NO TEST FILES |
| `device/` | (none) | none | IMPLEMENTED |
| `federation/` | (none) | service_test.go (newly added) | PARTIAL (fail-closed) |
| `friends/` | (none) | (has tests) | IMPLEMENTED |
| `guild/` | (none) | none | IMPLEMENTED (no tests) |
| `hub/` | (none) | (has tests) | IMPLEMENTED |
| `internal/app/` | container.go, module.go | module_security_test.go | fx DI container |
| `internal/config/` | config.go, config_test.go | config_test.go | CONFIGURATION |
| `internal/cqrs/` | command.go, cqrs_test.go, event_store.go, query.go | cqrs_test.go | CQRS |
| `internal/crypto/` | message.go, message_test.go, multidevice.go, multidevice_test.go, protocol.go, protocol_test.go, ratchet.go, ratchet_test.go, session_store.go, test_vectors_test.go, x25519.go | multiple test files | CRYPTO (X3DH mismatch) |
| `internal/domain/` | types.go | none | DOMAIN TYPES |
| `internal/event/` | event.go, factory.go, kafka_bus.go, memory_bus.go | none | EVENT BUS |
| `internal/gateway/` | auth.go, auth_test.go, client_ip_test.go, handler.go, handler_test.go, message.go, rate_limit.go, redis_limiter.go, redis_session_validator.go, registry.go, registry_security_test.go, registry_test.go, security.go, tracer.go | multiple test files | GATEWAY |
| `internal/observability/` | (none) | none | OBSERVABILITY |
| `internal/pubsub/` | pubsub.go, shard.go, shard_test.go | shard_test.go | PUBSUB |
| `internal/repository/` | (none) | none | REPOSITORY |
| `logger/` | (none) | none | LOGGER |
| `message/` | (none) | none | MESSAGE (no tests) |
| `metrics/` | (none) | none | METRICS |
| `middleware/` | (none) | none | MIDDLEWARE |
| `migrations/` | 15 SQL files | migrations_test | MIGRATIONS |
| `migrations_test/` | (none) | migration tests | MIGRATION TESTS |
| `models/` | (none) | none | MODELS |
| `moderation/` | (none) | none | MODERATION (IP log fixed) |
| `security/` | (none) | none | SECURITY |
| `spam/` | (none) | none | SPAM |
| `static/` | (none) | none | STATIC FILES |
| `user/` | (none) | none | USER |

### Key Go Files
| File | Lines | Purpose |
|------|-------|---------|
| `backend/main.go` | 499 | Legacy HTTP server entrypoint |
| `backend/cmd/api/main.go` | 137 | Next-gen API (fx container) |
| `backend/cmd/worker/main.go` | 33 | Worker skeleton |
| `backend/cmd/sfu/main.go` | 33 | SFU control plane skeleton |
| `backend/internal/config/config.go` | 371 | Configuration with validation |
| `backend/internal/app/container.go` | 133 | fx DI container |
| `backend/internal/app/module.go` | 305 | fx module definition |

---

## 3. Client (Tauri 2 + React + TypeScript) Inventory

### package.json
- **Name:** `zenthril-client`
- **Version:** `0.1.0`
- **Type:** module
- **Key Dependencies:** @noble/curves, @noble/hashes, @tauri-apps/api, react, react-dom, socket.io-client, zustand, i18next, tailwindcss
- **Dev Dependencies:** vitest, eslint, typescript, @tauri-apps/cli, happy-dom, jsdom

### Client Source Structure

| Directory | Purpose |
|-----------|---------|
| `src/api/` | API client layer |
| `src/components/` | React components |
| `src/config/` | Configuration |
| `src/crypto/` | Cryptographic utilities (ratchet.ts) |
| `src/features/` | Feature modules (calls, e2ee, p2p, voice, webrtc) |
| `src/i18n/` | Internationalization (en, ru, uk) |
| `src/lib/` | Shared libraries |
| `src/notifications/` | Notification system |
| `src/platform/` | Platform-specific code |
| `src/resilience/` | Connection resilience |
| `src/store/` | Zustand state management |
| `src/transport/` | Transport layer |
| `src/types/` | TypeScript types |
| `src/util/` | Utilities |
| `src/voice/` | Voice components |

### E2EE Client Files
| File | Purpose |
|------|---------|
| `src/crypto/ratchet.ts` | Symmetric Double Ratchet chain |
| `src/features/e2ee/pairwiseSession.ts` | X3DH + Double Ratchet session management |
| `src/features/e2ee/deviceKeys.ts` | Device key management |
| `src/features/e2ee/deviceKeyStore.ts` | Device key storage |
| `src/features/e2ee/deviceRegistration.ts` | Device registration |
| `src/features/e2ee/deviceVerificationStore.ts` | Device verification |
| `src/features/e2ee/doubleRatchetDesign.test.ts` | Double Ratchet design tests |
| `src/features/e2ee/doubleRatchetLifecycle.test.ts` | Double Ratchet lifecycle tests |
| `src/features/e2ee/encoding.ts` | Binary encoding |
| `src/features/e2ee/groupSession.ts` | Group session (Sender Keys) |
| `src/features/e2ee/messageAAD.ts` | Message Associated Data |
| `src/features/e2ee/messageEnvelopes.ts` | Message envelope format |
| `src/features/e2ee/pairwiseSession.restart.test.ts` | Restart tests |
| `src/features/e2ee/pairwiseSession.security.test.ts` | Security tests |
| `src/features/e2ee/pairwiseSession.skippedKeys.security.test.ts` | Skipped keys security |
| `src/features/e2ee/pairwiseSession.test.ts` | Session tests |
| `src/features/e2ee/pairwiseSession.trace.test.ts` | Trace tests |
| `src/features/e2ee/safetyNumber.test.ts` | Safety number tests |
| `src/features/e2ee/safetyNumber.ts` | Safety number implementation |
| `src/features/e2ee/simple.test.ts` | Simple tests |
| `src/features/e2ee/types.ts` | E2EE types |

---

## 4. Backend-2.0 Inventory

**Status:** Skeleton - NOT IMPLEMENTED

| Path | Files | Status |
|------|-------|--------|
| `backend-2.0/go.mod` | go.mod | Module `zenthril-backend-2.0`, Go 1.26.1 |
| `backend-2.0/cmd/api/` | (empty) | No Go files |
| `backend-2.0/cmd/worker/` | (empty) | No Go files |
| `backend-2.0/internal/app/` | (empty) | No Go files |
| `backend-2.0/internal/config/` | (empty) | No Go files |
| `backend-2.0/internal/domains/` | administration, autonomous_defense, communication, cryptographic, device, identity, media, messaging, notification | Empty domain directories |
| `backend-2.0/internal/infrastructure/` | database, redis, secret_manager | Empty infrastructure directories |
| `backend-2.0/internal/platform/` | http, logger, websocket | logger.go only |
| `backend-2.0/pkg/crypto/` | (empty) | No Go files |
| `backend-2.0/pkg/errors/` | errors.go | AppError type only |

---

## 5. Migrations Inventory

| Migration | Description |
|-----------|-------------|
| `001_initial_schema.sql` | Initial schema |
| `002_global_bans.sql` | Global bans table |
| `003_friends.sql` | Friends table |
| `004_admin_audit_log.sql` | Admin audit log |
| `005_ip_tracking_and_guild_bans.sql` | IP tracking and guild bans |
| `006_device_keys.sql` | Device keys table |
| `007_add_mfa_totp.sql` | MFA/TOTP table |
| `008_message_crypto_envelope.sql` | Message crypto envelope |
| `009_federation_messages.sql` | Federation messages |
| `010_rbac_v2.sql` | RBAC v2 |
| `011_message_aad_v2.sql` | Message AAD v2 |
| `012_device_identity_dh_key.sql` | Device identity DH key |
| `013_message_recipient_envelopes.sql` | Message recipient envelopes |
| `014_device_sessions.sql` | Device sessions |
| `015_recipient_envelope_dh_public_key.sql` | Recipient envelope DH public key |

---

## 6. ZenthrilDB Inventory

| Path | Status |
|------|--------|
| `zenthrildb-push/` | Untracked C++ code, not integrated with Go backend |
| `Zenthril_ZenthrilDB_Technical_Analysis.docx` | Analysis document |

---

## 7. Configuration Inventory

| File | Status |
|------|--------|
| `backend/.env` | Contains plaintext password (**SECRET EXPOSED**) |
| `.env.example` | Template |
| `backend/.golangci.yml` | Lint config |
| `client/.env` | Client environment |
| `client/.env.example` | Client template |
| `docker-compose.yml` | Docker Compose (Postgres 16, Redis 7, Backend) |
| `docker-compose.debug.yml` | Debug compose |
| `Dockerfile` | Backend Dockerfile |
| `backend/Dockerfile` | Backend-specific Dockerfile |
| `railway.toml` | Railway deployment |

---

## 8. Documentation Inventory

| File | Status |
|------|--------|
| `README.md` | English README (Alpha status) |
| `README.ru.md` | Russian README |
| `README.uk.md` | Ukrainian README |
| `SECURITY.md` | Security policy |
| `THREAT_MODEL.md` | Threat model |
| `CHANGELOG.md` | Changelog |
| `CONTRIBUTING.md` | Contribution guide |
| `CODE_OF_CONDUCT.md` | Code of conduct |
| `AUDIT_BASELINE.md` | Audit baseline |
| `CODEBASE_AUDIT.md` | Codebase audit |
| `INTEGRATION_ANALYSIS.md` | Integration analysis |
| `PROGRESS_STATUS.md` | Progress status |
| `PHASE1_PRODUCT_SECURITY.md` | Phase 1 security |
| `ZFDA_1.3_FINAL_SUMMARY.md` | ZFDA 1.3 summary |
| `ZENTHRIL_COMPLETION_PLAN.md` | Completion plan |
| `docs/` | Documentation directory (partially populated) |

---

## 9. Git Status Summary

- **Branch:** `main`
- **HEAD:** `2476e9b`
- **Modified files (17):** .github/workflows/ci.yml, README.md, SECURITY.md, backend/Dockerfile, backend/auth/handler.go, backend/auth/service.go, backend/internal/crypto/protocol.go, backend/internal/crypto/protocol_test.go, backend/internal/crypto/ratchet.go, backend/internal/crypto/test_vectors_test.go, backend/internal/crypto/x25519.go, backend/main.go, client/src-tauri/tauri.conf.json, client/src/features/e2ee/pairwiseSession.security.test.ts, client/src/features/e2ee/pairwiseSession.trace.test.ts, docker-compose.yml, research/results/RESEARCH_REPORT.md
- **Untracked files (11):** .dockerignore, .gitleaks.toml, .kilo/, INTEGRATION_ANALYSIS.md, ZFDA_1.3_FINAL_SUMMARY.md, Zenthril_ZenthrilDB_Technical_Analysis.docx, backend-2.0/, backend/auth/totp.go, backend/federation/service_test.go, backend/migrations_test/, client/src/features/e2ee/doubleRatchetDesign.test.ts, docs/architecture/, docs/audits/, docs/security/ZFDA/, generate_dissertation.py, zenthrildb-push/

---

## 10. Secret Detection

**CRITICAL:** `backend/.env` contains plaintext password:
```
POSTGRES_PASSWORD=[REDACTED_POSTGRES_PASSWORD]
DB_URL=postgres://zenthril:[REDACTED_POSTGRES_PASSWORD]@localhost:5432/zenthril?sslmode=disable
JWT_SECRET=[REDACTED_JWT_SECRET]
```

This file must NOT be committed to Git. It is already not in the git index (untracked or ignored).

---

## 11. Known Gaps Summary

| Category | Gap | Severity |
|----------|-----|----------|
| E2EE | X3DH KDF `info` mismatch between Go and TypeScript | CRITICAL |
| Auth | TOTP stores SHA-256 hash instead of base32 secret | CRITICAL |
| Auth | No rate limiting on TOTP endpoints | HIGH |
| Architecture | backend-2.0 is empty skeleton | HIGH |
| Testing | No tests for guild, message, cmd packages | HIGH |
| Auth | No password reset / account recovery | HIGH |
| Data | ZenthrilDB not integrated | HIGH |
| Security | No safety numbers UI verification | MEDIUM |
| Security | No secure local key storage encryption | MEDIUM |
| Security | Security log without rotation | MEDIUM |
| Testing | Race condition tests not runnable on Windows | MEDIUM |
| Docs | Documentation not synchronized with code | LOW |
