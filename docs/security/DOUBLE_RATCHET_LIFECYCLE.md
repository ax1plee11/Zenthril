# Double Ratchet: фактическая семантика и lifecycle

**Статус:** доказано кодом и тестами. Криптографический код **не изменён**.
**Дата:** 01.10.2026
**Ревизия:** `6553816` + `doubleRatchetLifecycle.test.ts`
**Связанные документы:** `E2EE_ARCHITECTURE_AUDIT.md`, `CRY-203_INVESTIGATION.md`

---

## PROVEN

### P-1. Мест, задающих реальный DH-материал, ровно два

Исчерпывающий поиск по всем production-файлам (`*.ts`, `*.tsx`, исключая
`.test.`/`.spec.`) записей в `dhSendPublic` / `dhSendPrivate`:

| Строка | Место | Семантика |
|---|---|---|
| `pairwiseSession.ts:425-426` | `initializeDHRatchet` | создание сессии |
| `pairwiseSession.ts:450-451` | `performDHRatchetTurn` | **только приём** |

Остальные записи — `serialize` (`:106`), `restore` (`:131`), `copyState`
(`:335-336`), `deriveSessionState` (`:394-395`) и объявления типа (`:42-43`,
`types.ts:24`). Ни одна из них не задаёт новый ключевой материал.

### P-2. `performDHRatchetTurn` вызывается ровно из одного места

`messageEnvelopes.ts:117`, внутри `decryptChannelMessage`. В send-path
(`:59` — только `nextSendMessageKey`, `:74` — неизменённый `state.dhSendPublic`)
вызова нет.

### P-3. Направление DH-ратчета — **приёмное, каскадное**

`dhRatchetTurn` (`crypto/ratchet.ts:38-72`) выполняет:
`DH(dhSendPrivate, newPeer)` → root KDF → новая **receive** chain; затем
генерирует новую пару, `DH(newPriv, newPeer)` → root KDF → новая **send** chain;
новая пара становится `dhSendPrivate`/`dhSendPublic`.

Это структурно соответствует документированной модели Double Ratchet: сторона,
**получившая** новый публичный ключ, выполняет шаг и генерирует новую пару для
отправки. Отправитель инициировать шаг не должен — он делает это в ответ на
обнаруженное изменение.

**Следовательно, P-3 не является нарушением модели Double Ratchet.**

### P-4. Но каскад никогда не стартует

Установлено по bootstrap-коду:

| Сторона | `dhSendPublic` после bootstrap | Что пир сохранил как `dhRecvPublic` |
|---|---|---|
| Инициатор | свежая пара `dhSend` (`:190`, публикуется в `header.senderDHPublicKey`) | `peer.signed_pre_key` (`:217`) |
| Получатель | `local.signedPreKey.publicKey` (`:265`) | `header.senderDHPublicKey` (`:261`) |

Значит у каждой стороны `dhSendPublic` **побайтово равен** `dhRecvPublic` пира.
При получении первого сообщения условие
`envelope.dhPublicKey !== state.dhRecvPublic` (`:116`) ложно, DH-шаг не
выполняется. То же симметрично.

Тест `bootstrap leaves each side's send key equal to what the peer already
expects` подтверждает равенство обеих пар.

### P-5. DH-шаг выполняется ровно один раз за сессию

Тест `never advances the DH ratchet during ordinary bidirectional traffic`:
после 5 итераций «отправил → получил → ответил → получил» на обеих сторонах
`rootKey` и `dhSendPublic` **побайтово не изменились**.

### P-6. Post-compromise security по оси DH не обеспечивается

Тест `post-compromise: a captured state still predicts future keys`:
снимок полного состояния Alice, сделанный после двух сообщений, при
продвижении на два шага вперёд выдаёт **те же** `messageKey`, что и живая
сессия. Аналогичный тест подтверждает совпадение и по receive-цепочке.

В работающем Double Ratchet эти последовательности обязаны расходиться после
шага ратчета. Расхождения нет, потому что шага нет.

---

## CURRENT PROTOCOL

Обозначения: `RK` — root key, `CKs`/`CKr` — цепочки отправки/приёма,
`DHs⁺/DHs` — приватный/публичный DH отправки, `DHr` — публичный DH пира,
`Ns`/`Nr` — счётчики.

| Событие | Alice state | Bob state |
|---|---|---|
| session init | `RK=K0`, `CKs=C0`, `CKr=C1`, `DHs⁺=a`, `DHs=A`, `DHr=Bspk`, `Ns=Nr=0` | `RK=K0`, `CKs=C1`, `CKr=C0`, `DHs⁺=Bspk⁺`, `DHs=Bspk`, `DHr=A`, `Ns=Nr=0` |
| Alice send #1 | `CKs` продвигается, `Ns=1`; в конверт `dh_public_key=A` | — |
| Bob receive #1 | — | `dhPublicKey(A)==DHr(A)` → **шага нет**; `CKr` продвигается, `Nr=1` |
| Bob send #1 | — | `CKs` продвигается, `Ns=1`; `dh_public_key=Bspk` |
| Alice receive #1 | `dhPublicKey(Bspk)==DHr(Bspk)` → **шага нет**; `CKr` продвигается, `Nr=1` | — |
| DH change | **не наступает**: ни одна сторона не генерирует новую пару вне шага | то же |
| restart | `DHs⁺` теряется, остальное восстанавливается | то же |

`dhRecvPrivate` в модели **отсутствует**: приватный DH-ключ пира никогда не
хранится и не используется. Для DH-шага используется собственный `DHs⁺` с
публичным ключом пира.

---

## SECURITY FINDING

### SF-1 — Post-compromise security по оси DH не обеспечивается (HIGH)

**Суть.** Единственный механизм, который в Double Ratchet обеспечивает
восстановление после компрометации состояния, — это DH-шаг: он вводит свежий
секрет, недоступный атакующему, даже если тот знает всё предыдущее состояние.
В текущей реализации этот шаг выполняется ровно один раз, при bootstrap, и
никогда более. Поэтому компрометация состояния сессии **не имеет срока
 действия**: атакующий, получивший снимок, предсказывает все будущие ключи
неограниченно долго.

**Доказательство.** P-3, P-4, P-5, P-6. Тест `post-compromise: a captured
state still predicts future keys` проходит, то есть нерасхождение
воспроизведено.

**Нарушенный инвариант.** «Скомпрометированное состояние перестаёт
предсказывать будущие ключи после ратчета».

**Что НЕ нарушено.** Forward secrecy по симметричной оси работает: скомпрометирован
`messageKey` конкретного сообщения, но `CK` продвигается, поэтому ключи
следующих сообщений из старого состояния не выводятся без доступа к
`rootKey`/цепочке. Компрометация **состояния сессии** — другой случай.

**Что требуется для исправления.** Одно из:

- генерация новой DH-пары при каждом отправленном сообщении или по расписанию
  (ротация на стороне отправителя);
- генерация новой DH-пары при выполнении симметричного шага;
- иная схема, при которой каскад стартует.

Любой из вариантов изменяет wire-format: поле `dh_public_key` начнёт меняться,
а получатель начнёт выполнять шаг. Это изменение протокола и требует решения
владельца.

**Почему это не сломало работу.** Проблема не проявляется, потому что
отсутствие шага не нарушает согласованность: обе стороны используют одни и те
же неизменные DH-ключи, поэтому расшифровка работает. Деградация
функциональная, а не операционная.

### SF-2 — `dhSendPrivate` не переживает restart (HIGH, CRY-203)

**Суть.** Поле отсутствует в персистированном представлении. После restart
`performDHRatchetTurn` отказывает.

**Следствие, установленное в этом раунде.** Пока SF-1 не исправлен, DH-шаг
не происходит вовсе, поэтому **SF-2 не эксплуатируется**: нечего
восстанавливать. SF-2 станет эксплуатируемым **сразу после** исправления SF-1.

Это меняет приоритет: исправлять только SF-2 бессмысленно, а исправлять SF-1
без решения по SF-2 нельзя — restart переведёт SF-2 из латентного в
блокирующее состояние.

Тест `the persisted representation carries no dhSendPrivate` подтверждает
отсутствие поля; тесты в `pairwiseSession.restart.test.ts` подтверждают
fail-closed.

### SF-3 — Отсутствует защита от rollback и повторного использования старого DH-ключа (MEDIUM)

`decryptChannelMessage` (`:116`) сравнивает `envelope.dhPublicKey` с
`state.dhRecvPublic`. Условие — «ключ отличается» или «ключ совпадает». Нет
проверки, что встреченный ключ **новее** ранее виденного: нет ни счётчика, ни
списка ранее принятых DH-ключей.

Пока SF-1 не исправлен, поле `dh_public_key` постоянно, поэтому эксплуатация
невозможна. После исправления SF-1 потребуется явная защита: иначе ранее
перехваченный конверт с прежним `dh_public_key` может быть предъявлен повторно.

Проверки `sessionId`, `protocol_version`, `cipherSuite` в конверте присутствуют
(`validateHeader`, `validateSessionState` соответственно), то есть привязка к
сессии и версии есть, а вот защита от повторного DH-ключа — нет.

---

## CRY-203: что именно ломается после restart

| Аспект | Состояние |
|---|---|
| Симметричные цепочки | работают |
| Оба направления | работают |
| Skipped keys | работают |
| Replay-защита | работает |
| Стабильность `dhSendPublic` | работает, ложной ротации нет |
| DH-шаг | **отказ** |
| Симптом у пользователя | отсутствует, пока SF-1 не исправлен |

**Вывод.** CRY-203 в текущем состоянии проекта — латентная проблема. Она не
блокирует работу мессенджера. Её целесообразно решать вместе с SF-1, иначе
решение будет бессмысленным.

---

## OPTIONS для CRY-203

Выбор не производится.

### A. Персистентность приватного DH-ключа

| | |
|---|---|
| Изменения | `types.ts`, `serializePairwiseSession`, `restorePairwiseSession`, `deviceKeyStore.ts`, тесты |
| Хранилище | существующий Tauri keyring. `localStorageAdapter` **не расширяется** — он помечен `"insecure-localstorage"` (`deviceKeyStore.ts:177`) |
| Новый класс экспозиции | нет: бандл уже содержит `signedPreKey.secretKey`, `oneTimePreKeys[].secretKey`, `sendChainKey` |
| Проверка полноты | сериализация должна сохранить `rootKey`, `dhSendPrivate`, `dhSendPublic`, `dhRecvPublic`, `sendChainKey`, `receiveChainKey`, `sendCounter`, `receiveCounter`, `skippedKeys`, `sessionId`, `version`; `cipherSuite` в persisted-сессии **отсутствует** и должен быть либо добавлен, либо явно признан принадлежащим заголовку |
| Риск | компрометация локального хранилища даёт и DH-ключ, и цепочки; однако цепочки уже там лежат |
| Условие | имеет смысл **только** после исправления SF-1 |

### B. Повторная X3DH-сессия после restart

| | |
|---|---|
| Старая сессия | остаётся в бандле; без механизма сброса — повторно отвергает тот же конверт |
| Смена `sessionId` | новая сессия получает новый `sessionId`; пир определяет смену по заголовку `bootstrap_header` |
| Downgrade | атакующий, подсовывающий старый `bootstrap_header`, вызывает `acceptPairwiseSession`; защита — `recipientSignedPreKeyId` и одноразовость prekey (`pairwiseSession.ts:238-247`) |
| Старые сообщения | при новой сессии конверты не расшифровываются, поскольку `sessionId` другой |
| Skipped keys | относятся к старой сессии и не переносятся |
| Механизм в коде | **отсутствует**: `deletePairwiseSession`, `resetSession`, `rekey` — не найдено |
| Риск | потеря доступа к истории, расход одноразовых prekey |

### C. Иной механизм

Существующих в проекте механизмов восстановления DH-состояния **не найдено**.
Новый протокол не предлагается.

---

## TESTS

Файл `client/src/features/e2ee/doubleRatchetLifecycle.test.ts`, 7 тестов,
все проходят:

| Тест | Что доказывает |
|---|---|
| bootstrap leaves each side's send key equal to what the peer already expects | P-4 |
| never advances the DH ratchet during ordinary bidirectional traffic | P-5 |
| keeps the symmetric chains working across many messages | согласованность не нарушена |
| post-compromise: a captured state still predicts future keys | **SF-1** |
| post-compromise: a captured state also reproduces the live receive chain | **SF-1** |
| a DH turn is reachable when a peer genuinely advertises a different key | механизм работает, нужен триггер |
| the persisted representation carries no dhSendPrivate | **SF-2** |

Ранее добавленные: `pairwiseSession.restart.test.ts`, 10 тестов — все проходят.

**Проверки:** `npm test` — **31 файл / 202 теста passed**; `eslint` по новому
файлу — exit 0; `tsc --noEmit` — без ошибок в новом файле; `go build ./...` —
exit 0; `go test ./...` — exit 0.

---

## NOT IMPLEMENTED

| Пункт | Причина |
|---|---|
| Ротация DH на стороне отправителя | Изменение wire-format; требует решения владельца |
| Защита от replay старого `dh_public_key` | Следствие SF-1; бессмысленно до исправления |
| Персистентность `dhSendPrivate` | Вариант A/B не выбран |
| Механизм сброса сессии | Отсутствует в проекте |
| `cipherSuite` в persisted-сессии | Не добавлен |
| `moderation.ExtractRealIP` (SEC-401) | Не исправлен; см. `CRY-203_INVESTIGATION.md` §9 |
| Go E2EE | Не подключается; отдельное архитектурное решение |
| Таблицы `device_sessions`, `device_session_skipped_keys` | Не удаляются |

---

## GIT

Коммит с тестами и этим документом: `test(e2ee): document Double Ratchet lifecycle` и
`docs(security): record Double Ratchet lifecycle findings`.

Статус production E2EE не изменён: `client encrypts → server stores opaque data`.
