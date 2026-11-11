# Zenthril Frontend Architecture
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Frontend Structure

The desktop client is built with Tauri 2, React 18, TypeScript, and Tailwind CSS.

```
client/
├── src/
│   ├── App.tsx                  # Root application component
│   ├── main.tsx                 # Entry point
│   ├── index.css                # Global styles
│   ├── auth.test.ts             # Authentication tests
│   ├── auth.ts                  # Authentication client
│   ├── mood.ts                  # Mood state
│   ├── startupPrivacy.test.ts   # Startup privacy tests
│   ├── startupPrivacy.ts        # Startup privacy logic
│   ├── theme.ts                 # Theme management
│   ├── wsGlobal.ts              # WebSocket global state
│   ├── api/                     # API client layer
│   │   └── index.ts
│   ├── components/              # Reusable UI components
│   │   ├── avatar.tsx
│   │   ├── badge.tsx
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── dialog.tsx
│   │   ├── dropdown-menu.tsx
│   │   ├── input.tsx
│   │   ├── label.tsx
│   │   └── ...
│   ├── config/                  # Configuration
│   │   └── index.ts
│   ├── crypto/                  # Cryptographic utilities
│   │   ├── index.ts
│   │   ├── ratchet.ts           # Symmetric Double Ratchet
│   │   ├── ratchet.test.ts
│   │   └── ratchet.determinism.test.ts
│   ├── features/                # Feature modules
│   │   ├── calls/               # Voice/video calls
│   │   ├── e2ee/                # End-to-end encryption
│   │   ├── p2p/                 # Peer-to-peer
│   │   └── voice/               # Voice chat
│   ├── i18n/                    # Internationalization
│   │   ├── index.ts
│   │   ├── en.json
│   │   ├── ru.json
│   │   └── uk.json
│   ├── lib/                     # Shared libraries
│   │   └── index.ts
│   ├── notifications/           # Notification system
│   │   └── index.ts
│   ├── platform/                # Platform-specific code
│   │   └── index.ts
│   ├── resilience/              # Connection resilience
│   │   ├── connectivityPolicy.ts
│   │   ├── selfHealing.ts
│   │   └── ...
│   ├── store/                   # Zustand state management
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── services/
│   │   ├── tests/
│   │   └── types/
│   ├── transport/               # Transport layer
│   │   └── index.ts
│   ├── types/                   # TypeScript types
│   │   └── index.ts
│   ├── util/                    # Utilities
│   │   └── index.ts
│   └── voice/                   # Voice components
│       └── index.ts
├── src-tauri/                   # Tauri Rust backend
│   ├── src/
│   │   └── main.rs              # Tauri entry point
│   ├── tauri.conf.json          # Tauri configuration
│   └── Cargo.toml               # Rust dependencies
├── dist/                        # Built output
├── public/                      # Public assets
├── package.json                 # Node.js dependencies
├── tsconfig.json                # TypeScript configuration
├── vite.config.ts               # Vite configuration
├── tailwind.config.js           # Tailwind CSS configuration
├── postcss.config.js            # PostCSS configuration
└── eslint.config.js             # ESLint configuration
```

---

## 2. E2EE Client Structure

```
client/src/features/e2ee/
├── index.ts                     # Module exports
├── types.ts                     # E2EE types
├── encoding.ts                  # Binary encoding utilities
├── deviceKeys.ts                # Device key management
├── deviceKeyStore.ts            # Device key storage
├── deviceRegistration.ts        # Device registration
├── deviceVerificationStore.ts   # Device verification state
├── pairwiseSession.ts           # X3DH + Double Ratchet session
├── pairwiseSession.test.ts      # Session tests
├── pairwiseSession.restart.test.ts
├── pairwiseSession.security.test.ts
├── pairwiseSession.skippedKeys.security.test.ts
├── pairwiseSession.trace.test.ts
├── groupSession.ts              # Group session (Sender Keys)
├── groupSession.test.ts
├── messageAAD.ts                # Message Associated Data
├── messageAAD.test.ts
├── messageEnvelopes.ts          # Message envelope format
├── safetyNumber.ts              # Safety number computation
└── safetyNumber.test.ts
```

---

## 3. State Management

**Zustand** is used for client-side state management. Stores are organized in `src/store/`:

| Store | Responsibility |
|-------|---------------|
| `authStore` | Authentication state (user, tokens, session) |
| `deviceStore` | Device management state |
| `messageStore` | Message state (history, drafts) |
| `guildStore` | Guild and channel state |
| `friendStore` | Friend relationship state |
| `e2eeStore` | E2EE session state |
| `connectionStore` | WebSocket connection state |
| `settingsStore` | User settings and preferences |

---

## 4. API Client

The API client uses `fetch` for REST endpoints and `socket.io-client` for WebSocket communication.

**Authentication:** JWT access token stored in HttpOnly cookie (set by backend). Refresh token also in HttpOnly cookie.

**Request Flow:**
```
Client → fetch API → Backend REST API → JSON response
Client → socket.io-client → Backend WebSocket → JSON messages
```

**Error Handling:** Centralized error handler with retry logic for transient failures.

---

## 5. Tauri IPC

Tauri IPC is used for:
- File system access (secure local storage)
- Native notifications
- Window management
- System information

**Security:** IPC commands are validated and restricted to necessary capabilities.

---

## 6. Content Security Policy

Defined in `client/src-tauri/tauri.conf.json`:

```json
"csp": "default-src 'self'; connect-src 'self' http://localhost:8080 ws://localhost:8080 https://api.zenthril.example.com wss://api.zenthril.example.com; img-src 'self' data: https://cdn.zenthril.example.com; style-src 'self'; script-src 'self'"
```

**Issues:**
- No `nonce` or `hash` for inline scripts
- `data:` in img-src could be exploited
- Should be tightened for production

---

## 7. Local Storage

**Current:** E2EE keys and session state stored in Tauri local storage (unencrypted).
**Planned:** Encrypt with key derived from user password via Argon2id.

**Data stored locally:**
- Device key pair (identity, signed prekey, one-time prekeys)
- Pairwise session states (root key, chain keys, DH keys)
- Message history (cached)
- User settings

---

## 8. Internationalization

Three languages supported: English (`en.json`), Russian (`ru.json`), Ukrainian (`uk.json`).

---

## 9. Resilience

Features for connection resilience:
- Dynamic server list (`servers.json`)
- Local cached server list fallback
- Custom server entries
- Automatic retry across configured servers
- DNS-over-HTTPS bootstrap for alternate discovery

---

## 10. Planned Improvements

1. **Encrypted Local Storage** - Encrypt E2EE keys with user password
2. **Safety Numbers UI** - Add UI for comparing key fingerprints
3. **Component Tests** - Add React component rendering tests
4. **Tighten CSP** - Add nonce/hash, remove `data:` from img-src
5. **Offline-First** - Queue messages when offline, sync on reconnect
6. **Push Notifications** - Native desktop notifications for new messages

---
