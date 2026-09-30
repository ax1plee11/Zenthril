# CODEBASE_AUDIT.md

PHASE 0 — Repository Truth. Baseline собран 2026-09-30 на commit `5f193bb`.
Все утверждения ниже проверены командами; указанные команды воспроизводимы.

> **СТАТУС НА 2026-09-30 (продолжение работы).** Дефекты F-01, F-02, F-03
> устранены и проверены. Обнаружены и исправлены дополнительные дефекты,
> не входившие в исходный baseline: см. раздел 6. Актуальная сводка — в
> `PROGRESS_STATUS.md`.

## 1. Фактическая структура

| Область | Состояние |
|---|---|
| `backend/` | Go module `zenthril-backend`, `go 1.26.5`, 97 .go файлов |
| `client/` | Tauri 2 + React + TypeScript, 120 .ts/.tsx файлов, vitest |
| `backend/migrations/` | 13 .sql |
| `.github/workflows/` | `ci.yml`, `supply-chain.yml`, `build-desktop.yml`, `deploy-pages.yml` |
| Docker | `Dockerfile`, `backend/Dockerfile`, `docker-compose.yml`, `docker-compose.debug.yml`, `deployments/docker-compose.prod.yml`, `deployments/docker-compose.monitoring.yml` |
| `zenthrildb-push/` | **C++/CMake, вне git (untracked)**, содержит `.obj` артефакты сборки и `Отчет_по_этапу_разработки_ZenthrilDB.docx` |
| Docs | 53 .md |

## 2. Маркеры в коде (результат `git grep`)

| Маркер | Найдено | Интерпретация |
|---|---|---|
| TODO | 0 | — |
| FIXME | 0 | — |
| HACK / XXX | 0 | — |
| WIP | 9 | Из них 4 реальные в `pairwiseSession.security.test.ts`; остальные — ложные срабатывания (`AllowIP` содержит `wIP`) |
| stub | 53 | Подавляющее большинство — заглушки в тестах (test doubles), легитимно |
| placeholder | 44 | **Не заглушки.** 18 — детекторы placeholder-секретов в `config.go`; остальные — дефолты конфигурации и demo-UI компоненты |
| not implemented | 4 | Реальные заглушки, см. F-06 |
| panic | 1 | `backend/internal/crypto/test_vectors_test.go:390` — panic в тест-хелпере |

**Важно:** 44 вхождения «placeholder» — это защитный механизм `isPlaceholderSecret()`
(`backend/config/config.go:241`, `backend/internal/config/config.go:363`), который
fail-closed валидирует продакшн-конфигурацию. Это положительная находка, не долг.

## 3. Подтверждённые сильные стороны

| # | Факт | Доказательство |
|---|---|---|
| S-01 | `go vet` чист | exit 0, 0 замечаний |
| S-02 | `go test ./...` проходит | exit 0, 15 пакетов `ok` |
| S-03 | Race detector реально запускается в CI | `ci.yml:74` `go test -race` на `ubuntu-latest` |
| S-04 | `govulncheck` — настоящий гейт | `supply-chain.yml:28-32`, без `continue-on-error` |
| S-05 | CodeQL — настоящий гейт | `supply-chain.yml:34-53`, без `continue-on-error` |
| S-06 | Fail-closed валидация продакшн-конфига | `backend/config/config.go:79-120` |
| S-07 | Пароли: Argon2id + политика сложности | `backend/auth/password.go:33,68` |
| S-08 | Хардкод-секретов в Go-исходниках не найдено | `git grep` по `password/secret/api_key` |
| S-09 | JWT: HS256 жёстко закреплён, `alg:none` отвергается | `backend/auth/jwt.go:95-101` |
| S-10 | Fail-closed CSWSH-защита | `backend/hub/hub.go` |

## 4. Подтверждённые дефекты

### F-01 — ~~CI на `main` КРАСНЫЙ~~ **ИСПРАВЛЕНО**
```
npx eslint src/crypto/ratchet.ts src/features/e2ee/groupSession.test.ts
✖ 5 problems (5 errors, 0 warnings)   exit 1
```
5 ошибок в закоммиченном состоянии:
- `client/src/crypto/ratchet.ts:57` — `prefer-const` (`newRecvChainKey`)
- `client/src/features/e2ee/groupSession.test.ts:111,117,124` — `no-explicit-any`
- `client/src/features/e2ee/groupSession.test.ts:129` — `prefer-const`

**Исправлено:** `newRecvChainKey` → `const`; `any` заменён на типизированный
хелпер `withUnsupportedVersion` с `unknown`-cast (runtime-поведение тестов
идентично). `prefer-const` в тесте устранён. Результат: `exit 0`.

### F-02 — ~~Линтер Go не может провалить сборку~~ **ИСПРАВЛЕНО**
`ci.yml:77-82`: `golangci-lint-action` с `continue-on-error: true`.
Сначала устранены все находки линтера (0 issues подтверждено локально
`golangci-lint 2.6.0`), затем снят `continue-on-error`. Версия закреплена на
`v2.6.0` — той, которая была проверена локально.

Устранено: 7 мёртвых объявлений (`getMemberLevel`, `storeSkippedMessageKeys`
— точный дубликат логики в `ratchet.go:232-246`, `globalBanUser`,
`MetricsCollector.mu`, `allowedCORSMethods`, `allowedCORSHeaders`,
`validPreflightRequest` в `main.go`) и 35 файлов, не проходивших gofmt.

### F-03 — ~~Бенчмарки не проверяются~~ **ИСПРАВЛЕНО**
`ci.yml:88`: `go test -bench=. -benchmem ./benchmarks/... || true` — вывод
отбрасывался. Теперь `|| true` убран, а вывод сохраняется через
`tee benchmarks.txt` и публикуется артефактом `backend-benchmarks`.


### F-04 — Trivy не блокирует сборку (MEDIUM)
`ci.yml:136`: `continue-on-error: true`. SARIF выгружается
(`ci.yml:145-149`), но CRITICAL/HIGH уязвимости не останавливают пайплайн.
Фактически это только отчётность, а не гейт.

### F-05 — 15 из 30 Go-пакетов без тестов (HIGH)
Без тестов: `cmd/api`, `cmd/sfu`, `cmd/worker`, `db`, `federation`, `friends`,
`internal/domain`, `internal/event`, `internal/observability`, `internal/pubsub`,
`logger`, `models`, `moderation`, `spam`, `security`-соседние слои.
Не покрыты тестами именно слои с наибольшим риском: federation, moderation,
friends, точки входа.

### F-06 — MFA заявлена в роутах, но отсутствует (HIGH)
`backend/auth/handler.go:168,172,176` — три обработчика возвращают
`501 Not Implemented` для TOTP setup / confirm / disable.
`backend/main.go:314` это подтверждает в комментарии.
Маршруты зарегистрированы, функциональности нет.

### F-07 — Отключённые security-утверждения в тесте (HIGH)
`client/src/features/e2ee/pairwiseSession.security.test.ts:395,404`
(незакоммиченные правки пользователя) содержат закомментированные `expect(...)`:
```ts
// expect(Array.from(recv2.messageKey)).toEqual(Array.from(msg2.messageKey));
```
Кросс-стороннее сопоставление DH ratchet **не проверяется**, а тест при этом
зелёный. Зелёный статус вводит в заблуждение.

### F-08 — ZenthrilDB вне контроля версий и не интегрирован (HIGH)
`zenthrildb-push/` — untracked. Содержит C++ исходники (`main.cpp`, `core/`,
`storage/`, `wal/`, `transaction/`, `recovery/`), `CMakeLists.txt`, отчёт `.docx`
и **скомпилированные `.obj` артефакты**. Интеграции с Go-бэкендом нет.
Заявлять ZenthrilDB как замену PostgreSQL оснований не имеется.

### F-09 — Клиент и Go выводят несовместимый криптографический материал (CRITICAL, crypto)

Проверено построчно. Совпадение частичное, что важно для корректной задачи.

**Root ratchet — СОВПАДАЕТ** (это единственная общая деривация):
| | salt | info | раскладка |
|---|---|---|---|
| Go `ratchet.go:113` | `rootKey` | `"zenthril-ratchet-v1:root"` | `root=0:32, chain=32:64` |
| Клиент `ratchet.ts:75` | `rootKey` | `"zenthril-ratchet-v1:root"` | `newRoot=0:32, newChain=32:64` |

**Chain ratchet — НЕ СОВПАДАЕТ:**
| | salt | info | раскладка output (76 байт) |
|---|---|---|---|
| Go `ratchet.go:292,297-301` | `nil` | `"zenthril-ratchet-v1:chain"` | `key=0:32, nonce=32:44, nextChain=44:76` |
| Клиент `ratchet.ts:26-31` | `"zenthril-ratchet-v1"` | `"chain"` | `newChain=0:32, key=32:64, nonce=64:76` |

Разные salt и info дают полностью разный вывод, плюс раскладка сегментов различается.

**Инициализация сессии — НЕ СОВПАДАЕТ:**
- Go `ratchet.go:94`: IKM = один `sharedSecret`, salt `nil`, info `"zenthril-ratchet-v1:init:"+sessionInfo`, 96 байт.
- Клиент `pairwiseSession.ts:330`: IKM = конкатенация 3–4 DH-значений, salt `"zenthril.e2ee.x3dh.root.v1"`, info `"zenthril.e2ee.x3dh.session.v1|"+canonicalHeader`, 96 байт.
- Клиентский `"zenthril-ratchet-v1:init-dh"` (`pairwiseSession.ts:368`) в Go отсутствует полностью.

Unit-тесты обеих сторон проходят независимо, потому что каждый проверяет только
собственную реализацию. Кросс-реализационных test vectors нет, поэтому
несовместимость не обнаруживается ни одним существующим тестом.
Заявлять соответствие Signal нельзя.

**Корректная постановка задачи:** унифицировать цепной ратчет и инициализацию
(выбрать одного владельца схемы — рекомендуется Go как эталон, т.к. сервер
хранит состояние сессий), затем добавить кросс-реализационный test vector,
использующий один и тот же вектор в Go и TypeScript.

### F-10 — Race detector не запускается на dev-машине (ENV BLOCKER)
`-race is not supported on windows/386`; при `GOARCH=amd64` —
`-race requires cgo`, C-тулчейн отсутствует. Гейт покрыт CI (S-03),
но локальная проверка невозможна без установки тулчейна.

### F-11 — Несоответствие версий инструментов (LOW)
`go.mod` требует `1.26.5`, локально `1.26.1` → сборка падает без сети.
Локальный Node `v25.2.1`, CI — `24`.

## 5. Статус предыдущей задачи

TASK #CRYPTO-007 закрыт: 5 дефектов исправлено, 9 regression-тестов добавлено,
`29 files / 185 tests passed`. Изменения не закоммичены. Детали в отчёте задачи.
F-09 остаётся открытым и требует отдельной задачи на унификацию деривации.
