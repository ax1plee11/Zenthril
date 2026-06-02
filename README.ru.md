# Zenthril

**[English](README.md)** | **[Русский](README.ru.md)** | **[Українська](README.uk.md)**

**Zenthril** — open-source self-hosted мессенджер, сфокусированный на безопасности, производительности и исследовании realtime communication systems.

Проект разрабатывается студентом выпускного курса Software Engineering как практическое приложение и академический инженерный проект.

## Статус

**Alpha Stage**

Проект активно развивается. Core messaging, hybrid voice и foundational E2EE components уже реализованы, но **Zenthril пока не рекомендуется для production-использования в недоверенных окружениях**.

На текущем этапе проект подходит для локальной разработки, controlled self-hosted testing и академических исследований.

## Ключевые возможности

- Realtime messaging через WebSocket.
- Guilds и channels с базовой модерацией.
- Hybrid Voice system: P2P + Mesh + SFU с fallback-логикой.
- Device management и key revocation foundation.
- JWT authentication с refresh token support.
- Multi-language interface: EN / RU / UK.

## Технический стек

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

## Архитектура

- Модульный Go backend с authentication, guilds, messaging, device keys, voice и gateway foundations.
- WebSocket realtime layer с ongoing security hardening.
- PostgreSQL для relational data и Redis для cache/session-oriented infrastructure.
- Tauri desktop client на React и TypeScript.
- Research artifacts и benchmark results в [`research/`](research/).

## Безопасность и E2EE

Безопасность — один из главных приоритетов проекта.

**Уже реализовано**

- Argon2id password hashing.
- JWT с refresh tokens.
- Device key management и revocation.
- Basic E2EE foundation: X25519 + AES-256-GCM.

**Важный disclaimer**

Zenthril сейчас реализует **foundational E2EE components**, а не полный Signal-grade protocol. Double Ratchet, полноценная forward secrecy, robust session healing и зрелая multi-device модель еще находятся в разработке.

**Не используйте проект для высокочувствительных коммуникаций на этой стадии.**

Текущие security controls описаны в [`docs/SECURITY_HARDENING.md`](docs/SECURITY_HARDENING.md).

## Performance Research

Все опубликованные performance numbers измерялись в controlled benchmark environments.

**Пример результатов: single node, Intel Core i7-12700K, 32GB RAM**

- Peak WebSocket throughput: около 14,800 messages/sec.
- P95 latency при 500 concurrent users: около 98ms.

Методология, scripts и raw results находятся в [`research/`](research/). Академический план описан в [`docs/ACADEMIC_RESEARCH_PLAN.md`](docs/ACADEMIC_RESEARCH_PLAN.md).

## Roadmap

Краткое направление:

- Security hardening и E2EE protocol maturity.
- WebSocket gateway layer для multi-instance deployments.
- Production deployment improvements.
- Basic federation только после проектирования безопасного protocol.
- Comparative research и материалы для дипломной/магистерской работы.

Подробная дорожная карта: [`docs/ROADMAP.ru.md`](docs/ROADMAP.ru.md).

## Для разработчиков

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

cp .env.example .env
# Настройте сильные секреты и allowed origins
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

Contributions приветствуются. Приоритетные направления:

- Security review и cryptographic improvements.
- E2EE protocol development и testing.
- Performance analysis и benchmarking.
- Documentation.

Для крупных изменений сначала откройте Issue.

## License

MIT License. См. [`LICENSE`](LICENSE).
## Documentation Source of Truth

Current documentation is kept in the repository: [`README.md`](README.md), [`SECURITY.md`](SECURITY.md), [`THREAT_MODEL.md`](THREAT_MODEL.md), and [`docs/`](docs/). Do not rely on a separate GitHub Wiki for current project state, because it can drift from the code.

