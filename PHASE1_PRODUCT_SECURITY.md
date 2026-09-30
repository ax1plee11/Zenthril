# PHASE 1 — PRODUCT MODEL и PHASE 2 — SECURITY

Дата: 2026-09-30. Commit `5f193bb`. Все выводы проверены по коду.
Обозначения: **IMPLEMENTED** / **PARTIAL** / **BROKEN** / **MISSING** / **UNVERIFIED**.

## 1. Product Model (PHASE 1)

| Сценарий | Статус | Доказательство |
|---|---|---|
| Регистрация | IMPLEMENTED | `auth/handler.go:41-58` — валидация, политика паролей; Argon2id `auth/password.go:68` |
| Вход | IMPLEMENTED | `main.go:311` + `secGuard.BruteForceProtect`; generic-ошибка `handler.go:95` |
| Выход | IMPLEMENTED | `main.go:312` |
| Refresh / ротация | IMPLEMENTED | `main.go:313`, Redis GETDEL |
| Профиль | PARTIAL | Есть только `GET /auth/me` (`main.go:326`) и поиск (`main.go:382`). Нет смены профиля |
| Устройства | IMPLEMENTED | `main.go:331-333` + `key-bundles/claim` (`main.go:337`) |
| Прямые сообщения (DM) | MISSING | В инвентаре маршрутов нет DM-эндпоинтов. Только guilds/channels |
| Группы (guilds) | PARTIAL | `main.go:341-359`. Лимита участников в роутах не видно, проверка в сервисе |
| Каналы | IMPLEMENTED | `main.go:358-359` |
| E2EE | **BROKEN (interop)** | Внутри клиента согласованно после #CRYPTO-007, но клиент и Go выводят разный материал — см. F-09 |
| Multi-device | PARTIAL | Сессии на устройства есть, синхронизация `previousCounter`/safety numbers не завершена |
| Session management | PARTIAL | `logout-all` (`main.go:325`), но persistence состояния сессии не хранит `dhSendPrivate` |
| Уведомления | UNVERIFIED | Не проверялось в этом проходе |
| Голос | PARTIAL | `cmd/sfu` без тестов; WebRTC в клиенте |
| **Вложения** | **MISSING** | **В инвентаре маршрутов нет ни одного upload/file-эндпоинта** |
| Администрирование | PARTIAL | `admin` — только global ban/unban (`main.go:411-412`) + guild-роли |
| Self-hosting | IMPLEMENTED | 6 Docker-файлов, 4 workflow |
| Backup / recovery | UNVERIFIED | Документации и тестов не найдено в этом проходе |

**Продуктовый вывод:** отсутствие DM и вложений означает, что сценарий
«начать переписку с новым человеком» и «отправить файл» сейчас невозможны.
Это блокирует любой рыночный сценарий «личный мессенджер».

## 2. API Inventory (PHASE 2)

| Method | Path | Auth | Авторизация | Rate limit | Чувствительность |
|---|---|---|---|---|---|
| GET | `/livez` | нет | — | нет | низкая |
| GET | `/health`,`/healthz`,`/ready`,`/readyz` | operational token | — | нет | низкая |
| GET | `/metrics`,`/metrics/prometheus` | metrics token | — | нет | **средняя** (метрики) |
| POST | `/api/v1/auth/register` | нет | — | **НЕТ** | средняя |
| POST | `/api/v1/auth/login` | нет | — | BruteForceProtect | высокая |
| POST | `/api/v1/auth/logout` | нет | refresh-cookie | **НЕТ** | высокая |
| POST | `/api/v1/auth/refresh` | нет | refresh-cookie | **НЕТ** | высокая |
| POST | `/api/v1/auth/mfa/totp/*` | нет | — | нет | — (скрыто в prod) |
| POST | `/api/v1/auth/ws-ticket` | JWT | self | BruteForceProtect | высокая |
| POST | `/api/v1/auth/logout-all` | JWT | self | нет | высокая |
| GET | `/api/v1/auth/me` | JWT | self | нет | средняя |
| GET/POST/DELETE | `/api/v1/devices/*` | JWT | self | нет | высокая |
| POST | `/api/v1/key-bundles/claim` | JWT | проверка в сервисе | нет | **высокая** |
| GET/POST | `/api/v1/guilds/*` | JWT | проверка в сервисе | нет | средняя |
| POST | `/api/v1/invites/{code}/join` | JWT | invite-validate | нет | средняя |
| GET/POST | `/api/v1/channels/{id}/messages` | JWT | **requireChannelAccess** | spamGuard на POST | **высокая** |
| GET | `/api/v1/channels/{id}/e2ee-recipients` | JWT | requireChannelAccess | нет | **высокая** |
| PATCH/DELETE | `/api/v1/messages/{id}` | JWT | в сервисе | нет | средняя |
| GET | `/api/v1/users/search` | JWT | — | нет | средняя |
| GET | `/api/v1/users/{userId}/devices` | JWT | **было: нет. Стало: self+shared guild** | нет | **высокая** |
| GET/POST/DELETE | `/api/v1/friends/*` | JWT | в сервисе | нет | средняя |
| POST/DELETE | `/api/v1/admin/users/{id}/ban` | JWT + adminOnly | admin | нет | **критическая** |
| GET | `/ws` | WS-ticket | Origin fail-closed | in-protocol | высокая |
| POST/GET | `/federation/v1/*` | federation token | federation | нет | **высокая** |

## 3. Проверка трёх названных в программе endpoint

### 3.1 `GET /api/v1/channels/{channelId}/messages` — ЗАЩИЩЁН
`message/handler.go:71-101` → `service.GetHistory` → `requireChannelAccess`
(`service.go:155`) → `guild.UserHasChannelAccess` (`guild/service.go:495`).
Отказ → `ErrNotChannelMember` → HTTP 403. Лимит `limit` зажат `maxLimit=50`
(`service.go:151`). Утечки BOLA нет.

### 3.2 `GET /api/v1/users/{userId}/devices` — БЫЛА BOLA, ИСПРАВЛЕНА
**Было:** `device/handler.go:60-67` брал `userId` из URL и вызывал
`listUserDevices(userID)` без всякой проверки. `ListUserDevices`
(`device/service.go:177-192`) возвращал полный `Device`:
`identity_public_key`, `identity_dh_public_key`, `signed_pre_key`,
`signed_pre_key_signature`, `fingerprint`, `trust_state`,
`one_time_prekey_count`, `last_seen_at`.

**Эксплойт-цепочка (полностью подтверждена):**
1. `GET /api/v1/users/search?q=al` → `{id, username}` до 20 пользователей
   (`user/service.go:50-60`). Требуется только авторизация.
2. `GET /api/v1/users/<id>/devices` → полное ключевое содержимое устройств жертвы.

**Последствия:** кража отпечатков для целевого MITM в X3DH; наблюдение
`one_time_prekey_count` показывает, когда у жертвы заканчиваются prekeys
(окно ослабления X3DH и тайминга); `trust_state` раскрывает.security-посттуру
жертвы; `last_seen_at` — трекинг активности.

**Исправлено:** `device/service.go` — добавлены `PublicDevice`
(только `device_id`, `fingerprint`, `trust_state`), `deviceVisibilityDecision`
и `ListUserDevicesPublic` с обязательной проверкой общего гильда и
незабаненности обеих сторон. `device/handler.go` — `ListUser` теперь требует
аутентификацию, отдаёт полные данные только себе, а для других — минимизированную
проекцию; при отсутствии общего гильда отдаёт 404, чтобы нельзя было зондировать
существование аккаунта.

Клиент этот endpoint не вызывает (`api.users.devices` определён в
`client/src/api/index.ts:298`, но call sites отсутствуют), поэтому изменение
контракта ничего не ломает внутри репозитория. Изменение API задокументировано
здесь и требует записи в CHANGELOG.

### 3.3 `GlobalBan` — ЗАЩИЩЁН
`main.go:408-413`: `r.Use(authSvc.Middleware)` + `r.Use(adminOnly(cfg))`.
Двойной барьер, корректно.

## 4. Исправления моих предыдущих ошибочных утверждений

Первоначальный обзор содержал три неверных утверждения. Исправляю:

| Утверждение | Статус | Факт |
|---|---|---|
| «`GET .../messages` — нет проверки, потенциальный IDOR» | **НЕВЕРНО** | `requireChannelAccess` проверяет членство |
| «`GlobalBan` не проверяет роль администратора» | **НЕВЕРНО** | применён `adminOnly(cfg)` |
| «MFA-заглушки отдают 500 в production» | **НЕВЕРНО** | handlers отдают 501, и в production маршруты **вообще не регистрируются** (`main.go:317`) |

## 5. Новые находки PHASE 2

| ID | Severity | Находка | Доказательство |
|---|---|---|---|
| S-01 | HIGH | `POST /auth/register` без rate limit — массовая регистрация аккаунтов и занятие username | `main.go:310` (BruteForceProtect только на `/login`, `main.go:311`) |
| S-02 | HIGH | `POST /auth/refresh` и `/logout` без rate limit | `main.go:312-313` |
| S-03 | MEDIUM | Нет глобального HTTP rate-limit middleware; лимиты только точечные | `main.go` |
| S-04 | MEDIUM | Нет ограничения числа устройств на пользователя — неограниченная регистрация devices раздувает `devices` | `main.go:332`, миграция `006` |
| S-05 | LOW | `GET /users/search` отдаёт 200 + `[]` на некорректный запрос, смешивая валидацию с успехом | `main.go:386-393` |

## 6. Что осталось непроверенным (честно)

- Federation (`/federation/v1/*`) — токен-аутентификация не разбиралась.
- Friends-сервис — авторизация не разбиралась (пакет без тестов).
- Moderation, spam, security-пакеты — без тестов, поведение не подтверждено.
- WebSocket-протокол (hub) — частично, ранее в этом сессии.
- Реальное развёртывание, backup/restore, performance — отдельные фазы.
