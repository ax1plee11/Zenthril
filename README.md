<div align="center">

<img src="https://raw.githubusercontent.com/ax1plee11/Zenthril/main/client/src-tauri/icons/icon.png" width="96" height="96" alt="Zenthril" />

# Zenthril

**Next-generation realtime communication platform.**

*Ultra-lightweight · Privacy-first · End-to-end encrypted · Open source*

<br/>

[![CI](https://github.com/ax1plee11/Zenthril/actions/workflows/ci.yml/badge.svg)](https://github.com/ax1plee11/Zenthril/actions/workflows/ci.yml)
[![Deploy](https://github.com/ax1plee11/Zenthril/actions/workflows/deploy-pages.yml/badge.svg)](https://ax1plee11.github.io/Zenthril/)
[![License: MIT](https://img.shields.io/badge/license-MIT-8b9dff.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.5-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Tauri](https://img.shields.io/badge/Tauri-2.0-FFC131?logo=tauri&logoColor=white)](https://tauri.app/)
[![Tests](https://img.shields.io/badge/tests-70%20passing-22c55e.svg)](#)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-8b9dff.svg)](CONTRIBUTING.md)

<br/>

[**Live Demo**](https://ax1plee11.github.io/Zenthril/) · [**Download**](#-desktop-releases) · [**Docs**](docs/) · [**Contributing**](CONTRIBUTING.md)

<br/>

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                   │
│   Zenthril  ·  Realtime  ·  Encrypted  ·  Federated             │
│                                                                   │
│   P2P Voice  ──►  Mesh  ──►  SFU   (auto-switching)             │
│   AES-256-GCM  ·  X25519  ·  Argon2id  ·  JWT                   │
│   14,823 msg/s  ·  P95 < 100ms  ·  1000+ concurrent users       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

</div>

---

## Why Zenthril?

> Most communication platforms are either too heavy, closed-source, or compromise your privacy.  
> Zenthril is built differently — lightweight, encrypted by default, and fully open.

| | Zenthril | Discord | Matrix Synapse | XMPP |
|---|---|---|---|---|
| **E2EE by default** | ✅ | ❌ | ✅ | ✅ |
| **Self-hosted** | ✅ | ❌ | ✅ | ✅ |
| **P95 latency** | **98ms** | ~120ms | 312ms | 187ms |
| **Throughput** | **14,823 msg/s** | — | 3,200 | 5,600 |
| **Memory/conn** | **48 KB** | — | 187 KB | 89 KB |
| **Desktop app** | ✅ | ✅ | ❌ | ❌ |
| **Open source** | ✅ | ❌ | ✅ | ✅ |

---

## Key Features

<table>
<tr>
<td width="50%">

### 🔒 Privacy First
- **End-to-end encryption** — X25519 + AES-256-GCM
- **Zero tracking** — no analytics, no telemetry
- **Self-hosted** — your data, your server
- **Argon2id** password hashing

</td>
<td width="50%">

### ⚡ Realtime First
- **WebSocket** with sub-100ms latency
- **14,823 messages/second** throughput
- **1,000+ concurrent users** per node
- **Optimistic UI** updates

</td>
</tr>
<tr>
<td width="50%">

### 🎙️ Hybrid Voice System
- **P2P** for 1-on-1 calls (lowest latency)
- **Mesh** for small rooms (3–6 users)
- **SFU** for large rooms (7+ users)
- **Auto-switching** between modes

</td>
<td width="50%">

### 🖥️ Cross-Platform Desktop
- **Windows** — NSIS installer + portable .exe
- **macOS** — .dmg bundle
- **Linux** — .deb + .AppImage
- Built with **Tauri** (Rust + WebView)

</td>
</tr>
<tr>
<td width="50%">

### 🌍 Internationalization
- **English**, **Русский**, **Українська**
- Auto-detection from browser
- Easy to add new languages

</td>
<td width="50%">

### 📊 Built-in Observability
- **Prometheus** metrics endpoint
- **P50/P95/P99** latency tracking
- **k6** load testing suite
- Research-grade benchmarks

</td>
</tr>
</table>

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        Zenthril Stack                         │
├──────────────────┬───────────────────────────────────────────┤
│   Desktop Client │  Tauri 2 + React 18 + TypeScript 5        │
│                  │  Tailwind CSS + Framer Motion + Zustand    │
│                  │  WebRTC (P2P / Mesh / SFU auto-switch)     │
├──────────────────┼───────────────────────────────────────────┤
│   Backend        │  Go 1.22 — single binary, ~15MB           │
│                  │  Chi router + WebSocket hub                │
│                  │  PostgreSQL 16 + Redis 7                   │
├──────────────────┼───────────────────────────────────────────┤
│   Encryption     │  X25519 ECDH key exchange                  │
│                  │  AES-256-GCM message encryption            │
│                  │  Argon2id password hashing                 │
├──────────────────┼───────────────────────────────────────────┤
│   Infrastructure │  Docker Compose (dev)                      │
│                  │  Railway / Fly.io (production)             │
│                  │  Kubernetes manifests included             │
└──────────────────┴───────────────────────────────────────────┘
```

### Voice Architecture

```
Participants ≤ 2  →  P2P Direct     (lowest latency, no server)
Participants ≤ 6  →  Hybrid Mesh    (partial relay, balanced)
Participants  7+  →  SFU Server     (scalable, single upload)
```

---

## Quick Start

### Option 1 — Docker (recommended)

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril
cp .env.example .env          # edit JWT_SECRET
docker compose up -d
```

Open `http://localhost:1420` — backend at `http://localhost:8080`.

### Option 2 — Manual

```bash
# Backend
cd backend
go run .

# Client (new terminal)
cd client
npm install
npm run dev
```

### Environment Variables

```env
DB_URL=postgres://zenthril:zenthril@localhost:5432/zenthril
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key-min-32-chars
HTTP_ADDR=:8080
```

---

## Desktop Releases

Download the latest release for your platform:

| Platform | Format | Link |
|----------|--------|------|
| 🪟 Windows | `.exe` installer | [Releases →](https://github.com/ax1plee11/Zenthril/releases) |
| 🪟 Windows | `.msi` package | [Releases →](https://github.com/ax1plee11/Zenthril/releases) |
| 🐧 Linux | `.AppImage` | [Releases →](https://github.com/ax1plee11/Zenthril/releases) |
| 🐧 Linux | `.deb` package | [Releases →](https://github.com/ax1plee11/Zenthril/releases) |
| 🍎 macOS | `.dmg` bundle | [Releases →](https://github.com/ax1plee11/Zenthril/releases) |

Or build from source — see [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md).

---

## Performance

Benchmarked on Intel Core i7-12700K, 32GB RAM, NVMe SSD:

```
WebSocket Throughput    14,823 msg/sec  (peak)
P50 latency @ 500 users    31ms
P95 latency @ 500 users    98ms   ← target: < 100ms ✅
P99 latency @ 500 users   187ms
Database INSERT             0.2ms  avg
Database SELECT (indexed)   1.9ms  avg
AES-256-GCM encryption      0.8ms  per 1KB message
Max concurrent users      1,000+  single node
Memory per connection        48KB
```

Full benchmark results: [`research/results/`](research/results/)

---

## Roadmap

- [x] E2EE messaging (X25519 + AES-256-GCM)
- [x] WebSocket real-time hub
- [x] Voice calls (P2P / Mesh / SFU)
- [x] Desktop app (Windows / Linux / macOS)
- [x] Internationalization (EN / RU / UK)
- [x] Performance benchmarks & metrics
- [ ] Mobile app (iOS / Android)
- [ ] Spatial audio
- [ ] AI noise suppression
- [ ] Plugin system
- [ ] Federation protocol
- [ ] Screen sharing
- [ ] File transfers (E2EE)

---

## Project Structure

```
zenthril/
├── backend/              # Go server — single binary
│   ├── auth/             # JWT + Argon2id authentication
│   ├── guild/            # Servers, channels, roles
│   ├── message/          # E2EE message handling
│   ├── hub/              # WebSocket connection hub
│   ├── friends/          # Friend system
│   ├── metrics/          # Prometheus metrics
│   ├── security/         # DDoS + brute-force protection
│   └── spam/             # Rate limiting (Token Bucket)
├── client/               # Tauri + React desktop app
│   └── src/
│       ├── components/   # Shared UI components
│       ├── features/
│       │   ├── calls/    # 1:1 voice calls (WebRTC)
│       │   └── voice/    # Group voice rooms (P2P/Mesh/SFU)
│       ├── crypto/       # E2EE implementation
│       └── store/        # Zustand state management
├── deployments/          # Docker, Kubernetes, Caddy configs
├── research/             # Benchmarks, load tests, results
└── docs/                 # Architecture, deployment guides
```

---

## Open Source Philosophy

Zenthril is built on the belief that **communication infrastructure should be open, auditable, and privacy-respecting**.

- **No vendor lock-in** — self-host on any server
- **No black boxes** — every line of code is readable
- **No surveillance** — zero telemetry, zero tracking
- **Community-driven** — your contributions shape the roadmap

---

## Contributing

We welcome contributions of all kinds.

```bash
# Fork → Clone → Branch → Code → Test → PR
git checkout -b feat/your-feature
go test ./...          # backend
npm test               # frontend
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

**Good first issues:** look for the [`good first issue`](https://github.com/ax1plee11/Zenthril/issues?q=label%3A%22good+first+issue%22) label.

---

## Community

| | |
|---|---|
| 💬 **Discussions** | [GitHub Discussions](https://github.com/ax1plee11/Zenthril/discussions) |
| 🐛 **Bug Reports** | [GitHub Issues](https://github.com/ax1plee11/Zenthril/issues) |
| 🔒 **Security** | ax1plee@gmail.com |
| 📧 **Contact** | ax1plee@gmail.com |

---

## Tech Stack

<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![React](https://img.shields.io/badge/React-61DAFB?style=for-the-badge&logo=react&logoColor=black)
![Tauri](https://img.shields.io/badge/Tauri-FFC131?style=for-the-badge&logo=tauri&logoColor=black)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Tailwind](https://img.shields.io/badge/Tailwind-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)

</div>

---

## License

MIT © [Zenthril Contributors](https://github.com/ax1plee11/Zenthril/graphs/contributors)

---

<div align="center">

**Built with ❤️ for the open-source community**

[⭐ Star this repo](https://github.com/ax1plee11/Zenthril) · [🍴 Fork it](https://github.com/ax1plee11/Zenthril/fork) · [📢 Share it](SHARE.md)

</div>
