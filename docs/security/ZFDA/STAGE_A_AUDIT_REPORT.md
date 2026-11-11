# ZFDA 1.3 — Этап A: Отчёт о read-only аудите текущего состояния

**Дата аудита:** 02.11.2026  
**Проект:** Zenthril (мессенджер, analog Discord)  
**Путь к репозиторию:** `C:\Users\User\Desktop\дс analogs`  
**Ветка:** `main`  
**Статус:** Alpha (не production-ready)  
**Метод:** Read-only аудит, без изменений в коде

---

## 1. Git-состояние

| Параметр | Значение | Статус |
|----------|----------|--------|
| Git root | `C:\Users\User\Desktop\дс analogs` | Verified |
| Текущая ветка | `main` | Verified |
| Удалённый репозиторий | `https://github.com/ax1plee11/Zenthril.git` | Verified |
| Зафиксировано ли последнее изменение | Нет — есть uncommitted изменения | **Failed** |
| Изменённых файлов (modified) | 8 | |
| Неотслеживаемых файлов/папок | 11 | |

### Modified файлы (не закоммичены):
- `.github/workflows/ci.yml` (-1 строка)
- `backend/internal/crypto/protocol.go` (+45/-)
- `backend/internal/crypto/ratchet.go` (+30/-)
- `backend/internal/crypto/test_vectors_test.go` (+330/--)
- `client/src/features/e2ee/pairwiseSession.security.test.ts` (+194/----)
- `client/src/features/e2ee/pairwiseSession.trace.test.ts` (+/-16)
- `docker-compose.yml` (+60/---)
- `research/results/RESEARCH_REPORT.md` (+20/-)

### Untracked (не в Git):
- `.kilo/`
- `INTEGRATION_ANALYSIS.md`
- `Zenthril_ZenthrilDB_Technical_Analysis.docx`
- `backend/migrations_test/`
- `client/src/features/e2ee/doubleRatchetDesign.test.ts`
- `docs/architecture/`
- `docs/audits/`
- `generate_dissertation.py`
- `zenthrildb-push/`

**Вывод:** Последние изменения ZFDA 1.0–1.2 находятся в одном репозитории, но **не закоммичены**. Требуется коммит перед продолжением работ.

---

## 2. Архитектурная карта компонентов

### Backend (Go, порт 8080)

```
cmd/
  api/          — основная точка входа HTTP API
  worker/       — фоновый обработчик
  sfu/          — WebRTC SFU
internal/
  app/          — DI контейнер (fx)
  config/       — загрузка конфигурации
  crypto/       — X25519, HKDF, AES-GCM, Double Ratchet, X3DH
  cqrs/         — CQRS/event store
  domain/       — доменные типы
  event/        — event bus (memory, kafka)
  gateway/      — WebSocket шлюз (rate limiting, IP валидация)
  observability/— трейсинг
  repository/   — sharded repository
auth/           — аутентификация (JWT, refresh, TOTP)
db/             — PostgreSQL подключение (pgx)
device/         — управление устройствами, ключевые связки
federation/     — федерация (отключена по умолчанию)
friends/        — друзья, запросы в друзья
guild/          — гильдии, роли, каналы, приглашения, бан
hub/            — WebSocket хаб (сообщения, лимиты)
message/        — сообщения, история, редактирование, удаление
metrics/        — Prometheus метрики
middleware/     — CORS, sécurité заголовки, rate limiting
security/       — Guard (IP rate limiting, trusted proxy)
spam/           — защита от спама
user/           — пользователи, поиск
```

Ключевые библиотеки:
- `go-chi/chi/v5` — HTTP роутер
- `golang-jwt/jwt/v5` — JWT (HS256)
- `jackc/pgx/v5` — PostgreSQL драйвер
- `redis/go-redis/v9` — Redis клиент
- `golang.org/x/crypto v0.27.0` — криптография
- `segmentio/kafka-go` — event bus

**Статус:** 20 пакетов проходят тесты, `go vet` чист, `go build` успешен.

### Frontend (Tauri v2 + React + TypeScript, порт 1420 dev)

```
src/
  api/          — API клиент, сессия
  crypto/       — клиентская E2EE (ключи, сессии, ratchet)
  features/
    e2ee/       — pairwise сессия, групповая сессия, верификация устройств
    p2p/        — P2P обмен сообщениями
    voice/      — голосовой чат
    webrtc/     — ICE политика
  components/   — React компоненты UI
  store/        — Zustand сторы (auth, ws, theme)
  transport/    — транспорт (padding, связь)
```

Ключевые зависимости:
- `@tauri-apps/api ^2.0.0` — Tauri API
- `@tauri-apps/plugin-shell ^2.3.5` — shell plugin (потенциальный риск)
- `@noble/curves ^2.0.1` — эллиптические кривые
- `@noble/hashes ^2.2.0` — хеширование
- `socket.io-client ^4.8.3` — WebSocket клиент

**Статус:** `npm audit` чист, тесты проходят, сборка успешна.

### Инфраструктура

| Компонент | Образ | Версия | Порты | Статус |
|-----------|-------|--------|-------|--------|
| PostgreSQL | `postgres:16-alpine` | pinned | 5432 (не опубликован) | Partially Verified |
| Redis | `redis:7-alpine` | pinned | 6379 (не опубликован) | Partially Verified |
| Backend | самобильд (`./backend`) | — | 8081→8080 | Partially Verified |

### Нарушение принципа ZFDA

**Каждый компонент должен иметь собственную защиту.** В текущем состоянии:
- PostgreSQL и Redis не имеют TLS между собой и бэкендом (`sslmode=disable`)
- Redis не имеет ACL/аутентификации
- Бэкенд использует переменные окружения для секретов (не secret manager)
- Нет разделения ролей в БД (бэкенд использует одного пользователя для всего)

---

## 3. Настройки production и конфигурация

### docker-compose.yml

**Проблемы (Critical/High):**

1. **Нет secret management** — все секреты передаются через environment variables
2. **Нет .dockerignore** — контекст сборки включает всё, включая потенциальные секреты
3. **Нет rootless execution** — контейнеры запускаются без явного указания пользователя (бэкенд имеет `USER zenthril`, но postgres/redis — образы по умолчанию)
4. **Нет read-only файловой системы** — нет `read_only: true`
5. **Нет Linux capabilities** — не ограничены capabilities
6. **Нет ресурсных лимитов на все контейнеры** — только mem_limit/cpus на бэкенде
7. **PostgreSQL: `sslmode=disable`** — трафик между бэкендом и БД не зашифрован
8. **Redis: нет ACL, нет аутентификации** — открыт для любого в сети docker
9. **Environment по умолчанию: `development`** — если не設定, применяются небезопасные дефолты
10. **SECURE_COOKIES по умолчанию: `false`** — куки не Secure в dev

**Положительные моменты:**
- Obrazy закреплены версиями (`postgres:16-alpine`, `redis:7-alpine`)
- Порты БД и Redis не опубликованы наружу
- Health checks настроены
- Многослойная сборка бэкенда (multi-stage)
- Бэкенд запускается от пользователя `zenthril` (не root)

### backend/Dockerfile

**Положительные:**
- Многослойная сборка (builder + alpine runtime)
- Не root пользователь (`zenthril`)
- `ca-certificates` установлены
- Health check настроен
- `CGO_ENABLED=0` (статичный бинарник)

**Проблемы:**
- Нет `.dockerignore`
- Нет сканирования уязвимостей в Dockerfile
- Нет pinned базового образа для runtime (alpine:3.19 — ок, но можно закрепить дайджестом)

### backend/config/config.go

**Положительные:**
- `JWT_SECRET` минимум 32 символа
- Production mode: fail-fast на placeholder секреты
- Production mode: требует `METRICS_TOKEN`, `OPERATIONAL_TOKEN`
- Production mode: требует `CORS_ALLOWED_ORIGINS` и `WS_ALLOWED_ORIGINS`
- Нет wildcard в CORS/WS origins
- `TrustedProxyHops` по умолчанию 0 (безопасный дефолт)
- Валидация схем origins (только http/https, без путей)

**Проблемы:**
- `RedisURL` по умолчанию `redis://localhost:6379` без аутентификации
- `Environment` по умолчанию `development`
- `SecureCookies` по умолчанию `false`
- Нет проверки на сильные пароли для PostgreSQL

---

## 4. Управление секретами (инвентаризация)

| Секрет | Назначение | Место хранения | Доступ | Ротация | Статус |
|--------|------------|----------------|--------|---------|--------|
| `JWT_SECRET` | Подпись JWT access/refresh токенов | `.env` (реальное значение) | Все с доступом к файлу | Не документировано | **Failed** |
| `POSTGRES_PASSWORD` | Пароль PostgreSQL | `.env` (реальное значение) | Все с доступом к файлу | Не документировано | **Failed** |
| `METRICS_TOKEN` | Доступ к метрикам | `.env.example` (placeholder) | — | Не документировано | Not Verified |
| `OPERATIONAL_TOKEN` | Доступ к health/readiness endpoints | `.env.example` (placeholder) | — | Не документировано | Not Verified |
| `FEDERATION_TOKEN` | Авторизация федерации | `.env.example` (placeholder) | — | Не документировано | Not Verified |
| `VITE_ALLOW_INSECURE_KEY_STORAGE` | Разрешение небезопасного хранения ключей | `client/.env` = `true` | — | — | **Failed** |

**Критические проблемы:**
1. **Реальные секреты в `.env` файлах** в рабочей копии (`.gitignore` их исключает, но они находятся на диске)
2. **Нет secret manager** (HashiCorp Vault, AWS Secrets Manager, etc.)
3. **Нет процедуры ротации JWT ключей** — при компрометации все сессии уязвимы
4. **Нет процедуры ротации TOTP encryption keys**
5. **Нет разделения ключей по назначению** (один JWT_SECRET для всего)
6. **Нет отзыва при компрометации** — нет механизма экстренного отзыва всех сессий

---

## 5. Защита данных

### Шифрование

| Аспект | Состояние | Статус |
|--------|-----------|--------|
| Encryption in transit (клиент↔бэкенд) | Нет enforced TLS, HTTP по умолчанию | **Failed** |
| Encryption in transit (бэкенд↔PostgreSQL) | `sslmode=disable` | **Failed** |
| Encryption in transit (бэкенд↔Redis) | Нет TLS | **Failed** |
| Encryption at rest (диск) | Не настроено в Docker | Not Verified |
| Application-level encryption | TOTP через AES-256-GCM | Verified |
| End-to-end encryption (сообщения) | X25519 + HKDF + AES-256-GCM, протокол v2 | Partially Verified (alpha) |

**E2EE:** Сервер не имеет доступа к приватным ключам пользователей. Сообщения шифруются на клиенте. Это соответствует принципу ZFDA.

**Проблемы:**
- E2EE еще не Signal-grade (нет полного X3DH/Double Ratchet интеграции на бэкенде)
- Нет внешнего криптографического аудита
- Групповой E2EE ограничен 10 устройствами (Sender Keys foundation)

---

## 6. Безопасность клиента (Tauri)

### tauri.conf.json

**Критические проблемы:**

1. **CSP слишком разрешительный:**
   ```json
   "connect-src 'self' http://localhost:8080 ws://localhost:8080 https: wss:"
   ```
   `https:` и `wss:` разрешают соединение на **любой** внешний HTTPS/WSS хост. Это позволяет exfilтрации данных.

2. **`style-src 'self' 'unsafe-inline'`** — небезопасно для防 XSS через инлайн стили.

3. **Нет раздела `capabilities`** — не явно ограничены возможности приложения.

4. **`@tauri-apps/plugin-shell`** — позволяет выполнение shell команд, потенциальный вектор атаки.

5. **`devUrl: http://localhost:1420`** — в dev режиме доступен локальный сервер.

### client/.env

```
VITE_ALLOW_INSECURE_KEY_STORAGE=true
```
**Критично:** Реальный `.env` файл разрешает небезопасное хранение приватных ключей E2EE в localStorage. Это недопустимо для production.

---

## 7. CI/CD

| Проверка | Наличие | Статус |
|----------|---------|--------|
| `.github/workflows/` | Директория существует, но пуста | **Failed** |
| GitHub Actions | Нет workflow файлов | **Failed** |
| Dependency scanning | Нет | Not Applicable |
| SAST | Нет | Not Applicable |
| Container scanning | Нет | Not Applicable |
| Secret scanning | Нет | Not Applicable |
| SBOM generation | Нет | Not Applicable |
| Trivy | Нет | Not Applicable |

**Критично:** Отсутствие CI/CD означает, что нет автоматизированных проверок безопасности при коммитах, pull-запросах или релизах.

---

## 8. Автономная защита (Autonomous Defense)

Из кода (`backend/internal/gateway/`, `backend/security/`):
- IP rate limiting реализован
- Trusted proxy хопы валидируются
- WebSocket rate limiting на уровне хаба
- Spam guard на сообщения
- Блокировка `/debug/*` endpoints в production

**Не проверено:**
- Получение реальных событий
- Отсутствие бесконечных циклов реагирования
- Поведение при недоступности Redis/PostgreSQL
- Audit trail действий защиты

Статус: Partially Verified — частично реализовано, не протестировано в сценариях отказа.

---

## 9. Резервное копирование и аварийное восстановление

| Компонент | Бэкап | Восстановление | Статус |
|-----------|-------|----------------|--------|
| PostgreSQL | Нет настроенного | Не тестировалось | Not Applicable |
| Redis | Нет настроенного | Не тестировалось | Not Applicable |
| Конфигурации | Нет | — | Not Applicable |
| Секреты | Нет | — | Not Applicable |

Нет документации по RPO/RTO, retention, шифрованию бэкапов, offsite копиям.

---

## 10. Карта угроз (THREAT_MODEL.md)

Существует и охватывает:
- Сетевые атакующие
- Злоумышленники с CORS/CSWSH
- Компрометированные токены
- Компрометированные клиенты/устройства
- Компрометированный оператор сервера
- Supply-chain атакующие

Trust boundaries определены. Mitigations перечислены. Статус: Verified (для alpha).

---

## 11. Компоненты, готовые к production

| Компонент | Готовность | Обоснование |
|-----------|------------|-------------|
| Backend Go код | Partially | Тесты проходят, но нет CI/CD, нет Docker hardening |
| Frontend код | Partially | Тесты проходят, но CSP слишком разрешительный, insecure key storage |
| PostgreSQL | Not Verified | Нет TLS, нет не-superuser пользователя, нет бэкапов |
| Redis | Not Verified | Нет ACL, нет аутентификации, нет бэкапов |
| Docker образы | Partially | Multi-stage, non-root, но нет .dockerignore, нет capabilities |
| E2EE | Not Verified | Alpha, не аудитировано, не Signal-grade |
| CI/CD | Failed | Нет workflow файлов |
| Secrets management | Failed | Нет secret manager, секреты в .env |
| Monitoring/logging | Not Verified | Есть метрики, но нет централизованного логирования |

---

## 12. План работ (Этапы B–J)

### Необходимые действия (по приоритету):

**Критические (блокируют production):**
1. [ ] Создать `.dockerignore` для исключения секретов из контекста сборки
2. [ ] Удалить `.env` файлы с реальными секретами из рабочей копии (оставить только `.env.example`)
3. [ ] Настроить CI/CD с GitHub Actions (trivy, dependency scanning, SAST, secret scanning)
4. [ ] Исправить CSP в `tauri.conf.json` (убрать `https:` и `wss:` из connect-src)
5. [ ] Установить `VITE_ALIGHT_INSECURE_KEY_STORAGE=false` в клиентском `.env`
6. [ ] Добавить Redis ACL и аутентификацию
7. [ ] Настроить TLS между бэкендом и PostgreSQL (`sslmode=require`)
8. [ ] Создать не-superuser роль для PostgreSQL
9. [ ] Документировать процедуру ротации JWT и TOTP ключей
10. [ ] Создать `docs/security/ZFDA/` документы для всех этапов

**Высокий приоритет:**
11. [ ] Добавить rootless execution в docker-compose
12. [ ] Добавить read-only файловую систему где возможно
13. [ ] Ограничить Linux capabilities
14. [ ] Добавить resource limits на все контейнеры
15. [ ] Настроить secret management (переменные окружения из секретного хранилища)
16. [ ] Протестировать восстановление из бэкапа
17. [ ] Провести аудит Tauri capabilities и IPC

**Средний приоритет:**
18. [ ] Добавить резервное копирование PostgreSQL/Redis
19. [ ] Документировать RPO/RTO
20. [ ] Добавить зависимость проверок в CI/CD (блокирующие критические проблемы)
21. [ ] Провести статический анализ кода (golangci-lint, semgrep)
22. [ ] Настроить мониторинг и централизованное логирование

---

## 13. Статус этапов ZFDA 1.3

| Этап | Название | Статус |
|------|----------|--------|
| A | Аудит текущего состояния | **Verified** (отчёт составлен) |
| B | Infrastructure Security | Not Started |
| C | Secrets Management | Not Started |
| D | Data Protection | Not Started |
| E | Backup & Disaster Recovery | Not Started |
| F | CI/CD Security | Not Started |
| G | Tauri Client Security | Not Started |
| H | Autonomous Defense Production Readiness | Not Started |
| I | Security Testing | Not Started |
| J | Production Deployment Plan | Not Started |

---

## 14. Вывод

Zenthril находится на стадии alpha и **не готов к production-развёртыванию**. Основные блокирующие проблемы:

1. **Отсутствие CI/CD** — нет автоматизированных проверок безопасности
2. **Секреты в `.env`** — реальные значения JWT_SECRET и POSTGRES_PASSWORD в рабочей копии
3. **Небезопасный CSP клиента** — разрешает соединения на любые HTTPS/WSS хосты
4. **Небезопасное хранение ключей клиента** — `VITE_ALLOW_INSECURE_KEY_STORAGE=true`
5. **Отсутствие TLS** между компонентами инфраструктуры
6. **Отсутствие secret management** — нет централизованного хранения и ротации секретов

Следующий шаг: Этап B — Infrastructure Security (PostgreSQL, Redis, Docker, OS hardening).
