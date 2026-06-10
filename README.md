# Zenthril

**Zenthril** is an open-source, self-hosted messaging platform focused on realtime communication, privacy, and practical security engineering.

> **Security Status: Alpha**
>
> Zenthril is an alpha-stage secure messaging project.
> Do not use it for sensitive or high-risk communication yet.
> The current E2EE implementation is experimental and has not been externally audited.
> It is not equivalent to the Signal Protocol.
> X3DH, full Double Ratchet, skipped-message-key handling, production-grade device verification, secure recovery, and external audit are still required before production security claims.

## Security Status

| Area | Status |
| --- | --- |
| Auth hardening | Partial / Alpha |
| Refresh token rotation | Implemented |
| WebSocket tickets | Implemented |
| E2EE HKDF envelope | Implemented foundation |
| X3DH | Incomplete / WIP |
| Double Ratchet | Incomplete / WIP |
| Device verification | Incomplete |
| Secure key storage | Incomplete |
| External audit | Not done |
| Federation security | Not production-ready |

The project is developed as a practical self-hosted product for small communities while the security and reliability foundations mature.

## Status

> **Alpha / early development stage**
>
> Zenthril is raw, incomplete, and still a work in progress. It is **not production ready** and should not be used for sensitive communication, public deployments, or untrusted environments at this stage.

The current codebase is suitable for local development, controlled self-hosted experiments, and small trusted testing groups. It is not yet suitable for real organizations, public communities, or users who expect mature reliability, audited cryptography, or production-grade federation.

This repository should be read as an engineering alpha: useful, testable, and actively improving, but still rough in important areas.

> **Note:** Federation is planned and partially scaffolded but is not production-ready.
> All federation endpoints are disabled by default (`FEDERATION_ENABLED=false`).

## Documentation Source of Truth

Project documentation lives in this repository: [`README.md`](README.md),
[`SECURITY.md`](SECURITY.md), [`THREAT_MODEL.md`](THREAT_MODEL.md), and
[`docs/`](docs/). Do not rely on a separate GitHub Wiki for current project
state, because it can drift from the code. Documentation changes should be
committed together with the code or configuration they describe.

## Known Limitations / Current Issues

- **Not production ready:** deployment, monitoring, incident response, key storage, and operational procedures still require more work.
- **E2EE is incomplete:** Zenthril has foundational cryptographic components, but it does not yet implement a complete Signal-grade protocol.
- **Double Ratchet is still in progress:** forward secrecy, skipped message key handling, session healing, and full multi-device behavior are not production complete.
- **Federation is not ready:** federation endpoints are alpha-level, disabled by default, and should not be described as a finished decentralized protocol.
- **Scalability is still being validated:** benchmark results are useful research data, not a guarantee of real-world performance under hostile or large-scale workloads.
- **Desktop key storage needs hardening:** production desktop builds should move private key material to OS keychain or a stronger storage mechanism.
- **Startup privacy is conservative by default:** saved sessions open offline until the user explicitly connects, unless auto-connect is enabled.
- **Voice and realtime features are experimental:** hybrid voice and WebSocket gateway work exist, but edge cases and abuse resistance still need more testing.
- **External security audit has not been completed:** do not treat the project as audited secure software.

## Key Features

### Implemented

- WebSocket-based realtime messaging
- Guilds, channels, invites, and basic moderation
- RBAC roles foundation with multiple roles per guild member
- JWT authentication with refresh token support
- Device registration and key revocation foundation
- Basic E2EE building blocks using X25519, HKDF-SHA256, and AES-256-GCM
- Hybrid voice foundation with P2P / mesh / SFU-oriented architecture
- Strict CORS and WebSocket Origin validation
- Security headers, protected metrics, and production configuration validation
- PostgreSQL and Redis backed backend services
- Tauri desktop client with React and TypeScript
- Privacy-first startup mode: saved sessions do not contact the server until explicit Connect by default
- Multilingual UI foundation: English, Russian, and Ukrainian
- Benchmark and research materials under [`research/`](research/)

### In Progress

- Full Double Ratchet integration into real message flows
- X3DH-style session setup and stronger multi-device E2EE semantics
- Production-grade private key storage for the desktop client
- Multi-instance WebSocket gateway behavior and distributed fan-out
- Federation protocol design and inter-node trust model
- Better observability with OpenTelemetry tracing and expanded metrics
- More complete load testing, security testing, and operator documentation

## Security & E2EE

Security is a major focus of Zenthril, but the current implementation must be treated carefully.

**Current security-related work includes:**

- Argon2id password hashing
- Short-lived JWT access tokens
- Refresh tokens with server-side tracking and revocation
- Redis-backed access token blacklist support
- Strict CORS allowlist validation
- Strict WebSocket Origin validation to reduce CSWSH risk
- Message size limits and basic WebSocket flood protection
- Security headers including CSP, HSTS, frame protection, MIME sniffing protection, Referrer Policy, and Permissions Policy
- Protected metrics and operational endpoints in production mode
- Production configuration validation for secrets and allowed origins

**Important E2EE disclaimer:**

Zenthril currently provides **E2EE foundations**, not a finished, audited, Signal-grade encryption system. The project does not yet provide complete Double Ratchet behavior, robust recovery after compromise, mature multi-device session handling, or independently reviewed cryptographic guarantees.

Do **not** use Zenthril for highly sensitive communication at this stage. Treat the current E2EE layer as a work in progress and as research-oriented engineering, not as a security promise.

Saved sessions use a privacy-first startup mode: by default the client opens
locally and does not load guilds, request WebSocket tickets, or open realtime
connections until the user presses **Connect**. Users can explicitly enable
auto-connect in the offline startup screen.

See [`SECURITY.md`](SECURITY.md), [`THREAT_MODEL.md`](THREAT_MODEL.md), [`docs/SECURITY_HARDENING.md`](docs/SECURITY_HARDENING.md), and [`docs/security/E2EE_FOUNDATION.md`](docs/security/E2EE_FOUNDATION.md) for more detail.

## Connectivity Resilience

Zenthril now includes an alpha multi-server client foundation for self-hosted
outage recovery:

- dynamic `servers.json` server list;
- local cached server list fallback;
- custom server entries in the client;
- administrator-configured backup endpoints promoted into retry targets;
- automatic retry across configured servers when network errors occur.
- DNS-over-HTTPS bootstrap for alternate server-list discovery;
- explicit custom server selection.

This is a connectivity resilience foundation, not a guarantee of invisible or
indistinguishable traffic. See
[`docs/CONNECTIVITY_RESILIENCE.md`](docs/CONNECTIVITY_RESILIENCE.md) for the
current design and limitations.

## Technical Stack

**Backend**

- Go 1.23+
- Chi router
- PostgreSQL
- Redis
- Gorilla WebSocket
- Prometheus-compatible metrics

**Client**

- Tauri 2
- React
- TypeScript
- Tailwind CSS
- shadcn/ui

**Research / Infrastructure**

- Go benchmarks
- k6 load testing
- Docker Compose deployment files
- Kubernetes-oriented deployment manifests
- ADRs and research documentation

## Performance Research

Zenthril includes local benchmark and load-test materials under [`research/`](research/). Any performance numbers published in this repository should be treated as preliminary local benchmark data unless they include hardware, methodology, commit/date, and raw results.

Benchmark results do **not** imply production readiness. Real deployment behavior depends on TLS termination, network conditions, database load, Redis behavior, storage latency, abuse traffic, and multi-node fan-out design.

## For Developers

### Quick Start

Requirements:

- Go 1.23+
- Node.js and npm
- PostgreSQL
- Redis
- Tauri prerequisites for desktop development

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

cp .env.example .env
# Edit .env before running:
# - set a strong JWT_SECRET
# - configure DB_URL and REDIS_URL
# - set CORS_ALLOWED_ORIGINS and WS_ALLOWED_ORIGINS
```

Run the backend:

```bash
cd backend
go run .
```

Run the client:

```bash
cd client
npm install
npm run tauri dev
```

Run tests:

```bash
cd backend
go test ./...

cd ../client
npm test
```

## Usage Policy & Restrictions

Zenthril is provided for lawful software development, education, experimentation, and research.

You may **not** use this project for illegal activity, fraud, spam, harassment, credential theft, malware distribution, unauthorized surveillance, abuse of platform moderation systems, or any other harmful purpose. The maintainer does not endorse or authorize such use.

Students and researchers are welcome to study the code, run experiments, and learn from the architecture. However, if you copy substantial parts of this project into coursework, diploma work, theses, articles, presentations, or derivative repositories, you must clearly cite the original repository and author.

Suggested attribution:

```text
Based on Zenthril by ax1plee11:
https://github.com/ax1plee11/Zenthril
```

This section is project guidance and attribution policy. The legal license terms are defined in [`LICENSE`](LICENSE).

For the full attribution and acceptable-use policy, see [`NOTICE`](NOTICE) and
[`docs/USAGE_POLICY.md`](docs/USAGE_POLICY.md).

## Contributing

Contributions are welcome, especially in areas where the project is currently raw and incomplete:

- Security review
- Cryptographic protocol review
- Double Ratchet and device-session implementation
- WebSocket gateway scalability
- Abuse prevention and moderation tooling
- Benchmark design and reproducible research
- Documentation and translation improvements

Please open an Issue before making large architectural or security-sensitive changes.

## License

See [`LICENSE`](LICENSE) for legal terms and [`NOTICE`](NOTICE) for attribution
and project provenance.
