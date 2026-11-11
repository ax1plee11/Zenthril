# Zenthril Target Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Overview

Zenthril is a self-hosted secure messaging platform. The target architecture is a **Modular Monolith** with clear layer separation, designed for maintainability, testability, and future horizontal scaling without the complexity of distributed microservices.

**Design Principles:**
- Low coupling, high cohesion
- Domain layer independent of infrastructure
- Dependency injection via Uber fx
- Type-safe contracts between layers
- Graceful degradation and failure isolation

---

## 2. Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Layer 10: Desktop Presentation (Tauri 2 + React + TS)   │
├─────────────────────────────────────────────────────────┤
│ Layer 9:  Observability (Tracing, Metrics, Logging)     │
├─────────────────────────────────────────────────────────┤
│ Layer 8:  Transport/API (Chi HTTP, WebSocket, Tauri IPC)│
├─────────────────────────────────────────────────────────┤
│ Layer 7:  Realtime/Messaging (WebSocket Hub, Pub/Sub)   │
├─────────────────────────────────────────────────────────┤
│ Layer 6:  Security (AuthN, AuthZ, Rate Limiting, E2EE)  │
├─────────────────────────────────────────────────────────┤
│ Layer 5:  Application (Use Cases, Services, CQRS)       │
├─────────────────────────────────────────────────────────┤
│ Layer 4:  Domain (Entities, Value Objects, Domain Svc)  │
├─────────────────────────────────────────────────────────┤
│ Layer 3:  Persistence (Repositories, Migrations)        │
├─────────────────────────────────────────────────────────┤
│ Layer 2:  Infrastructure (PostgreSQL, Redis, Kafka)     │
├─────────────────────────────────────────────────────────┤
│ Layer 1:  Shared Contracts (Types, Events, DTOs)        │
└─────────────────────────────────────────────────────────┘
```

**Dependency Rule:** Lower layers are independent of upper layers. Layer N can only depend on Layer N-1 and below, plus Shared Contracts (Layer 1).

---

## 3. Layer Responsibilities

### Layer 1: Shared Contracts
- TypeScript types, Go structs, protobuf definitions
- API request/response DTOs
- Domain events
- Error codes and types
- Configuration interfaces

### Layer 2: Infrastructure
- PostgreSQL 16 (primary database)
- Redis 7 (sessions, rate limiting, cache)
- Kafka (event bus - future)
- Docker containerization
- Secrets management

### Layer 3: Persistence
- Repository pattern implementations
- Database migrations (SQL)
- Connection pooling
- Transaction management
- Data seeding

### Layer 4: Domain
- Entities (User, Device, Message, Guild, Channel, Friendship)
- Value Objects (Email, Username, KeyBundle, SessionState)
- Domain services (EncryptionService, TokenService)
- Domain events (UserRegistered, MessageSent, DeviceRevoked)
- Business rules and invariants

### Layer 5: Application
- Use cases / application services
- CQRS: Command handlers and Query handlers
- Application events and event handlers
- DTO mapping
- Orchestration of domain operations

### Layer 6: Security
- JWT issuance and validation
- Refresh token rotation
- Argon2id password hashing
- TOTP/MFA
- Rate limiting (IP-based, endpoint-based)
- Brute-force protection
- CORS and CSP configuration
- Input validation and sanitization
- E2EE key management (X3DH, Double Ratchet)

### Layer 7: Realtime/Messaging
- WebSocket connection hub
- Message broadcasting
- Connection lifecycle (connect, disconnect, reconnect)
- Presence tracking
- Message queues

### Layer 8: Transport/API
- Chi HTTP router
- REST API endpoints
- WebSocket upgrade handler
- Tauri IPC handlers
- Middleware (auth, logging, rate limiting, CORS)

### Layer 9: Observability
- Structured logging (slog)
- Distributed tracing (OpenTelemetry)
- Prometheus metrics
- Health checks
- Readiness/liveness probes

### Layer 10: Desktop Presentation
- Tauri 2 shell
- React UI components
- Zustand state management
- TypeScript API client
- WebSocket client
- Local encrypted storage
- E2EE client-side (X3DH, Double Ratchet, AES-GCM)

---

## 4. Module Boundaries

### Core Modules

| Module | Layer | Responsibility | Dependencies |
|--------|-------|---------------|--------------|
| `auth` | 5,6 | Authentication, JWT, MFA | Domain: User, Device; Infra: PostgreSQL, Redis |
| `device` | 4,5,6 | Device registration, key management | Domain: Device, KeyBundle |
| `message` | 5,6,7 | Message send, receive, history | Domain: Message; Infra: PostgreSQL |
| `guild` | 4,5 | Guilds, channels, roles | Domain: Guild, Channel, Role |
| `friends` | 4,5 | Friend requests, relationships | Domain: Friendship, User |
| `federation` | 5,6,7 | Inter-server communication | Domain: FederationMessage; Infra: HTTP |
| `moderation` | 5,6 | Bans, mutes, reports | Domain: ModerationAction |
| `crypto` | 4,6 | X3DH, Double Ratchet, key derivation | Domain: KeyBundle, SessionState |
| `hub` | 7 | WebSocket hub, broadcasting | Infra: Redis (pub/sub) |
| `security` | 6 | Rate limiting, brute force, IP blocking | Infra: Redis |

### Dependency Direction

```
Desktop (Layer 10) → Transport (Layer 8) → Application (Layer 5) → Domain (Layer 4) → Infrastructure (Layer 2)
                                                    ↑                                              ↑
                                              Security (Layer 6) ←───────────────────────────────────┘
                                                    ↑
                                              Realtime (Layer 7)
                                                    ↑
                                              Observability (Layer 9)
```

---

## 5. Data Flow

### Message Send Flow

1. **Desktop** (Layer 10) → Encrypts message with E2EE → Sends via WebSocket
2. **Transport** (Layer 8) → WebSocket upgrade → Auth validates JWT → Routes to Message handler
3. **Application** (Layer 5) → MessageService validates, creates domain event
4. **Domain** (Layer 4) → Message entity enforces invariants (sender, recipient, content size)
5. **Persistence** (Layer 3) → Repository saves to PostgreSQL within transaction
6. **Infrastructure** (Layer 2) → PostgreSQL persists; Redis publishes for fan-out
7. **Realtime** (Layer 7) → Hub broadcasts to recipient WebSocket connections
8. **Observability** (Layer 9) → Logs message sent, increments metrics counter

### Authentication Flow

1. **Desktop** → POST /api/v1/auth/login with credentials
2. **Transport** → Chi router → Auth handler
3. **Application** → AuthService validates credentials, generates token pair
4. **Domain** → User entity verifies password hash (Argon2id)
5. **Persistence** → UserRepository fetches user from PostgreSQL
6. **Infrastructure** → Redis stores refresh token with TTL
7. **Security** → JWT signed with HS256; refresh token hashed and stored in Redis

---

## 6. Interface Definitions

### Domain Interfaces (Go)

```go
// UserRepository defines data access for users
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByUsername(ctx context.Context, username string) (*User, error)
    Update(ctx context.Context, user *User) error
}

// DeviceRepository defines data access for devices
type DeviceRepository interface {
    Create(ctx context.Context, device *Device) error
    GetByID(ctx context.Context, id string) (*Device, error)
    Revoke(ctx context.Context, id string) error
}

// KeyStore defines access to device key bundles
type KeyStore interface {
    GetBundle(ctx context.Context, userID, deviceID string) (DeviceKeyBundle, error)
    SaveSession(ctx context.Context, session SessionState) error
    ConsumeOneTimePreKey(ctx context.Context, userID, deviceID string) ([]byte, uint32, error)
}
```

### Application Interfaces

```go
// AuthService defines authentication use cases
type AuthService interface {
    Register(ctx context.Context, req RegisterRequest) (*AuthResult, error)
    Login(ctx context.Context, req LoginRequest) (*AuthResult, error)
    Refresh(ctx context.Context, refreshToken string) (*AuthResult, error)
    Logout(ctx context.Context, accessToken, refreshToken string) error
    SetupTOTP(ctx context.Context, userID string) (secret, uri string, err error)
    VerifyTOTP(ctx context.Context, userID, code string) error
}

// MessageService defines messaging use cases
type MessageService interface {
    SendMessage(ctx context.Context, req SendMessageRequest) (*Message, error)
    GetHistory(ctx context.Context, channelID string, limit int) ([]*Message, error)
    EditMessage(ctx context.Context, messageID, userID, content string) error
    DeleteMessage(ctx context.Context, messageID, userID string) error
}
```

---

## 7. Error Handling Strategy

1. **Domain errors** - Business rule violations (e.g., `ErrUserNotFound`, `ErrInvalidCredentials`)
2. **Infrastructure errors** - Database, network, cache failures (wrapped with context)
3. **Transport errors** - HTTP status codes, WebSocket close codes
4. **Validation errors** - Input validation failures with field-level details

All errors use sentinel values (`errors.Is`) for programmatic handling, with user-safe messages for display.

---

## 8. Security Boundaries

### Trust Boundaries
- **Client ↔ Server:** TLS 1.3, JWT authentication, WebSocket Origin validation
- **Server ↔ Database:** Trusted network (Docker internal), parameterized queries
- **Server ↔ Redis:** Trusted network, password authentication
- **Client ↔ Client:** E2EE (server never sees plaintext)

### Secrets
- JWT_SECRET: Minimum 32 bytes, never logged
- Refresh tokens: Hashed with SHA-256 before Redis storage
- TOTP secrets: Stored encrypted at rest
- E2EE private keys: Never leave client device

---

## 9. Deployment Architecture

```
Single Node (Development/Production):
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Client    │────→│   Backend   │────→│ PostgreSQL  │
│  (Tauri)    │  WSS│   (Go)      │     │   (16)      │
└─────────────┘      └─────────────┘      └─────────────┘
                                │
                                ↓
                          ┌─────────────┐
                          │    Redis    │
                          │     (7)     │
                          └─────────────┘

Multi Node (Production Scale):
┌─────────────┐      ┌─────────────┐
│   Client    │────→│  Load Bal.  │────→┌─────────────┐
│  (Tauri)    │  WSS│  (Nginx)    │→→→│ Backend A   │
└─────────────┘      └─────────────┘  │ (Go)        │
                                     └─────────────┘
                                             │
                                     ┌─────────────┐
                                     │ Backend B   │
                                     │ (Go)        │
                                     └─────────────┘
                                             │
                                     ┌─────────────┐
                                     │ PostgreSQL  │
                                     │   (16)      │
                                     └─────────────┘
                                             │
                                     ┌─────────────┐
                                     │    Redis    │
                                     │  (Cluster)  │
                                     └─────────────┘
```

---

## 10. Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Desktop | Tauri 2, React, TypeScript | 2.x, 18.x, 5.x |
| Backend | Go, Chi, Gorilla WebSocket | 1.26.x, v5.x, v1.x |
| Database | PostgreSQL | 16.x |
| Cache/Session | Redis | 7.x |
| Event Bus | Kafka | 3.x (planned) |
| DI Container | Uber fx | v1.x |
| Cryptography | golang.org/x/crypto, @noble/curves | v0.27.x, v2.x |
| Container | Docker, Docker Compose | Latest |
| Observability | slog, OpenTelemetry, Prometheus | Latest |

---
