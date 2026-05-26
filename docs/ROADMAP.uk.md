# Дорожня карта Zenthril

Ця дорожня карта описує шлях від поточної alpha-версії до безпечнішої, підтримуваної та корисної для досліджень платформи обміну повідомленнями.

Zenthril все ще перебуває на alpha-стадії. Заплановані пункти не слід описувати як production-ready, доки вони не реалізовані, не протестовані та не задокументовані.

## Phase 0: Security Foundation

Статус: переважно завершено для поточної alpha-стадії.

- Строга перевірка CORS і WebSocket Origin.
- Обмеження розміру WebSocket-повідомлень і базовий захист від flood-атак.
- Middleware із security headers.
- Захищені metrics та operational endpoints.
- JWT access tokens, refresh tokens і Redis-backed revocation.
- Безпечніша валідація production-конфігурації.
- Non-root Docker runtime і health checks для deployment.

Що залишилося:

- Перенести зберігання приватних ключів в OS keychain або Stronghold для production desktop builds.
- Розширити security-тести для browser-origin атак і token replay.
- Описати threat model для alpha-релізу.

## Phase 1: E2EE Protocol Maturity

Статус: у процесі.

- Інтегрувати Double Ratchet state у реальні message send/receive flows.
- Додати X3DH-style pre-key bundles і використання one-time pre-keys.
- Додати safety number verification у клієнтський UX.
- Покращити multi-device session handling.
- Додати session healing після пропущених повідомлень і змін пристроїв.
- Додати property-based tests і test vectors для ratchet behavior.

Критерії завершення:

- Сервер не може розшифрувати вміст повідомлень.
- Key rotation перевіряється автоматичними тестами.
- Device revocation блокує майбутні сесії з відкликаними пристроями.
- Документація чесно описує залишковий metadata exposure.

## Phase 2: Architecture And Observability

Статус: частково розпочато.

- Продовжити перехід next-generation API на explicit dependency injection через `uber/fx`.
- Додати OpenTelemetry traces і metrics для HTTP, WebSocket, database та event paths.
- Чіткіше розділити gateway, event bus, repository і service boundaries.
- Зберігати legacy API стабільним, поки нова внутрішня архітектура дозріває.
- Додати structured logs для security-relevant events.

## Phase 3: Realtime Scaling

Статус: заплановано.

- Запускати кілька WebSocket gateway instances.
- Додати Redis Pub/Sub fan-out між gateway nodes.
- Підготувати event layer для Kafka, Redpanda або NATS JetStream.
- Додати tests для connection draining і graceful shutdown.
- Визначити delivery semantics: at-least-once delivery з idempotent client-side deduplication.

## Phase 4: Federation

Статус: заплановано; поточні federation endpoints не є production-ready.

- Спершу визначити мінімальний federation protocol.
- Додати node identity, signed server-to-server requests і replay protection.
- Створити compatibility tests між двома локальними Zenthril nodes.
- Описати, які дані федералізуються, а які залишаються локальними.

## Phase 5: Research-Grade Evaluation

Статус: активно й постійно.

- Підтримувати відтворювані benchmark scripts.
- Фіксувати hardware, dataset і test parameters для кожного опублікованого результату.
- Порівнювати security/scaling tradeoffs з Matrix, Signal-style messaging і Discord-like realtime systems.
- Готувати графіки й аналіз для дипломної або магістерської роботи.
