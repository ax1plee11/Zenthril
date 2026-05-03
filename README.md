# Zenthril

[![CI](https://github.com/ax1plee11/Zenthril/actions/workflows/ci.yml/badge.svg)](https://github.com/ax1plee11/Zenthril/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22-blue.svg)](https://golang.org/)
[![Node Version](https://img.shields.io/badge/node-20-green.svg)](https://nodejs.org/)

**[English](README.md)** | **[Русский](README.ru.md)** | **[Українська](README.uk.md)**

---

A decentralized messenger with federated architecture and end-to-end encryption.

## Tech Stack

- **Backend**: Go + PostgreSQL + Redis
- **Client**: Tauri + Vite + React/TypeScript
- **Encryption**: X25519 + AES-256-GCM (E2EE)
- **Authentication**: JWT + Argon2id
- **UI**: Tailwind CSS + shadcn/ui + Glass Minimal Design
- **i18n**: Multi-language support (EN, RU, UK)

## Features

✨ **Modern Design**
- Glass Minimal UI inspired by Apple/Notion
- Dark theme with glassmorphism effects
- Smooth animations and transitions
- Responsive layout

🌍 **Internationalization**
- Automatic language detection
- Support for English, Russian, Ukrainian
- Easy to add new languages

🔒 **Security & Privacy**
- End-to-end encryption (E2EE)
- Federated architecture
- No tracking or data collection
- Open source

💬 **Communication**
- Text channels and direct messages
- Voice channels (WebRTC)
- GIF support (Tenor/Giphy)
- Real-time messaging (WebSocket)

📊 **Performance & Research**
- Built-in metrics and monitoring (`/metrics` endpoint)
- Comprehensive benchmarking suite
- Load testing tools (k6)
- Performance analysis scripts
- Suitable for academic research and thesis work

## Documentation

- [SECURITY.md](SECURITY.md) — How to report security vulnerabilities
- [docs/PRIVACY.md](docs/PRIVACY.md) — Privacy policy draft for public services
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — Public hosting: TLS, `VITE_API_BASE`, CORS/WS, backups
- [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md) — How to build desktop application (.exe)
- [SHARE.md](SHARE.md) — **How to share Zenthril with friends** 🎉
- [docs/GITHUB_PAGES.md](docs/GITHUB_PAGES.md) — Deploy web version to GitHub Pages
- [research/README.md](research/README.md) — Performance benchmarking and research tools

## Code Quality (Local)

**Backend** (`backend/`):

```bash
go vet ./...
go test ./... -count=1
# go test -race ./...   # on Linux/macOS and Windows amd64; not available on win/386
```

Linter: [golangci-lint](https://golangci-lint.run/) with config `backend/.golangci.yml` (same as in CI).

**Client** (`client/`):

```bash
npm run lint
npm run test
npm run test:coverage
npm run build
```

## Project Structure

```
zenthril/
├── backend/          # Go server (federated network node)
│   ├── auth/         # Authentication & authorization
│   ├── config/       # Configuration via environment variables
│   ├── migrations/   # PostgreSQL SQL migrations
│   ├── metrics/      # Performance monitoring
│   ├── benchmarks/   # Performance benchmarks
│   └── main.go       # HTTP server entry point
├── client/           # Tauri + Vite + React/TypeScript desktop client
│   ├── src/
│   │   ├── components/  # React components
│   │   ├── i18n/        # Internationalization (EN, RU, UK)
│   │   └── store/       # State management
│   └── src-tauri/    # Tauri (Rust) desktop wrapper
├── research/         # Performance research & benchmarking tools
│   ├── load_test/    # Load testing with k6
│   ├── scripts/      # Analysis scripts
│   └── README.md     # Research documentation
├── docs/             # Documentation
├── docker-compose.yml
└── .env.example
```

## Quick Start

```bash
# 1. Copy environment variables
cp .env.example .env

# 2. Start backend services (PostgreSQL + Redis)
docker compose up -d

# 3. Install client dependencies
cd client
npm install

# 4. Copy client environment variables
cp .env.example .env

# 5. Start development server
npm run dev -- --host 0.0.0.0 --port 1420
```

Open `http://localhost:1420/`. Backend will be available at `http://localhost:8080/`.

## Environment Variables

### Backend (repository root)

- `DB_URL` (required) — PostgreSQL connection string
- `REDIS_URL` (default: `redis://localhost:6379`)
- `JWT_SECRET` (required) — Secret key for JWT tokens
- `HTTP_ADDR` (default: `:8080`)
- `CORS_ALLOWED_ORIGINS` (optional) — CORS allowed origins
- `WS_ALLOWED_ORIGINS` (optional) — WebSocket allowed origins
- `ADMIN_USER_IDS` (optional, comma-separated UUIDs) — System administrator access

### Client (`client/` folder)

Copy `client/.env.example` → `client/.env`.

- `VITE_API_BASE` (for production build) — Backend origin, e.g., `https://api.example.com` (see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md))
- `VITE_TENOR_KEY` (optional) — Tenor API key for GIF search
- `VITE_GIPHY_KEY` (optional) — Giphy API key for GIF search

## Building Desktop Application (Windows)

To build the desktop application on Windows, you need **Visual Studio Build Tools** (with `link.exe`):

1. Download [Visual Studio Build Tools 2022](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)
2. Select "Desktop development with C++"
3. Install (~6 GB)
4. Restart your terminal

Then build:

```bash
cd client
npm run tauri build
```

The `.exe` file will be in `client/src-tauri/target/release/bundle/`.

See [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md) for detailed instructions.

## Screenshots

### Glass Minimal Design
![Auth Screen](https://via.placeholder.com/800x500?text=Auth+Screen+with+Language+Switcher)

### Multi-language Support
- 🇬🇧 English
- 🇷🇺 Русский
- 🇺🇦 Українська

Language is automatically detected from browser settings and can be changed manually.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on:
- 🐛 How to report bugs
- 💡 How to suggest features
- 🔒 How to report security vulnerabilities
- 🛠️ How to contribute code
- 🌍 How to add translations

**Contact:** ax1plee@gmail.com

## License

This project is open source. See LICENSE file for details.

## Acknowledgments

- Design inspired by Apple and Notion
- UI components from [shadcn/ui](https://ui.shadcn.com/)
- Icons from [Lucide](https://lucide.dev/)

## Research & Academic Use

Zenthril includes comprehensive performance monitoring and benchmarking tools, making it suitable for:
- Master's thesis and PhD research
- Performance analysis of federated systems
- E2EE encryption overhead studies
- WebSocket scalability research
- Comparative analysis with other messaging platforms

See [research/README.md](research/README.md) for details on:
- Running benchmarks
- Load testing with k6
- Collecting performance metrics
- Analyzing results

### Key Metrics Available:
- WebSocket connection handling (P50, P95, P99 latency)
- Encryption/decryption performance (AES-256-GCM)
- Database query performance
- HTTP request latency
- Real-time monitoring via `/metrics` endpoint
