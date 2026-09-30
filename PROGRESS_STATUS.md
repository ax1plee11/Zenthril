# PROGRESS_STATUS.md

Дата: 2026-09-30. Commit `5f193bb` + незакоммиченные изменения текущей сессии.
Все результаты получены реальными прогонами команд.

## 1. Исправленные дефекты безопасности

| ID | Дефект | Где | Как исправлено | Тест |
|---|---|---|---|---|
| C-01 | **BOLA на ключевой материал E2EE.** `GET /users/{userId}/devices` отдавал любому аутентифицированному пользователю полный `Device` чужого пользователя: identity keys, signed prekeys, подписи, fingerprint, trust_state, счётчик one-time prekeys, last_seen | `backend/device/handler.go`, `service.go` | `PublicDevice` (только `device_id`, `fingerprint`, `trust_state`) + `deviceVisibilityDecision` + `ListUserDevicesPublic` c проверкой общего гильда; 404 при отсутствии общего гильда, чтобы исключить зондирование существования аккаунта | `device/service_test.go` — 5 тестов |
| C-02 | **CORS-preflight ломал браузерный клиент.** `Header.Values()` не разбивает по запятой, а браузер шлёт `authorization, content-type` одной строкой → 403 на любой JSON POST с авторизацией | `backend/middleware/cors.go` | Разбиение по запятой, как в заглушечной копии в `main.go` | `middleware/cors_test.go` — 6 тестов, до фикса падал |
| C-03 | **Нулевой DH-приватный ключ при DH-ратчете.** `initializeDHRatchet` сохранял ссылку на буфер, который `initiatePairwiseSession` обнулял в `finally`; в persisted-формате нет `dhSendPrivate` → после restore тоже нули. X25519 клампит нулевой скаляр в публичную константу → shared secret вычислим любым | `client/src/features/e2ee/pairwiseSession.ts` | Копирование ключевого материала + fail-closed, если ключ не восстановлен | `pairwiseSession.skippedKeys.security.test.ts` |
| C-04 | **IV зависел от порядка доставки.** `nextMessageKey` брал IV как `newChainKey[0:12]`, а skipped-store отдавал `messageNonce` → out-of-order сообщения не расшифровывались | там же | Используется дедицированный выход `step.messageNonce` | Тот же файл, реальный AES-GCM round-trip |
| C-05 | **Необратимая порча цепочки.** `nextReceiveMessageKey` мутировала входной state | там же | Функция стала чисто функциональной | Тот же файл |
| C-06 | **Счётчик из серверного конверта не валидировался.** `NaN`/дробное значение сдвигали учёт skipped-ключей и рассинхронизировали сессию | там же + `messageEnvelopes.ts` | `Number.isSafeInteger` + fail-closed на границе decrypt | Тот же файл |
| C-07 | **Отсутствие rate limit на happy-path auth.** `BruteForceProtect` реагирует только на 401, поэтому `/register`, `/refresh`, `/logout` не ограничены | `backend/security/guard.go`, `main.go` | `AuthRateLimit(scope, limit, window)` — fail-closed при отказе Redis, паника при невалидной конфигурации; register 10/час, refresh и logout 30/мин | `security/guard_ratelimit_test.go` — 3 теста |

## 2. Исправленные дефекты CI/качества

| ID | Дефект | Статус |
|---|---|---|
| F-01 | Frontend lint падал на 5 ошибках → CI на `main` красный | ИСПРАВЛЕНО, `exit 0` |
| F-02 | golangci-lint с `continue-on-error` не мог провалить сборку; 7 мёртвых объявлений, 35 файлов не проходили gofmt | ИСПРАВЛЕНО, `0 issues` |
| F-03 | Результаты бенчмарков отбрасывались (`\|\| true`) | ИСПРАВЛЕНО, публикуются артефактом |

## 3. Исправленные ошибки в моих предыдущих отчётах

Первоначальный обзор содержал неверные утверждения — исправлены в
`PHASE1_PRODUCT_SECURITY.md` §4:

- «`GET .../messages` — нет проверки, потенциальный IDOR» → **неверно**,
  `requireChannelAccess` проверяет членство.
- «`GlobalBan` не проверяет роль администратора» → **неверно**, применён `adminOnly`.
- «MFA-заглушки отдают 500 в production» → **неверно**, обработчики отдают 501,
  а в production маршруты вообще не регистрируются.
- «Фикс nonce aligns with the Go backend» → **неверно**, фикс чинит внутреннюю
  согласованность клиента; клиент и Go выводят несовместимый материал (F-09).

## 4. Текущее состояние проверок

| Проверка | Команда | Результат |
|---|---|---|
| Backend build | `go build ./...` | **exit 0** |
| Backend vet | `go vet ./...` | **exit 0** |
| Backend lint | `golangci-lint run` (без лимитов) | **0 issues, exit 0** |
| Backend tests | `go test ./...` | **exit 0**, 18 пакетов `ok` |
| Client tests | `npm test` | **29 файлов / 185 тестов passed** |
| Client lint (мои файлы) | `npx eslint <touched>` | **exit 0** |
| Client lint (WIP пользователя) | `npm run lint` | 8 ошибок, см. §5 |
| Client typecheck | `npx tsc --noEmit` | 8 ошибок, только в WIP-файле |
| Race detector | `go test -race` | **BLOCKED локально** (`windows/386`, нет cgo). Покрыт CI |

## 5. Предупреждение о WIP пользователя

`client/src/features/e2ee/pairwiseSession.security.test.ts` содержит
незакоммиченные изменения (93 вставки / 77 удалений) с 8 неиспользуемыми
импортами и закомментированными `expect(...)` на строках 395 и 404.

Эти 8 ошибок происходят **только из незакоммиченных правок**. Коммиченное
состояние чистое. Но при коммите пользовательских изменений шаг
`frontend-test` упадёт, потому что `npm run lint` в CI не имеет
`continue-on-error`.

Импорты намеренно НЕ удалены: они принадлежат закомментированным
утверждениям, и их удаление уничтожило бы заготовку WIP-тестов.

## 6. Открытые блокеры

| ID | Severity | Блокер | Что требуется |
|---|---|---|---|
| F-09 | CRITICAL | Клиент и Go криптографически несовместимы: chain ratchet и session init используют разные salt/info/раскладку | Унифицировать на одном владельце схемы + кросс-реализационный test vector |
| F-10 | ENV | Race detector недоступен локально | Установить C-тулчейн или полагаться на CI (Linux) |
| F-04 | MEDIUM | Trivy в CI с `continue-on-error` | Один прогон для определения реального числа уязвимостей, затем решение о гейте. Локально Trivy нет |
| F-05 | HIGH | 15 из 30 Go-пакетов без тестов (federation, moderation, friends, cmd/*) | Начать с `friends` и `moderation` — там логика доступа |
| S-04 | MEDIUM | Нет лимита числа устройств на пользователя | Лимит + тест |
| — | MEDIUM | ZenthrilDB вне git, содержит `.obj` артефакты, не интегрирован с Go | Решение: в git или явно документировать как исследовательский компонент |
| — | MEDIUM | Отсутствуют DM и вложения | Продуктовое решение |

## 7. Изменённые файлы (не закоммичены)

Мои: `backend/device/{handler,service,service_test}.go`,
`backend/security/{guard.go,guard_ratelimit_test.go}`,
`backend/middleware/cors.go`, `backend/middleware/cors_test.go`,
`backend/main.go`, `backend/guild/service.go`, `backend/moderation/guild_ban.go`,
`backend/internal/crypto/message.go`, `backend/internal/observability/tracer.go`,
`client/src/features/e2ee/{pairwiseSession,messageEnvelopes,pairwiseSession.trace.test}.ts`,
`client/src/features/e2ee/pairwiseSession.skippedKeys.security.test.ts` (новый),
`client/src/crypto/ratchet.ts`, `client/src/features/e2ee/groupSession.test.ts`,
`.github/workflows/ci.yml`, 35 файлов, отформатированных gofmt.

Пользовательские (не тронуты): `client/src/features/e2ee/pairwiseSession.security.test.ts`,
`docker-compose.yml`, `zenthrildb-push/`, `generate_dissertation.py`.

## 8. Ничего не закоммичено

`git reset --hard`, `git clean`, force push, переписывание истории и удаление
веток не выполнялись. Коммит не создавался — по правилам проекта коммит
делается только по явному запросу.
