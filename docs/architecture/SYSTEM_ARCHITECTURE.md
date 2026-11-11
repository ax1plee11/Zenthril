# Zenthril System Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. System Overview

Zenthril is a self-hosted secure messaging platform consisting of:

1. **Backend Server** (Go) - HTTP API + WebSocket gateway
2. **Desktop Client** (Tauri 2 + React + TypeScript) - Desktop application
3. **PostgreSQL 16** - Primary relational database
4. **Redis 7** - Session store, rate limiting, cache, pub/sub

---

## 2. Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     DESKTOP CLIENT (Tauri 2)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐    │
│  │ React UI    │  │ Zustand     │  │ Tauri IPC / Shell   │    │
│  │ (Components)│  │ (State)     │  │ (File system, keys) │    │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘    │
│         │                │                     │               │
│         └────────────────┴─────────────────────┘               │
│                     WebSocket (socket.io-client)                │
│                         HTTPS (fetch API)                       │
└─────────────────────────────────────────────────────────────────┘
                                      │
                                      │ WSS/HTTPS
                                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    BACKEND SERVER (Go)                          │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Chi HTTP Router                                          │   │
│  │  ├─ /api/v1/auth      (Auth handlers)                    │   │
│  │  ├─ /api/v1/devices   (Device handlers)                  │   │
│  │  ├─ /api/v1/guilds    (Guild handlers)                   │   │
│  │  ├─ /api/v1/channels  (Channel handlers)                 │   │
│  │  ├─ /api/v1/messages  (Message handlers)                 │   │
│  │  ├─ /api/v1/friends   (Friend handlers)                  │   │
│  │  ├─ /api/v1/users     (User handlers)                    │   │
│  │  ├─ /api/v1/admin     (Admin handlers)                   │   │
│  │  └─ /ws               (WebSocket upgrade)                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│  ┌─────────────────────────┴──────────────────────────────┐   │
│  │  Security Middleware (Rate limiting, CORS, Auth)       │   │
│  │  Service Layer (Auth, Device, Message, Guild, Friends) │   │
│  │  Repository Layer (PostgreSQL access)                  │   │
│  └────────────────────────────────────────────────────────┘   │
│                              │                                   │
│  ┌─────────────────────────┴──────────────────────────────┐   │
│  │  PostgreSQL 16  │  Redis 7                             │   │
│  │  (Users, Guilds,│  (Sessions, Rate limits,               │   │
│  │   Messages, etc)│   Cache, Pub/Sub)                      │   │
│  └─────────────────┴───────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Communication Protocols

| Path | Protocol | Port | Authentication | Encryption |
|------|----------|------|---------------|------------|
| Client → Backend API | HTTPS | 8080 | JWT (cookie/header) | TLS |
| Client → Backend WS | WSS | 8080 | WebSocket ticket | TLS |
| Backend → PostgreSQL | TCP | 5432 | Password | Optional TLS |
| Backend → Redis | TCP | 6379 | Password | Optional TLS |
| Backend → Backend (federation) | HTTPS | 8080 | Federation token | TLS |

---

## 4. Deployment Topology

### Single-Node (Recommended for Self-Hosted)

All components run on a single host via Docker Compose:

```
Host Machine
├── Docker Compose
│   ├── postgres:16 (port 5432, internal network)
│   ├── redis:7 (port 6379, internal network)
│   └── backend:zenthril (port 8080, internal network)
└── Client connects via public reverse proxy (nginx/TLS)
```

### Multi-Node (Production Scale)

```
Internet → Load Balancer (TLS termination) → Backend Nodes → Shared PostgreSQL + Redis Cluster
```

---

## 5. Environment Configuration

| Variable | Description | Default | Required in Production |
|----------|-------------|---------|----------------------|
| `HTTP_ADDR` | HTTP listen address | `:8080` | Yes |
| `DB_URL` | PostgreSQL connection string | - | Yes |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` | Yes |
| `JWT_SECRET` | JWT signing secret (min 32 bytes) | - | Yes |
| `OPERATIONAL_TOKEN` | Health/metrics endpoint token | - | Yes |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | - | Yes |
| `WS_ALLOWED_ORIGINS` | Allowed WebSocket origins | - | Yes |
| `ENVIRONMENT` | `development`, `production` | `development` | Yes |
| `SECURE_COOKIES` | Use Secure/SameSite=None cookies | `true` | Yes |
| `METRICS_TOKEN` | Metrics endpoint token | - | Yes |
| `FEDERATION_ENABLED` | Enable federation | `false` | Optional |
| `FEDERATION_TOKEN` | Federation authentication token | - | If federation enabled |
| `TRUSTED_PROXY_HOPS` | Number of trusted reverse proxies | `0` | If behind proxy |
| `EVENT_BUS_DRIVER` | Event bus driver (`memory`, `kafka`, `nats`) | `memory` | `kafka`/`nats` in prod |

---

## 6. Data Flow Summary

### User Registration
```
Client: POST /api/v1/auth/register
  → Backend: Validate password complexity
  → Backend: Hash password with Argon2id
  → Backend: Insert user into PostgreSQL
  → Backend: Generate JWT access + refresh tokens
  → Backend: Store refresh token hash in Redis
  → Client: Set HttpOnly cookies, return user data
```

### Message Send (with E2EE)
```
Client: Encrypt message with Double Ratchet key
Client: POST /api/v1/channels/{id}/messages (encrypted payload)
  → Backend: Validate authentication
  → Backend: Store encrypted message in PostgreSQL
  → Backend: Publish to Redis for fan-out
  → Backend: WebSocket broadcast to online recipients
  → Client: Decrypt message with local session key
```

### Device Registration
```
Client: Generate X25519 identity key pair
Client: POST /api/v1/devices/register (public key, device info)
  → Backend: Store device key bundle in PostgreSQL
  → Client: Store private key in secure local storage
```

---

## 7. Failure Modes

| Failure | Detection | Mitigation |
|---------|-----------|------------|
| PostgreSQL down | Readiness probe fails | Return 503, queue operations |
| Redis down | Readiness probe fails | Rate limiting disabled, sessions in memory |
| WebSocket disconnect | Ping/pong timeout | Auto-reconnect with ticket |
| JWT expired | 401 response | Refresh token rotation |
| Refresh token replay | Redis GetDel returns nil | Revoke all user refresh tokens |
| E2EE key mismatch | Decryption failure | Re-bootstrap via X3DH |

---
