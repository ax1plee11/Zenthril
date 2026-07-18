# Zenthril

**[Русский](README.ru.md)** | **[English](README.md)** | **[Українська](README.uk.md)**

**Zenthril** — open-source self-hosted платформа обміну повідомленнями, сфокусована на безпеці, продуктивності та дослідженні realtime communication systems.

Проєкт розробляється студентом випускного курсу Software Engineering як практичний застосунок і академічний інженерний проєкт.

## Статус

**Alpha Stage**

Проєкт активно розвивається. Core messaging, hybrid voice і foundational E2EE components уже реалізовані, але **Zenthril поки не рекомендований для production-використання в недовірених середовищах**.

На цьому етапі проєкт підходить для локальної розробки, controlled self-hosted testing і академічних досліджень.

## Ключові можливості

- Realtime messaging через WebSocket.
- Guilds і channels з базовою модерацією.
- Hybrid Voice system: P2P + Mesh + SFU з fallback-логікою.
- Device management і key revocation foundation.
- JWT authentication з refresh token support.
- Multi-language interface: EN / RU / UK.

## Технічний стек

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

- Go benchmarks.
- k6 load testing.
- Prometheus-compatible metrics endpoint.

## Архітектура

- Модульний Go backend з authentication, guilds, messaging, device keys, voice і gateway foundations.
- WebSocket realtime layer з ongoing security hardening.
- PostgreSQL для relational data і Redis для cache/session-oriented infrastructure.
- Tauri desktop client на React і TypeScript.
- Research artifacts і benchmark results у [`research/`](research/).

## Безпека та E2EE

Безпека — один із головних пріоритетів проєкту.

**Уже реалізовано**

- Argon2id password hashing.
- JWT з refresh tokens.
- Device key management і revocation.
- Basic E2EE foundation: X25519 + AES-256-GCM.

**Важливий disclaimer**

Zenthril зараз реалізує **foundational E2EE components**, а не повний Signal-grade protocol. Double Ratchet, повноцінна forward secrecy, robust session healing і зріла multi-device модель ще перебувають у розробці.

**Не використовуйте проєкт для високочутливих комунікацій на цій стадії.**

Поточні security controls описані в [`docs/SECURITY_HARDENING.md`](docs/SECURITY_HARDENING.md).

## Performance Research

Усі опубліковані performance numbers вимірювалися в controlled benchmark environments.

**Приклад результатів: single node, Intel Core i7-12700K, 32GB RAM**

- Peak WebSocket throughput: близько 14,800 messages/sec.
- P95 latency при 500 concurrent users: близько 98ms.

Методологія, scripts і raw results знаходяться в [`research/`](research/). Академічний план описано в [`docs/ACADEMIC_RESEARCH_PLAN.md`](docs/ACADEMIC_RESEARCH_PLAN.md).

## Roadmap

Короткий напрям:

- Security hardening і E2EE protocol maturity.
- WebSocket gateway layer для multi-instance deployments.
- Production deployment improvements.
- Basic federation лише після проєктування безпечного protocol.
- Comparative research і матеріали для дипломної/магістерської роботи.

Детальна дорожня карта: [`docs/ROADMAP.uk.md`](docs/ROADMAP.uk.md).

## Для розробників

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

cp .env.example .env
# Налаштуйте сильні секрети та allowed origins
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

## Contribution

Contributions вітаються. Пріоритетні напрями:

- Security review і cryptographic improvements.
- E2EE protocol development і testing.
- Performance analysis і benchmarking.
- Documentation.

Для великих змін спочатку відкрийте Issue.

## License

MIT License. Див. [`LICENSE`](LICENSE).
## Documentation Source of Truth

Current documentation is kept in the repository: [`README.md`](README.md), [`SECURITY.md`](SECURITY.md), [`THREAT_MODEL.md`](THREAT_MODEL.md), and [`docs/`](docs/). Do not rely on a separate GitHub Wiki for current project state, because it can drift from the code.

