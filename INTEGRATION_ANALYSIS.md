# ZENTHRIL + ZENTHRILDB — ГЛУБОКИЙ АНАЛИЗ И ПЛАН ИНТЕГРАЦИИ

**Дата:** 01.10.2026
**Этап:** Phase 0 — code audit. **Код не изменялся.**
**Зафиксированные ревизии:**
- Zenthril: `40622fb` (branch `main`, remote `github.com/ax1plee11/Zenthril`)
- ZenthrilDB: `c7bdb5e9f73da1b48374e7ee58ee4eba6fd732f` (branch `main`, remote `github.com/ax1plee11/ZenthrilDB`, 2 коммита, тегов нет)

Все утверждения ниже проверены чтением исходников и одним фактическим запуском
тестов ZenthrilDB. Числовые результаты производительности **НЕ ИЗМЕРЕНО** —
см. раздел I.

---

## A. EXECUTIVE SUMMARY

**Zenthril** — self-hosted realtime-мессенджер. Go-бэкенд (Chi, pgxpool,
Redis, Gorilla WebSocket), Tauri 2 + React + TypeScript клиент. Данные
разделены: **PostgreSQL** — 25 таблиц бизнес-данных, **Redis** — временное
состояние (`refresh:`, `security:`, `ws:`).

**ZenthrilDB** — оригинальный C++23 storage engine foundation. Реализует
страничный слой (Page, PageDirectory, FreePageList, FileManager, CRC32),
буферный пул с LRU, WAL с LSN и per-record CRC32, recovery с undo/redo-планом,
транзакции, LockManager с детекцией дедлоков. Внешних зависимостей нет.

**Ключевой вывод анализа.** ZenthrilDB — это подсистема хранения **без
слоя записей**. `Page` — байтовый массив с opaque payload; `TableMetadata`
хранит имя таблицы и root page, но **не колонки и не типы**. То есть движок
умеет адресовать и защищать страницы, но не умеет хранить строку таблицы.
Поэтому на сегодняшний день **ни один компонент Zenthril не может быть
перенесён в ZenthrilDB без предварительного построения record layer** —
то есть без написания собственной СУБД.

**Имеет ли смысл интеграция?** Да, но не как «замена PostgreSQL», а как
исследование стоимости durable local storage в realtime-приложении.
Архитектурно оправданная точка встраивания сегодня — **не серверная БД, а
клиентское local-first хранилище** (ключи и состояние E2EE-сессий), потому
что ZenthrilDB прямо позиционируется как local-first движок, а клиент
Zenthril сегодня хранит состояние в `localStorage`/Tauri store. Второй
оправданный вариант — чисто исследовательский: измерить WAL/recovery
 ZentrhilDB против PostgreSQL на уровне страниц и журнала.

---

## B. АРХИТЕКТУРА ZENTHRIL

```
Client (Tauri 2 + React/TS)
   │  HTTP (Chi) + WebSocket
   ▼
Backend (main.go — построение маршрутов)
   │
   ├── auth        — JWT, Argon2id, refresh-ротация        auth/handler.go, auth/jwt.go
   ├── device      — регистрация устройств, key bundles     device/handler.go, device/service.go
   ├── guild       — guilds, roles, channels, RBAC          guild/service.go
   ├── message     — отправка/история, E2EE envelopes       message/service.go
   ├── friends     — заявки/принятие                         friends/service.go
   ├── user        — поиск                                   user/service.go
   ├── hub         — WebSocket hub, Origin-fail-closed        hub/hub.go
   ├── federation  — announce/peers/inbox                    federation/service.go
   ├── moderation  — баны                                    moderation/guild_ban.go
   ├── spam/security — rate limit                            spam/guard.go, security/guard.go
   │
   ▼
Persistence
   ├── db.Open(dsn) → *pgxpool.Pool            db/db.go   (тонкая обёртка, 26 строк)
   ├── прямые SQL-запросы в сервисах           message/service.go и др.
   ├── internal/repository/shard.go            кольцевой роутер шардинга (выбирает DSN)
   ├── PostgreSQL — 25 таблиц, 15 миграций
   └── Redis    — refresh:, security:, ws:
```

**Факты, установленные чтением кода:**

1. **Слой абстракции хранилища в Zenthril ОТСУТСТВУЕТ.** `db/db.go` — это
   `Open(dsn) (*pgxpool.Pool, error)` и ничего больше. Сервисы держат
   `db *pgxpool.Pool` и выполняют `s.db.Query(ctx, ...)` напрямую
   (например `message/service.go:68`, `:165`).
2. `internal/repository/shard.go` — **не** repository-абстракция. Это
   consistent-hash ring (FNV-1a, виртуальные ноды, веса), который по
   `userID`/`guildID`/`channelID` выбирает DSN. Он ортогонален понятию
   «движок хранения».
3. Таблицы (25, извлечено из 15 миграций): `users`, `sessions` — НЕ НАЙДЕНО
   (refresh-сессии в Redis), `messages`, `message_recipient_envelopes`,
   `channels`, `guilds`, `guild_members`, `guild_member_roles`, `roles`,
   `role_audit_log`, `invites`, `devices`, `device_one_time_prekeys`,
   `device_sessions`, `device_session_skipped_keys`, `user_mfa_totp`,
   `friendships`, `global_bans`, `banned_guilds`, `user_ip_log`,
   `security_log`, `admin_audit_log`, `federation_nodes`,
   `federation_messages`.
4. Конфигурация хранится **не в БД**, а в env/`config.go` — `postgres:`
   таблицы в миграциях нет.

---

## C. АРХИТЕКТУРА ZENTHRILDB

```
Application (main.cpp)
   │
   ▼
Storage API:  PageManager / MetadataManager / FreePageManager
   │
   ▼
BufferManager (core/BufferManager.hpp)  — LRU пул буферов
   │
   ▼
FileManager (core/FileManager.hpp)     — бинарный I/O + блокировки
   │
   ▼
.zdb  (DatabaseHeader, DatabaseMetadata, PageDirectory, FreePageList)

Отдельно:
  WAL:  WriteAheadLog → LogManager → LogRecord (LSN + CRC32) → LogPayload
        RecoveryContext → RecoveryEngine → RecoveryPlan → RecoveryExecutor
                        → RedoApplier / UndoApplier
  Transaction: Transaction → TransactionManager → TransactionLogAdapter
  Lock: LockManager + WaitForGraph + LockTypes (S/X, upgrade policy, victim)
```

| Модуль | Файлы | Реальное состояние |
|---|---|---|
| core | BufferManager, CRCManager, FileManager, FreePageManager, MetadataManager, PageManager, Types, EncryptionInterfaces | Реализовано. PageManager: allocate/free/read/write, flushDirtyPages, validatePage(CRC), pageInfo, защита `std::mutex` |
| storage | Page, PageHeader, DatabaseHeader, DatabaseMetadata, PageDirectory, FreePageList, TableMetadata, BinaryFormat, BinaryIO | Страничный слой и сериализация реализованы. **Record layer отсутствует** |
| wal | WriteAheadLog, LogManager, LogRecord, LogPayload, LogSequenceNumber, RecoveryContext | Реализовано: magic `ZWAL`=0x5A57414C, formatVersion 1, **CRC32 на каждую запись**, LSN, 12 типов записей, max payload 64 МБ, header 36 Б |
| recovery | RecoveryEngine, RecoveryPlan, RecoveryExecutor, CheckpointService | Архитектура undo/redo реализована |
| transaction | Transaction, TransactionManager, TransactionLogAdapter, TransactionContext | Transaction = конечный автомат Active/Committed/RolledBack; WAL пишет Started/Committed/RolledBack |
| lock | LockManager, WaitForGraph, LockTypes | S/X блокировки; ресурсы Database/Metadata/Table/Page; политики апгрейда Wait/Fail/PreAllocate; стратегии жертвы дедлока Oldest/Youngest/FewestLocks/MinimumWait |
| tests | 16 файлов (test_wal, test_recovery, test_transaction, test_lock_manager, test_page_format, test_buffer_manager, test_crc_manager, test_file_manager, test_free_page_manager, test_metadata_*, test_storage_boundaries, test_binary_io, test_log_payload, test_main) | **ФАКТИЧЕСКИ ПРОВЕРЕНО: `zenthrildb_tests.exe` → «All ZenthrilDB tests passed», exit 0** |

**Сборка:** CMake ≥3.20, C++23, MSVC `/W4 /permissive-` и GCC/Clang
`-Wall -Wextra -Wpedantic`. Внешних зависимостей нет. Цели:
`zenthrildb_core` (static lib), `zenthrildb_example`, `zenthrildb_tests`.

**Документация в репозитории (устаревшая относительно кода):**
`docs/architecture.md`, `docs/ROADMAP.md`, `docs/CHANGELOG.md`,
`docs/WAL_SPEC.md`, `docs/RECOVERY_SPEC.md`, `docs/TRANSACTION_SPEC.md`,
`docs/LOCK_SPEC.md`, `docs/ADR/0001-wal-binary-hardening.md`,
`docs/ADR/0002-binary-io-and-buffer-guards.md`.
README перечисляет только MVP-подсистему, хотя WAL/recovery/transaction/lock
уже включены в `CMakeLists.txt` — README не обновлялся.

---

## D. CURRENT STATE MATRIX

| Возможность | Zenthril | ZenthrilDB | Статус |
|---|---|---|---|
| create/open database | pgxpool DSN | DatabaseHeader/Open | **ZenthrilDB — реализовано** |
| create table | `CREATE TABLE` в SQL-миграциях | только `TableMetadata` (имя + rootPage, **без колонок**) | **ZenthrilDB — фундамент** |
| insert (запись) | SQL INSERT | страницы целиком; записи нет | **ZenthrilDB — отсутствует** |
| update | SQL UPDATE | перезапись страницы целиком | **ZenthrilDB — отсутствует** |
| delete | SQL DELETE | `FreePageManager::freePage` (только страниц) | **ZenthrilDB — отсутствует** |
| lookup по ключу | SQL + индексы | только по `PageId` | **ZenthrilDB — фундамент** |
| scan | SQL SELECT | обход страниц вручную | **ZenthrilDB — фундамент** |
| indexes | PostgreSQL B-tree/GIN | индексов нет; PageDirectory — только реестр страниц | **ZenthrilDB — отсутствует** |
| serialization | драйвер pgx | Page/PageHeader/LogRecord/TableMetadata — сериализация есть, **с бинарными guard'ами (ADR-0002)** | **реализовано** |
| concurrency | пул соединений + `pgx` | `std::mutex` в PageManager; LockManager с дедлоками | **частично** (внутри процесса) |
| transactions | ACID PostgreSQL | состояния есть; **atomicity/isolation не доказаны кодом** | **частично** |
| durability | fsync PostgreSQL | WAL + CRC32 + checkpoint (архитектура) | **частично** |
| recovery | PITR/WAL PostgreSQL | RecoveryEngine/Plan/Executor; **redo/undo данные не проверены** | **частично** |
| MVCC | да | нет | **отсутствует** |
| типы данных | SQL | типов нет | **отсутствует** |
| запросы | SQL | нет | **отсутствует** |

---

## E. INTEGRATION POINTS (реальные места в коде Zenthril)

| # | Точка | Файл | Текущий backend | Оправданность |
|---|---|---|---|---|
| IP-1 | Клиентское local-first хранилище состояния E2EE | `client/src/features/e2ee/pairwiseSession.ts` (`serializePairwiseSession` / `restorePairwiseSession`), `client/src/features/e2ee/deviceKeys.ts` | Tauri OS keyring + localStorage fallback | **Оправдано.** ZenthrilDB позиционируется как local-first движок; данные по природе single-device, локальные, без JOIN. Но нужен record layer + шифрование at-rest |
| IP-2 | Кэш/метрики на узле | `internal/observability/tracer.go`, `metrics/middleware.go` | in-memory + Prometheus | **Слабо.** Данные эфемерны; PostgreSQL/Redis тут не конкуренты |
| IP-3 | Append-only `security_log` | `security/guard.go:LogSecurityEvent` → таблица `security_log` | PostgreSQL | **Слабо.** Нужна выборка по IP/user → требует индекса, которого нет |
| IP-4 | `device_session_skipped_keys` | таблица из `014_device_sessions.sql`; клиент `skippedMessageKeys` | PostgreSQL | **Оправдано как первый кандидат.** Small, high-churn, single-owner, без JOIN. Но текущая семантика (лимит 2000, одноразовость ключа) требует атомарности |
| IP-5 | `messages` | `message/service.go` | PostgreSQL | **Сейчас нет.** Нужен record layer + составной индекс (channel, created_at) |
| IP-6 | users / guilds / channels / roles | `guild/service.go`, `user/service.go` | PostgreSQL | **Нет.** Требуют JOIN, внешних ключей, составных индексов, конкурентных транзакций |

---

## F. PROPOSED ARCHITECTURE (без кода)

```
Application (Zenthril services / client)
   │
   ▼
Storage Interface  ← ЭТО ОТСУТСТВУЕТ В ZENTHRIL, ЕГО НУЖНО СОЗДАТЬ
   │                (узкий, НЕ копия SQL-интерфейса)
   ├──────────────► PostgreSQL adapter (pgxpool)   — существующий путь
   │
   └──────────────► ZenthrilDB adapter (cgo / C-ABI)
                        │
                     ZenthrilDB core
```

**Требования к интерфейсу, выведенные из реального кода:**

- Интерфейс должен быть **построен на ячейках (row/record), а не на страницах**.
  Страницы — внутренность движка; выставлять их в Go наружу означает
  зафиксировать формат хранения как публичный API.
- Точки сериализации: сейчас `pgx` сериализует значения сам. Для ZenthrilDB
  сериализация должна быть **явной и версионированной** (record format
  version), иначе обновление клиента сломает существующие данные.
- Transaction boundary сейчас — внутри сервисного метода (`tx, _ := s.db.Begin()`,
  например `message/service.go`). Абстракция должна выражать это как
  `WithTx(ctx, func(Tx) error) error`.
- Concurrency control: у PostgreSQL это его работа. У ZenthrilDB —
  `LockManager`. Абстракция обязана **не выдавать** блокировки наружу иначе
  сломается multi-process сценарий.
- Retry и errors: сейчас сервисы сами интерпретируют `pgx.ErrNoRows` и
  мапят на HTTP-коды. Абстракция должна возвращать типизированные ошибки
  (`ErrNotFound`, `ErrConflict`), иначе HTTP-слой придётся переписывать.
- Durability: `fsync`/WAL flush — ответственность адаптера, не вызывающего кода.

---

## G. REQUIRED CHANGES

**Critical (без них интеграция невозможна):**
1. Record layer в ZenthrilDB: slot directory на странице либо B+-tree поверх
   PageManager. Текущий `Page` — opaque bytes.
2. Типовая система и версионирование формата записи.
3. Индекс (минимально — B+-tree) для `lookup`.
4. Storage-интерфейс в Zenthril (сейчас его нет) + узкий адаптер.
5. Граница Go↔C++: cgo либо отдельный C-ABI сервис. Целесообразнее
   отдельный процесс/сервис, чтобы не смешивать рантаймы и не блокировать
   Go-рантайм файловым I/O.

**Required:**
6. Атомарность commit и доказательство isolation в TransactionManager.
7. Определение durability-политики (group commit / flush на commit).
8. Шифрование at-rest, если в ZenthrilDB попадут ключи (IP-1).

**Optional:**
9. Compaction страниц; vacuum по аналогии с `FreePageManager`.
10. Метрики движка (hit rate буферного пула, число page reads/writes).

**Future:**
11. MVCC, если потребуется несколько писателей.
12. Репликация, распределённый режим (заявлены в README как цель, кода нет).

---

## H. ЭКСПЕРИМЕНТАЛЬНЫЙ ПЛАН

| Этап | Содержание | Что это докажет |
|---|---|---|
| Э-0 | Прогон `zenthrildb_tests` (уже выполнено: PASS, exit 0) | базовая целостность движка |
| Э-1 | Отдельный integration test: create → write pages → close → reopen → verify CRC | что формат переживает переоткрытие |
| Э-2 | WAL: измерить append-латентность, размер лога на единицу данных, throughput | стоимость журналирования vs PostgreSQL |
| Э-3 | Crash simulation: kill процесса в середине записи → reopen → recovery → сверка контрольных сумм | **реально ли работает recovery, а не только существует** |
| Э-4 | Деградация recovery: обрыв записи, CRC-битовая ошибка в WAL, битая страница | границы прочности |
| Э-5 | LockManager: конкурентные читатели/писатели, принудительный дедлок, таймаут | корректность блокировок под нагрузкой |
| Э-6 | Buffer pool: hit rate и число физических I/O на разных рабочих объёмах | где выигрыш локального движка |
| Э-7 | Построение record layer (heap-страницы) → повторить Э-1..Э-3 на строках | сколько реально стоит «простая СУБД» |
| Э-8 | Накладные расходы границы Go↔C++ (cgo vs IPC) | пригодность для realtime-пути |
| Э-9 | Сравнение с PostgreSQL на одинаковом железе | база для научного вывода |

**Методология обязательна:** фиксировать hardware, OS, compiler/runtime,
commit hash обоих репозиториев, размер данных, число итераций, raw-результаты.

---

## I. METRICS

| Метрика | Зачем |
|---|---|
| write latency (p50/p95/p99) | базовое сравнение |
| read latency, random vs sequential | разделить эффекты доступа |
| throughput (ops/s, MB/s) | ёмкость |
| WAL append latency и bytes-per-transaction | стоимость durability |
| **recovery time** после контролируемого kill | ключевое отличие движков |
| buffer pool hit rate | где физический I/O исчезает |
| startup / open time | применимость к realtime |
| CPU, RAM, размер на диске | стоимость ресурсов |
| lock contention и deadlock rate | корректность concurrency |
| cgo/IPC overhead на операцию | цена межъязыковой границы |

**Текущее состояние измерений:**

| Величина | Значение |
|---|---|
| ZenthrilDB: write/read/recovery/throughput latency | **НЕ ИЗМЕРЕНО** |
| ZenthrilDB: recovery time | **НЕ ИЗМЕРЕНО** |
| Сравнение PostgreSQL vs ZenthrilDB | **НЕ ИЗМЕРЕНО** |
| ZenthrilDB tests | **PASS, exit 0** (проверено) |
| Существующий отчёт Zenthril `research/results/RESEARCH_REPORT.md` | заявляет 0.207 ms INSERT, P95 98.7 ms @500 users, 14 823 msg/s на HP ProBook 455 G8 / Ryzen 5 5600U / 16 GB. **В этой сессии не верифицирован и для сравнения непригоден** |

**Существенное замечание о сопоставимости:** железо, на котором получены
замеры Zenthril (Ryzen 5 5600U, 16 GB), — того же класса, что и машина
анализа. Это делает будущее сравнение корректным, если фиксировать конфигурацию.

---

## J. НАУЧНОЕ НАПРАВЛЕНИЕ (после анализа кода)

Предварительная формулировка «интеграция специализированного локального
storage engine с распределённым realtime-приложением» **коду не
соответствует**, потому что в Zenthril нет ни распределённого storage-слоя,
ни абстракции хранилища. Возможные темы, вытекающие из фактов:

1. **Стоимость durable local storage в realtime-приложении.** Количественная
   оценка того, сколько кода и каких гарантий требуется, чтобы local-first
   клиент перестал использовать `localStorage`. Опора: реально найденный
   дефект — потеря `dhSendPrivate` при восстановлении сессии.
2. **WAL/recovery vs СУБД при локальном single-writer сценарии.** Сравнение
   crash-recovery на уровне страниц и журнала, а не SQL.
3. **Граница FFI как фактор применимости C++-движка к Go realtime-сервису.**
   Измерение cgo против IPC в контексте требований к latency.
4. **Buffer pool как определяющий фактор локальной производительности.**
   Связь hit rate с числом физических I/O при разных объёмах.
5. **Инженерный разбор: почему подсистема хранения без record layer не
   интегрируема** — формализация требований к минимальному набору
   возможностей. Это самостоятельный результат и может быть основой статьи.

Новизна не должна формулироваться до завершения Э-2/Э-3/Э-9.

---

## K. RISKS

1. **Ложная интеграция.** Попытка «просто заменить PostgreSQL» даст
   неработоспособную систему: нет record layer, индексов, типов, MVCC.
2. **Граница FFI как источник дефектов.** Формат записи, зафиксированный в
   двух языках, рассинхронизируется.
3. **Ускорение ради ускорения.** ZenthrilDB может проиграть PostgreSQL на
   всех реальных сценариях данных — и это тоже валидный результат.
4. **Смешение рантаймов.** cgo + асинхронный Go может дать блокировки.
5. **Потеря данных при незрелом recovery.** Recovery не доказан экспериментом;
   production-подобное использование недопустимо до Э-3.
6. **Лицензирование и атрибуция.** Оба репозитория принадлежат одному
   автору. Zenthril обозначен как source-available, но не публично
   лицензированный. Публикация требует корректной атрибуции и указания,
   какие части существовали ранее, а какие созданы в ходе исследования.
7. **Скомпилированные артефакты в репозитории** ZenthrilDB: `.obj` и `.exe`
   лежат в `build/` и в корне. Требует гигиены репозитория, иначе
   загрязняет диффы и сборку.

---

## L. ЧТО НЕ ТРОГАТЬ (现阶段)

- **PostgreSQL — не удалять.** Он держит 25 таблиц, JOIN, внешние ключи,
  составные индексы и ACID.
- **Redis — не трогать.** `refresh:`, `security:`, `ws:` — это горячее
  эфемерное состояние с TTL; файловый движок здесь хуже.
- **E2EE-криптографию.** Изменение формата записи не должно менять
  криптографические примитивы. (Напоминание: клиент и Go сейчас выводят
  несовместимый материал — это отдельный открытый дефект F-09, не связанный
  с ZenthrilDB.)
- **Sharding** (`internal/repository/shard.go`) — ортогонален локальному
  single-node движку.
- **Публичные API** и HTTP-коды ошибок.
- **Типовые границы Go↔C++** — до этапа Э-7.

---

## M. IMPLEMENTATION ROADMAP

| Phase | Содержание | Выход |
|---|---|---|
| **Phase 0 — code audit** | **выполнено**: обе архитектуры, матрица состояний, точки интеграции | этот документ |
| Phase 1 — storage interface | узкий record-ориентированный интерфейс в Zenthril + реализация поверх pgx (проверка, что абстракция выражает реальные потребности) | рабочий интерфейс без изменения поведения |
| Phase 2 — adapter | C-ABI обёртка ZenthrilDB + адаптер, реализующий только `open/close` | адаптер собирается |
| Phase 3 — minimal integration | **record layer** в ZenthrilDB (heap-страницы со slot directory) + первый набор операций | write/read/reopen на записях |
| Phase 4 — tests | Э-1, Э-4, Э-5: формат, деградация, конкуренция | зелёные тесты на реальном поведении |
| Phase 5 — benchmarks | Э-2, Э-6, Э-8, Э-9 с фиксацией конфигурации | измеренные числа вместо «НЕ ИЗМЕРЕНО» |
| Phase 6 — failure testing | Э-3: контролируемый kill, recovery, сверка | доказательство или опровержение durability |
| Phase 7 — analysis | обработка raw-результатов, выводы | отчёт с числами |
| Phase 8 — paper | формулировка новизны **после** Phase 7 | черновик |

---

## КРИТЕРИЙ УСПЕХА — четыре ответа

1. **Где именно встроить ZenthrilDB?** Сегодня — только как отдельный
   исследовательский компонент (Этап 1), а после построения record layer —
   в client-side local-first хранилище состояния E2EE (IP-1) и, возможно,
   в `device_session_skipped_keys` (IP-4). **Не** в users/messages/guilds.
2. **Какие изменения необходимы?** Record layer, типовая система с
   версионированием, индекс, storage-интерфейс в Zenthril, граница
   Go↔C++, доказательство atomicity. Полный список — раздел G.
3. **Что реально измерить?** Уже сейчас: append-латентность WAL, recovery
   time после kill, hit rate буферного пула, поведение LockManager под
   конкуренцией, overhead границы. Всё перечисленное — НЕ ИЗМЕРЕНО и
   может быть измерено на имеющемся железе.
4. **Как сформировать публикацию?** Через количественную оценку стоимости
   durable local storage (тема 1) и/или через формализацию минимальных
   требований к интегрируемому движку (тема 5). Обе темы опираются на уже
   найденные факты, а не на предположения.

---

## ПРИЛОЖЕНИЕ. Что было выполнено фактически на этом этапе

| Действие | Результат |
|---|---|
| Прочитан `zenthrildb-push/README.md`, `CMakeLists.txt`, `Page.hpp`, `TableMetadata.hpp`, `PageManager.hpp`, `LogRecord.hpp`, `Transaction.hpp`, `LockTypes.hpp` | установлено отсутствие record layer при полноте страничного слоя |
| `git -C zenthrildb-push log/status/remote` | 2 коммита, `c7bdb5e`, рабочее дерево чистое |
| **Запуск `zenthrildb_tests.exe`** | **«All ZenthrilDB tests passed», exit 0** |
| Извлечены 25 таблиц из 15 миграций Zenthril | полная карта данных |
| Прочитаны `db/db.go`, `internal/repository/shard.go`, `message/service.go`, `user/service.go` | доказано отсутствие storage-абстракции |
| Извлечены namespaces Redis (`refresh:`, `security:`, `ws:`) | определена граница с Redis |

**Код не изменялся. Замеров производительности не проводилось.**
