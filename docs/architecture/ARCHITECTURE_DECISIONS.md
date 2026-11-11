# Zenthril Architecture Decision Records
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## ADR-001: Modular Monolith Architecture

**Status:** Accepted

**Context:** The project needed an architecture that balances maintainability with development speed. Microservices were considered but rejected due to operational complexity for a self-hosted product.

**Decision:** Use a modular monolith with clear layer separation (Domain, Application, Infrastructure, Transport). All modules in a single Go binary with package-level boundaries.

**Consequences:**
- ✅ Simpler deployment (single binary)
- ✅ No network overhead between modules
- ✅ Easier refactoring within monolith
- ❌ Requires discipline to maintain boundaries
- ❌ Horizontal scaling requires running multiple instances

---

## ADR-002: Uber fx for Dependency Injection

**Status:** Accepted (for cmd/api path)

**Context:** The legacy main.go uses manual dependency wiring. The cmd/api path uses Uber fx.

**Decision:** Use Uber fx for the next-generation API entrypoint. The legacy main.go will be migrated gradually.

**Consequences:**
- ✅ Explicit dependency graph
- ✅ Lifecycle management (start/stop hooks)
- ✅ Testable with fx test utilities
- ❌ Learning curve for team
- ❌ Two entry points during migration period

---

## ADR-003: PostgreSQL as Primary Database

**Status:** Accepted

**Context:** Needed a relational database for structured data (users, guilds, messages).

**Decision:** Use PostgreSQL 16 with pgx driver.

**Consequences:**
- ✅ Mature, reliable, feature-rich
- ✅ JSONB support for flexible schemas
- ✅ Good Go driver support (pgx)
- ❌ Requires operational expertise
- ❌ Vertical scaling limits

---

## ADR-004: Redis for Sessions and Rate Limiting

**Status:** Accepted

**Context:** Need fast, TTL-based storage for sessions, rate limit counters, refresh tokens.

**Decision:** Use Redis 7 with go-redis client.

**Consequences:**
- ✅ Sub-millisecond latency
- ✅ Built-in TTL expiration
- ✅ Pub/Sub for message fan-out
- ❌ Additional infrastructure component
- ❌ Data loss on restart (mitigated by appendonly)

---

## ADR-005: JWT for Authentication

**Status:** Accepted

**Context:** Need stateless authentication for REST API.

**Decision:** Use JWT (HS256) with short-lived access tokens (15 min) and long-lived refresh tokens (30 days).

**Consequences:**
- ✅ Stateless server-side validation
- ✅ Refresh token rotation for security
- ✅ Cookie-based storage (HttpOnly, Secure)
- ❌ Token revocation requires blacklist
- ❌ HS256 requires shared secret (not suitable for federation)

---

## ADR-006: X3DH + Double Ratchet for E2EE

**Status:** Accepted

**Context:** Need end-to-end encryption with forward secrecy and post-compromise security.

**Decision:** Implement X3DH for session initialization and Double Ratchet for message encryption, matching Signal Protocol design.

**Consequences:**
- ✅ Forward secrecy
- ✅ Post-compromise security
- ✅ Asynchronous key exchange (offline recipients)
- ❌ Complex implementation
- ❌ Cross-language compatibility challenges
- ❌ No built-in group efficiency (Sender Keys only)

---

## ADR-007: Argon2id for Password Hashing

**Status:** Accepted

**Context:** Need secure password storage resistant to GPU cracking.

**Decision:** Use Argon2id with configurable parameters.

**Consequences:**
- ✅ Memory-hard (resistant to GPU/ASIC)
- ✅ Configurable security parameters
- ✅ Winner of Password Hashing Competition
- ❌ Higher computational cost than bcrypt

---

## ADR-008: Go 1.26.5

**Status:** Accepted

**Context:** Need a modern Go version with security fixes and performance improvements.

**Decision:** Use Go 1.26.5.

**Consequences:**
- ✅ Recent security patches
- ✅ Performance improvements
- ❌ May not be available on all platforms
- ❌ backend-2.0 uses Go 1.26.1 (version mismatch)

---

## ADR-009: Tauri 2 for Desktop Client

**Status:** Accepted

**Context:** Need a cross-platform desktop client with web technologies.

**Decision:** Use Tauri 2 with React and TypeScript.

**Consequences:**
- ✅ Smaller binary than Electron
- ✅ Native OS integration
- ✅ Web technology stack
- ❌ Rust learning curve for native code
- ❌ Tauri ecosystem smaller than Electron

---

## ADR-010: Backend-2.0 as Separate Module

**Status:** Reconsidering

**Context:** backend-2.0 was planned as a clean-architecture rewrite but is empty.

**Decision:** Either complete backend-2.0 implementation or consolidate into backend 1.x. Having two backends creates maintenance burden.

**Consequences:**
- ✅ Clean architecture (if completed)
- ❌ Two codebases to maintain
- ❌ Configuration duplication
- ❌ Deployment complexity

**Recommendation:** Consolidate backend-2.0 concepts into backend 1.x through gradual refactoring (Strangler Pattern).

---

## ADR-011: No Microservices

**Status:** Accepted

**Context:** Considered splitting backend into auth, message, guild services.

**Decision:** Stay with modular monolith. Evaluate microservices only if single instance cannot handle load.

**Consequences:**
- ✅ Operational simplicity
- ✅ No inter-service communication complexity
- ✅ Shared database simplifies transactions
- ❌ Single point of failure (mitigated by running multiple instances)

---

## ADR-012: No Built-in Group E2EE (MLS)

**Status:** Accepted

**Context:** Considered implementing Message Layer Security (MLS) for group E2EE.

**Decision:** Use Sender Keys foundation for group E2EE with fail-closed limit (10 devices). MLS is future work.

**Consequences:**
- ✅ Simpler implementation
- ✅ Works for small groups
- ❌ Not suitable for large groups (100+ members)
- ❌ Requires different protocol for large-scale E2EE groups

---
