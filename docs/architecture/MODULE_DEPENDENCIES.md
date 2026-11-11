# Zenthril Module Dependencies
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Go Backend Dependencies

### Direct Dependencies (`backend/go.mod`)

| Package | Version | Purpose | Layer |
|---------|---------|---------|-------|
| `github.com/go-chi/chi/v5` | v5.1.0 | HTTP router | Transport |
| `github.com/golang-jwt/jwt/v5` | v5.2.2 | JWT signing/validation | Security |
| `github.com/google/uuid` | v1.6.0 | UUID generation | Domain |
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket server | Transport |
| `github.com/jackc/pgx/v5` | v5.10.0 | PostgreSQL driver | Infrastructure |
| `github.com/joho/godotenv` | v1.5.1 | .env loading | Infrastructure |
| `github.com/redis/go-redis/v9` | v9.6.3 | Redis client | Infrastructure |
| `github.com/segmentio/kafka-go` | v0.4.47 | Kafka client | Infrastructure |
| `github.com/stretchr/testify` | v1.11.1 | Testing library | Testing |
| `go.uber.org/fx` | v1.24.0 | Dependency injection | Infrastructure |
| `golang.org/x/crypto` | v0.27.0 | Cryptographic primitives | Security/Crypto |

### Indirect Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/cespare/xxhash/v2` | v2.2.0 | Hashing |
| `github.com/davecgh/go-spew` | v1.1.1 | Debug printing (testify dep) |
| `github.com/dgryski/go-rendezvous` | - | Consistent hashing (kafka dep) |
| `github.com/jackc/pgpassfile` | v1.0.0 | PostgreSQL password file |
| `github.com/jackc/pgservicefile` | - | PostgreSQL service file |
| `github.com/jackc/puddle/v2` | v2.2.2 | Connection pooling (pgx dep) |
| `github.com/klauspost/compress` | v1.15.9 | Compression |
| `github.com/pierrec/lz4/v4` | v4.1.15 | LZ4 compression |
| `github.com/pmezard/go-difflib` | v1.0.0 | Diff library (testify dep) |
| `github.com/rogpeppe/go-internal` | v1.16.0 | Internal utilities |
| `go.uber.org/dig` | v1.19.0 | DI container (fx dep) |
| `go.uber.org/multierr` | v1.10.0 | Error handling (fx dep) |
| `go.uber.org/zap` | v1.26.0 | Structured logging (fx dep) |
| `golang.org/x/sync` | v0.21.0 | Concurrency primitives |
| `golang.org/x/sys` | v0.26.0 | System calls |
| `golang.org/x/text` | v0.39.0 | Text processing |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |

---

## 2. TypeScript Frontend Dependencies

### Direct Dependencies (`client/package.json`)

| Package | Version | Purpose |
|---------|---------|---------|
| `@noble/curves` | ^2.0.1 | X25519, Ed25519 cryptography |
| `@noble/hashes` | ^2.2.0 | SHA256, HKDF |
| `@radix-ui/react-avatar` | ^1.1.11 | Avatar component |
| `@radix-ui/react-dialog` | ^1.1.15 | Dialog component |
| `@radix-ui/react-dropdown-menu` | ^2.1.16 | Dropdown menu |
| `@radix-ui/react-label` | ^2.1.8 | Label component |
| `@radix-ui/react-slot` | ^1.2.4 | Slot component |
| `@tauri-apps/api` | ^2.0.0 | Tauri API |
| `@tauri-apps/plugin-shell` | ^2.3.5 | Shell plugin |
| `class-variance-authority` | ^0.7.1 | CSS class utilities |
| `clsx` | ^2.1.1 | CSS class utilities |
| `framer-motion` | ^12.38.0 | Animation library |
| `i18next` | ^26.0.8 | Internationalization |
| `i18next-browser-languagedetector` | ^8.2.1 | Language detection |
| `lucide-react` | ^1.14.0 | Icon library |
| `react` | ^18.3.1 | React framework |
| `react-dom` | ^18.3.1 | React DOM |
| `react-i18next` | ^17.0.6 | React i18n bindings |
| `socket.io-client` | ^4.8.3 | WebSocket client |
| `tailwind-merge` | ^3.5.0 | Tailwind CSS utilities |
| `zustand` | ^5.0.13 | State management |

### Dev Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `@eslint/js` | ^9.39.4 | ESLint |
| `@peculiar/webcrypto` | ^1.5.0 | WebCrypto polyfill |
| `@tauri-apps/cli` | ^2.0.0 | Tauri CLI |
| `@types/react` | ^18.3.3 | React types |
| `@types/react-dom` | ^18.3.0 | React DOM types |
| `@vitejs/plugin-react` | ^6.0.3 | Vite React plugin |
| `@vitest/browser` | ^4.1.10 | Vitest browser |
| `@vitest/coverage-v8` | ^4.1.10 | Coverage |
| `autoprefixer` | ^10.5.0 | PostCSS autoprefixer |
| `eslint` | ^9.39.4 | Linter |
| `eslint-plugin-react-hooks` | ^5.2.0 | React hooks linting |
| `eslint-plugin-react-refresh` | ^0.4.26 | React refresh linting |
| `fast-check` | ^4.6.0 | Property-based testing |
| `globals` | ^15.15.0 | Global variables |
| `happy-dom` | ^20.8.9 | DOM for testing |
| `jsdom` | ^29.0.1 | DOM for testing |
| `postcss` | ^8.5.13 | CSS processor |
| `tailwindcss` | ^3.3.0 | CSS framework |
| `tailwindcss-animate` | ^1.0.7 | Tailwind animations |
| `typescript` | ^5.5.3 | TypeScript compiler |
| `typescript-eslint` | ^8.58.1 | TypeScript ESLint |
| `vite` | ^8.1.0 | Build tool |
| `vitest` | ^4.1.10 | Test runner |

---

## 3. Rust Dependencies (`client/src-tauri/Cargo.toml`)

Tauri 2 Rust backend dependencies (from Cargo.toml):
- `tauri` - Tauri framework
- `tauri-plugin-shell` - Shell command execution

---

## 4. Module Dependency Graph (Go Backend)

```
┌─────────────────────────────────────────────────────────────────┐
│                     main.go (Entrypoint)                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
    ┌─▼─────┐          ┌─▼─────┐          ┌─▼─────┐
    │ auth  │          │ device│          │ guild │
    └─┬─────┘          └─┬─────┘          └─┬─────┘
      │                  │                  │
    ┌─▼─────┐          ┌─▼─────┐          ┌─▼─────┐
    │ hub   │          │ msg   │          │frnds  │
    └─┬─────┘          └─┬─────┘          └─┬─────┘
      │                  │                  │
    ┌─▼─────┐          ┌─▼─────┐          ┌─▼─────┐
    │sec    │          │spam   │          │ mod   │
    └─┬─────┘          └─┬─────┘          └─┬─────┘
      │                  │                  │
    ┌─▼──────────────────▼──────────────────▼─────┐
    │           internal/crypto                   │
    │           internal/config                   │
    │           internal/gateway                  │
    │           internal/app (fx DI)              │
    └─────────────────────────────────────────────┘
                           │
        ┌──────────────────┴──────────────────┐
        │                                     │
    ┌───▼────┐                          ┌─────▼─────┐
    │ pgxpool│                          │ redis     │
    │(Postgre│                          │(Redis 7)  │
    │  SQL)  │                          │           │
    └────────┘                          └───────────┘
```

---

## 5. Package Coupling Analysis

### High Coupling (Should Be Reduced)

| From | To | Reason | Recommendation |
|------|----|--------|----------------|
| `auth` | `pgxpool` | Direct DB access | Repository pattern |
| `message` | `pgxpool` | Direct DB access | Repository pattern |
| `guild` | `pgxpool` | Direct DB access | Repository pattern |
| `hub` | `guildSvc` | Direct service call | Event-based communication |
| `message` | `wsHub` | Direct hub access | Event-based communication |

### Acceptable Coupling

| From | To | Reason |
|------|----|--------|
| `auth` | `redis` | Refresh token storage (infrastructure concern) |
| `security` | `redis` | Rate limiting (infrastructure concern) |
| `internal/crypto` | `golang.org/x/crypto` | Cryptographic primitives |
| `internal/gateway` | `internal/crypto` | Session validation uses crypto |

---

## 6. TypeScript Module Dependencies

```
src/App.tsx
├── src/store/* (Zustand stores)
├── src/api/* (API client)
├── src/transport/wsGlobal.ts (WebSocket)
├── src/features/e2ee/* (E2EE)
│   ├── src/crypto/ratchet.ts
│   └── @noble/curves, @noble/hashes
├── src/features/calls/* (Voice/Video)
├── src/features/p2p/* (P2P)
└── src/features/voice/* (Voice)
```

---

## 7. Circular Dependency Check

**Go Backend:** No circular dependencies detected.

**TypeScript Client:** No circular dependencies detected.

---

## 8. External Service Dependencies

| Service | Purpose | Required | Fallback |
|---------|---------|----------|----------|
| PostgreSQL 16 | Primary database | Yes | None |
| Redis 7 | Sessions, rate limiting | Yes | In-memory (degraded) |
| Kafka | Event bus | No (planned) | Memory bus |
| Docker | Containerization | No (dev) | Direct binary |

---
