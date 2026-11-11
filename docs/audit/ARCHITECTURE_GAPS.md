# Zenthril Architecture Gaps Report
**Generated:** 2026-11-02  
**Scope:** Architecture analysis, module boundaries, dependency flow  
**Classification:** INTERNAL - For engineering use only

---

## 1. Current Architecture

### Backend 1.x (Go) - Current Production Code

**Structure:** Modular monolith with packages organized by feature (auth, device, message, guild, friends, federation, moderation, security, user, hub, metrics, spam).

**Layers (implicit, not enforced):**
- Transport: Chi HTTP router, Gorilla WebSocket
- Application: Service layer in each package
- Domain: Models in `models/` package (minimal)
- Infrastructure: PostgreSQL (pgx), Redis (go-redis), Kafka (segmentio/kafka-go)

**Dependency Injection:** Manual in `backend/main.go`. No DI container.

**Key Issues:**
- Business logic mixed with HTTP handlers in some packages
- No clear separation between domain, application, and infrastructure layers
- Direct PostgreSQL access from handlers (no repository abstraction)
- No use of the `fx` DI container in the legacy main.go
- `cmd/api/` uses fx but is experimental and incomplete

### Backend-2.0 (Go) - Promised Next-Gen

**Structure:** Empty directory skeleton with only:
- `go.mod` (module `zenthril-backend-2.0`, Go 1.26.1)
- `internal/platform/logger/logger.go`
- `pkg/errors/errors.go`

**Planned Architecture (from directory names):**
- `internal/app/` - Application layer
- `internal/config/` - Configuration
- `internal/domains/` - Domain layer (administration, autonomous_defense, communication, cryptographic, device, identity, media, messaging, notification)
- `internal/infrastructure/` - Infrastructure (database, redis, secret_manager)
- `internal/platform/` - Platform (http, logger, websocket)
- `pkg/crypto/` - Cryptographic primitives
- `pkg/errors/` - Error types
- `cmd/api/` - API entrypoint
- `cmd/worker/` - Worker entrypoint

**Status:** NOT IMPLEMENTED. All directories empty except logger.go and errors.go.

### Client (Tauri 2 + React + TypeScript)

**Structure:** Feature-based organization:
- `src/features/` - Calls, e2ee, p2p, voice, webrtc
- `src/components/` - Reusable UI components
- `src/store/` - Zustand state management
- `src/api/` - API client
- `src/transport/` - WebSocket transport
- `src/crypto/` - Cryptographic utilities

**State Management:** Zustand stores
**API Client:** Custom fetch-based client + socket.io-client for WebSocket
**E2EE:** @noble/curves, @noble/hashes for X25519, HKDF, AES-GCM

---

## 2. Architecture Gaps

### GAP-001: No Enforced Layer Separation in Backend 1.x
- **Severity:** MEDIUM
- **Description:** Handlers directly access PostgreSQL via `pgxpool.Pool`. No repository or data access abstraction. Business logic and data access are mixed.
- **Impact:** Hard to test, hard to modify data layer, tight coupling.
- **Fix:** Introduce repository pattern. Handlers → Services → Repositories → Database.

### GAP-002: Two Parallel Backend Codebases
- **Severity:** HIGH
- **Description:** `backend/` (working but legacy) and `backend-2.0/` (empty skeleton) exist simultaneously. No migration path defined.
- **Impact:** Confusion about which codebase to use. Duplicated configuration, dependencies.
- **Fix:** Either complete backend-2.0 and migrate, or consolidate backend-2.0 into backend 1.x.

### GAP-003: No CQRS/Event Sourcing Boundaries
- **Severity:** MEDIUM
- **Description:** `internal/cqrs/` exists with CommandBus, QueryBus, EventStore but is not wired into the legacy main.go. The fx-based cmd/api uses it but is experimental.
- **Impact:** CQRS patterns exist but unused in production path.
- **Fix:** Wire CQRS into legacy main.go or deprecate legacy path.

### GAP-004: No Domain Layer Boundaries
- **Severity:** MEDIUM
- **Description:** `internal/domain/` exists with `types.go` but no domain entities, value objects, or domain services. Anemic domain model.
- **Impact:** Business logic scattered across services and handlers.
- **Fix:** Define domain entities and move business logic from services to domain layer.

### GAP-005: No Interface Definitions for External Dependencies
- **Severity:** MEDIUM
- **Description:** PostgreSQL, Redis, WebSocket clients used directly throughout codebase. No interface abstractions.
- **Impact:** Hard to mock for testing, hard to swap implementations.
- **Fix:** Define interfaces for DB, Redis, WebSocket, event bus.

### GAP-006: No API Versioning Strategy
- **Severity:** MEDIUM
- **Description:** API at `/api/v1` with no versioning middleware or deprecation policy.
- **Impact:** Breaking changes require client updates without notice.
- **Fix:** Add API versioning and deprecation headers.

### GAP-007: No Observability Integration in Legacy Path
- **Severity:** LOW
- **Description:** `internal/observability/` exists with TracerProvider but only used in cmd/api (fx path). Legacy main.go has no tracing.
- **Impact:** No distributed tracing for production requests.
- **Fix:** Add tracing middleware to legacy HTTP router.

### GAP-008: No Feature Flag System
- **Severity:** LOW
- **Description:** No mechanism to enable/disable features at runtime without redeploy.
- **Impact:** Cannot gradually roll out or disable problematic features.
- **Fix:** Add feature flag middleware.

### GAP-009: No Anti-Corruption Layer for Federation
- **Severity:** MEDIUM
- **Description:** Federation endpoints exist but no ACL/anti-corruption layer to protect internal domain model from external federation messages.
- **Impact:** Malformed federation messages could corrupt internal state.
- **Fix:** Add validation/sanitization layer for federation inbound messages.

### GAP-010: No Multi-Tenancy Isolation
- **Severity:** MEDIUM
- **Description:** No clear tenant isolation in database queries. Guild/user IDs used but no enforcement at query level.
- **Impact:** Potential cross-tenant data leakage if queries miss WHERE clauses.
- **Fix:** Add tenant context to all database queries.

---

## 3. Target Architecture Requirements

Based on the analysis, the target architecture should be:

**Modular Monolith with Clear Layer Separation:**

```
┌─────────────────────────────────────────┐
│  Transport/API Layer (Chi, WebSocket)   │
├─────────────────────────────────────────┤
│  Application Layer (Services, UseCases) │
├─────────────────────────────────────────┤
│  Domain Layer (Entities, Value Objects) │
├─────────────────────────────────────────┤
│  Infrastructure Layer (DB, Redis, MQ)   │
└─────────────────────────────────────────┘
```

**Key Principles:**
1. Domain layer has zero dependencies on infrastructure
2. Application layer depends on domain interfaces
3. Infrastructure layer implements domain interfaces
4. Transport layer depends on application services
5. Dependency injection via fx container

---

## 4. Module Dependencies

### Current Dependencies (Go Backend)

```
main.go → auth, device, guild, friends, federation, message, hub, metrics, security, spam, user
auth → pgxpool, redis, jwt
device → pgxpool
guild → pgxpool, wsHub
friends → pgxpool, wsHub
federation → pgxpool
message → pgxpool, wsHub, guildSvc
hub → pgxpool, redis, guildSvc
security → pgxpool, redis
moderation → pgxpool
```

### Current Dependencies (TypeScript Client)

```
src/App.tsx → src/store/*, src/features/*, src/api/*, src/transport/*
src/features/e2ee/* → src/crypto/ratchet.ts, @noble/curves, @noble/hashes
src/transport/wsGlobal.ts → socket.io-client
src/api/auth.ts → fetch API
```

---

## 5. Migration Plan

### Phase 1: Stabilize Backend 1.x
- Fix critical defects (X3DH KDF, TOTP)
- Add missing tests
- Enforce repository pattern in new code

### Phase 2: Introduce Layer Boundaries
- Create repository interfaces
- Refactor services to use repositories
- Add domain entities

### Phase 3: Consolidate or Complete backend-2.0
- Either implement backend-2.0 fully with clean architecture
- Or merge backend-2.0 concepts into backend 1.x
- Remove duplicate codebase

### Phase 4: Add Observability and Feature Flags
- Distributed tracing
- Feature flag system
- Enhanced metrics

---
