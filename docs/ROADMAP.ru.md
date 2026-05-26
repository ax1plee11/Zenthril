# Дорожная карта Zenthril

Эта дорожная карта описывает путь от текущей alpha-версии к более безопасному, поддерживаемому и полезному для исследования мессенджеру.

Zenthril все еще находится в alpha-стадии. Запланированные пункты нельзя описывать как production-ready, пока они не реализованы, не протестированы и не задокументированы.

## Phase 0: Security Foundation

Статус: в основном завершено для текущей alpha-стадии.

- Строгая проверка CORS и WebSocket Origin.
- Ограничение размера WebSocket-сообщений и базовая защита от flood-атак.
- Middleware с security headers.
- Защищенные metrics и operational endpoints.
- JWT access tokens, refresh tokens и Redis-backed revocation.
- Более безопасная валидация production-конфигурации.
- Non-root Docker runtime и health checks для deployment.

Что осталось:

- Перенести хранение приватных ключей в OS keychain или Stronghold для production desktop builds.
- Расширить security-тесты для browser-origin атак и token replay.
- Описать threat model для alpha-релиза.

## Phase 1: E2EE Protocol Maturity

Статус: в процессе.

- Интегрировать Double Ratchet state в реальные message send/receive flows.
- Добавить X3DH-style pre-key bundles и расходование one-time pre-keys.
- Добавить safety number verification в клиентский UX.
- Улучшить multi-device session handling.
- Добавить session healing после пропущенных сообщений и изменений устройств.
- Добавить property-based tests и test vectors для ratchet behavior.

Критерии завершения:

- Сервер не может расшифровать содержимое сообщений.
- Key rotation проверяется автоматическими тестами.
- Device revocation блокирует будущие сессии с отозванными устройствами.
- Документация честно описывает оставшиеся metadata leaks.

## Phase 2: Architecture And Observability

Статус: частично начато.

- Продолжить перенос next-generation API на explicit dependency injection через `uber/fx`.
- Добавить OpenTelemetry traces и metrics для HTTP, WebSocket, database и event paths.
- Четче разделить gateway, event bus, repository и service boundaries.
- Сохранять legacy API стабильным, пока новая внутренняя архитектура созревает.
- Добавить structured logs для security-relevant events.

## Phase 3: Realtime Scaling

Статус: запланировано.

- Запускать несколько WebSocket gateway instances.
- Добавить Redis Pub/Sub fan-out между gateway nodes.
- Подготовить event layer для Kafka, Redpanda или NATS JetStream.
- Добавить tests для connection draining и graceful shutdown.
- Определить delivery semantics: at-least-once delivery с idempotent client-side deduplication.

## Phase 4: Federation

Статус: запланировано; текущие federation endpoints не являются production-ready.

- Сначала определить минимальный federation protocol.
- Добавить node identity, signed server-to-server requests и replay protection.
- Сделать compatibility tests между двумя локальными Zenthril nodes.
- Описать, какие данные федерализуются, а какие остаются локальными.

## Phase 5: Research-Grade Evaluation

Статус: активно и постоянно.

- Поддерживать воспроизводимые benchmark scripts.
- Фиксировать hardware, dataset и test parameters для каждого опубликованного результата.
- Сравнивать security/scaling tradeoffs с Matrix, Signal-style messaging и Discord-like realtime systems.
- Готовить графики и анализ для дипломной или магистерской работы.
