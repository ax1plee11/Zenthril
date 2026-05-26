# Zenthril

**Zenthril** is an open-source self-hosted messaging platform focused on security, performance, and research in real-time communication systems.

Developed by a final-year Software Engineering student as both a practical application and an academic project.

## Status

**Alpha Stage**

The project is under active development. Core messaging, hybrid voice communication, and foundational E2EE components are implemented, but **Zenthril is not yet recommended for production use in untrusted environments**.

It is currently suitable for local development, self-hosted testing in controlled environments, and academic research.

## Key Features

- Real-time messaging via WebSocket
- Guilds and channels with basic moderation
- Hybrid Voice system (P2P + Mesh + SFU with automatic switching)
- Device management and key revocation
- JWT authentication with refresh token support
- Multi-language support (EN / RU / UK)

## Technical Stack

**Backend**

- Go 1.23+
- Chi router
- PostgreSQL + Redis
- Gorilla WebSocket

**Client**

- Tauri 2
- React + TypeScript
- Tailwind CSS + shadcn/ui

**Research & Observability**

- Go benchmarks
- k6 load testing
- Prometheus-compatible metrics endpoint

## Architecture Highlights

- Modular Go backend with authentication, guilds, messaging, device keys, voice, and gateway foundations
- WebSocket-based realtime communication with ongoing hardening work
- PostgreSQL for core relational data and Redis for cache/session-oriented infrastructure
- Tauri desktop client built with React and TypeScript
- Research-oriented benchmark and measurement artifacts under [`research/`](research/)

## Security & Encryption

Security is one of the main priorities of the project.

**Implemented**

- Argon2id password hashing
- JWT with refresh tokens
- Device key management and revocation
- Basic E2EE foundation (X25519 + AES-256-GCM)

**Important Disclaimer**

Zenthril currently implements **foundational E2EE components**. A full Signal-grade protocol including Double Ratchet (forward secrecy), complete session healing, and advanced multi-device support is still under active development.

**Do not use for highly sensitive communications at this stage.**

## Performance Research

All published performance figures were measured in controlled benchmark environments.

**Example results (single node, Intel Core i7-12700K, 32GB RAM):**

- Peak WebSocket throughput: ~14,800 messages/sec
- P95 latency at 500 concurrent users: ~98ms

Detailed methodology, scripts, and raw results are available in the [`research/`](research/) directory.

## Roadmap

**Phase 1 (In Progress)**

- Security hardening
- Full E2EE protocol implementation (Double Ratchet)
- WebSocket gateway layer for multi-instance deployments

**Phase 2**

- Basic federation support
- Production deployment improvements
- Enhanced observability

**Phase 3**

- Advanced scaling and sharding
- Mobile / PWA client
- Comparative research and publications

## For Developers

### Quick Start

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

cp .env.example .env
# Configure strong secrets and allowed origins
```

Backend:

```bash
cd backend
go run .
```

Client:

```bash
cd client
npm install
npm run tauri dev
```

## Contributing

Contributions are welcome. Priority areas currently include:

- Security review and cryptographic improvements
- E2EE protocol development and testing
- Performance analysis and benchmarking
- Documentation

Please open an Issue first for significant changes.

## License

MIT License. See [`LICENSE`](LICENSE) for details.
