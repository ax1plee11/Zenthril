# AUDIT_BASELINE.md

Дата снятия baseline: 2026-09-30
Зафиксировано ДО любых изменений в рамках PHASE 0.

## 1. Git

| Параметр | Значение |
|---|---|
| Branch | `main` |
| HEAD | `5f193bb2a0a4ace80d74ec5d7edd0982c2fe332f` |
| Commit message | `fix(e2ee): remove duplicate interface, fix TS test state tracking` |
| Author | ax1plee <ax1plee11@users.noreply.github.com> |
| Commit date | 2026-09-23 10:21:14 +0300 |
| Remote | `origin` → https://github.com/ax1plee11/Zenthril.git |
| Commits in history | 144 |
| Tags | `v0.1.1-alpha-provenance`, `v0.1.0-alpha` |
| Local branches | `main` (current), `agents/e0095ad3-…`, `flint-respect`, `swamp-taker` |

Три локальные ветки кроме `main` — worktree/agent-ветки, удалённых веток нет.
`origin/HEAD -> origin/main`.

## 2. Toolchain (фактически установлено)

| Компонент | Установлено | Требуется проектом | Статус |
|---|---|---|---|
| OS | Windows 11 Pro 10.0.26200 (64-bit) | — | — |
| CPU | AMD Ryzen 5 5600U, 6 cores / 12 threads, 2.3 GHz | — | — |
| RAM | 15.31 GB | — | — |
| Go | go1.26.1 **windows/386** | `go 1.26.5` в `backend/go.mod` | mismatch |
| Node | v25.2.1 | CI использует Node 24 | mismatch |
| npm | 11.6.2 | — | — |
| Rust | rustc 1.94.1 / cargo 1.94.1 | Tauri 2 | OK |
| Docker | 29.3.1 | — | OK |
| PostgreSQL | не установлен локально | 16 (compose) | только Docker |
| Redis | не установлен локально | 7 (compose) | только Docker |

## 3. Фактические результаты проверок

Команды выполнены локально на этом окружении.

| Проверка | Результат |
|---|---|
| `go build ./...` (GOTOOLCHAIN=local) | **FAIL** — `go.mod requires go >= 1.26.5 (running go 1.26.1)` |
| `go build ./...` (auto toolchain) | **PASS** — toolchain 1.26.5 скачан автоматически |
| `go vet ./...` | **PASS**, exit 0, 0 замечаний |
| `go test ./...` | **PASS**, exit 0. 15 пакетов пройдено, **15 пакетов без тестов** |
| `go test -race` | **BLOCKED** — `-race is not supported on windows/386` |
| `GOARCH=amd64 go test -race` | **BLOCKED** — `-race requires cgo`, C-тулчейн отсутствует |
| `npm test` (client, 28 файлов на baseline) | **PASS**, 176/176 |
| `npx tsc --noEmit` (client) | **FAIL** — 13 ошибок (8 из них в незакоммиченных правках пользователя) |
| `npm run lint` (client) | **FAIL** — 5 ошибок в **закоммиченном** состоянии |

## 4. Незакоммиченные изменения пользователя (НЕ ТРОГАТЬ)

| Файл | Состояние |
|---|---|
| `client/src/features/e2ee/pairwiseSession.security.test.ts` | изменён, 170 строк — WIP-тесты DH ratchet |
| `docker-compose.yml` | изменён, −2 строки |
| `client/src/features/e2ee/pairwiseSession.ts` | изменён TASK #CRYPTO-007 (моя работа) |
| `client/src/features/e2ee/messageEnvelopes.ts` | изменён TASK #CRYPTO-007 (моя работа) |
| `client/src/features/e2ee/pairwiseSession.trace.test.ts` | изменён TASK #CRYPTO-007 (моя работа) |
| `client/src/features/e2ee/pairwiseSession.skippedKeys.security.test.ts` | новый файл TASK #CRYPTO-007 |
| `zenthrildb-push/` | untracked, вне git |
| `generate_dissertation.py` | untracked, вне git |

Никакие `reset --hard`, `clean -fd`, force push, переписывание истории не выполнялись.

## 5. Экологические блокеры

1. **Race detector недоступен локально.** Причина: toolchain собран под `windows/386`, а race требует amd64 + cgo. Обход: запускать в CI на Linux. В `ci.yml` job `backend-test` выполняет `go test -race` на `ubuntu-latest` — гейт достижим, но не на этой машине.
2. **Go version mismatch.** Локальный go1.26.1 < требуемых 1.26.5. Работает только через авто-загрузку toolchain (требует сети). При офлайн-сборке — падение.
3. **PostgreSQL и Redis отсутствуют локально.** Интеграционные тесты требуют Docker; локальный прогон возможен только через контейнеры.
4. **GOARCH=386 по умолчанию** ограничивает адресное пространство процесса (~2 ГБ), что критично для Go-сервера с пулами соединений. Тесты на этой архитектуре не репрезентативны для production amd64.

## 6. Воспроизводимость заявленных бенчмарков

README содержит таблицу бенчмарков со ссылкой на i7-12700K / 32 GB RAM.
На текущем оборудовании (Ryzen 5 5600U laptop, 15.31 GB RAM, 32-bit Go) эти
результаты **не воспроизводимы**. Любое цитирование этих чисел как доказательства
производительности недопустимо без повторного замера на фиксированном стенде.
