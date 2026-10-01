# CRY-101 — E2EE Protocol Compatibility Analysis

**Статус:** read-only анализ. Криптографический код не изменён.
**Дата:** 01.10.2026
**Ревизия:** `08a3a49`
**Назначение документа:** зафиксировать фактическое состояние двух реализаций E2EE
и подготовить данные для решения владельца протокола.

> Документ намеренно **не** выбирает нормативную схему. Раздел 10 содержит
> варианты без рекомендации.

## 1. Current implementation

В проекте существуют две независимые реализации протокола сквозного шифрования.

| Реализация | Язык | Расположение | Роль |
|---|---|---|---|
| Client | TypeScript | `client/src/crypto/`, `client/src/features/e2ee/` | Фактический producer и consumer шифротекста |
| Server | Go | `backend/internal/crypto/` | Хранение состояния сессий, выдача key bundles, relay |

Обе реализации заявляют поддержку X3DH и Double Ratchet, но несовместимы.

Существующее состояние зафиксировано ранее (`ZENTHRIL_COMPLETION_PLAN.md`, CRY-101).

## 2. Go cryptographic pipeline

### 2.1 X3DH — `backend/internal/crypto/x25519.go`

| Параметр | Значение | Место |
|---|---|---|
| DH1 | `DH(IK, SPK)` — identity private × peer signed prekey | `x25519.go:57` |
| DH2 | `DH(EK, IK)` — ephemeral private × peer identity DH | `x25519.go:61` |
| DH3 | `DH(EK, SPK)` — ephemeral × peer signed prekey | `x25519.go:65` |
| DH4 | `DH(EK, OPK)` — добавляется только при наличии one-time prekey | `x25519.go:72` |
| IKM | `concat(DH1, DH2, DH3[, DH4])` | `x25519.go:79` |
| KDF | HKDF-SHA256 | `ratchet.go:354` |
| **Salt** | **`make([]byte, 32)` — 32 нулевых байта** | `x25519.go:81` |
| **Info** | **`"Zenthril X3DH v1"`** | `x25519.go:82` |
| Output length | **32 байта (только shared secret)** | `x25519.go:82` |
| Encoding | raw bytes, без base64 внутри пайплайна | — |

### 2.2 Initial ratchet derivation — `ratchet.go:89`

| Параметр | Значение |
|---|---|
| IKM | результат X3DH (32 байта) |
| **Salt** | **`nil`** |
| **Info** | **`"zenthril-ratchet-v1:init:" + sessionInfo`** |
| sessionInfo | `"senderUID:senderDevice->recipientUID:recipientDevice"` (`protocol.go:124`) |
| Output length | 96 байт |
| Output split | `rootKey[0:32]`, `first[32:64]`, `second[64:96]` |

### 2.3 Root ratchet — `ratchet.go:108`

| Параметр | Значение |
|---|---|
| IKM | `dhOutput` |
| **Salt** | **`rootKey`** |
| **Info** | **`"zenthril-ratchet-v1:root"`** |
| Output length | 64 байта |
| Output split | `rootKey[0:32]`, `chainKey[32:64]` |

### 2.4 Symmetric chain — `ratchet.go:291`

| Параметр | Значение |
|---|---|
| IKM | `chainKey` |
| **Salt** | **`nil`** |
| **Info** | **`"zenthril-ratchet-v1:chain"`** |
| Output length | 76 байт (32 + 12 + 32) |
| **Output split** | **`messageKey[0:32]`, `nonce[32:44]`, `nextChainKey[44:76]`** |

### 2.5 Initial DH step — `protocol.go:132-154`

Go **не имеет отдельной стадии initial-DH**. Инициатор выполняет
`RootRatchet(ratchet.RootKey, dhOutput)`, то есть использует info
`"zenthril-ratchet-v1:root"` — тот же, что и для обычного DH-шага.

### 2.6 DH ratchet turn — `ratchet.go:135`

| Шаг | Действие |
|---|---|
| 1 | `DH(DHSendPrivate, newPeerDHPublic)` |
| 2 | `RootRatchet` → новая receive chain |
| 3 | генерация новой DH-пары |
| 4 | `DH(newPrivate, newPeerDHPublic)` |
| 5 | `RootRatchet` → новая send chain |

### 2.7 Ответная сторона X3DH — **ОТСУТСТВУЕТ**

`X3DHService` содержит единственный метод:

```
X3DHService.StartSession(...)      protocol.go:82
```

Метода принятия сессии не существует. Go не может выступить в роли получателя
X3DH-сессии.

### 2.8 AAD и шифрование — `message.go`

AAD передаётся вызывающим кодом (`EncryptMessage(state, plaintext, aad)`).
Формирование AAD в Go не найдено.

## 3. Client cryptographic pipeline

### 3.1 X3DH и session derivation — `client/src/features/e2ee/pairwiseSession.ts`

Инициатор (`:207-215`):

| Параметр | Значение |
|---|---|
| DH1 | `DH(IK, SPK)` (`:208`) |
| DH2 | `DH(EK, IK)` (`:209`) |
| DH3 | `DH(EK, SPK)` (`:210`) |
| DH4 | `DH(EK, OPK)` при наличии OPK (`:213`) |

Получатель (`:252-257`):

| Параметр | Значение |
|---|---|
| DH1 | `DH(SPK, IK)` (`:252`) — зеркально, значение совпадает по симметрии DH |
| DH2 | `DH(IK, EK)` (`:253`) |
| DH3 | `DH(SPK, EK)` (`:254`) |
| DH4 | `DH(OPK, EK)` (`:257`) |

### 3.2 Session derivation — `pairwiseSession.ts:368`

| Параметр | Значение |
|---|---|
| IKM | `concat(dhValues)` |
| **Salt** | **`"zenthril.e2ee.x3dh.root.v1"`** (`:12`) |
| **Info** | **`"zenthril.e2ee.x3dh.session.v1|" + canonicalHeader(header)`** (`:13`, `:378`) |
| Output length | **96 байт** |
| **Output split** | **`rootKey[0:32]`, `initiatorChain[32:64]`, `receiverChain[64:96]`** (`:381-383`) |

Отдельной стадии initial ratchet у клиента **нет**: функция
`DeriveInitialRatchetState` отсутствует.

### 3.3 canonicalHeader — `pairwiseSession.ts:526`

`JSON.stringify` с полями в алфавитном порядке ключей:
`cipherSuite`, `recipientDeviceId`, `recipientOneTimePreKeyId` (или `null`),
`recipientSignedPreKeyId`, `recipientUserId`, `senderDeviceId`,
`senderDHPublicKey`, `senderEphemeralPublicKey`, `senderIdentityDHPublicKey`,
`senderUserId`, `sessionId`, `version`.

### 3.4 Initial DH step — `pairwiseSession.ts:406-414`

| Параметр | Значение |
|---|---|
| IKM | `dhOutput` |
| Salt | `state.rootKey` |
| **Info** | **`"zenthril-ratchet-v1:init-dh"`** (`:412`) |
| Output length | 64 байта |
| Output split | `newRootKey[0:32]`, `newChainKey[32:64]` |

### 3.5 Root ratchet — `client/src/crypto/ratchet.ts:74`

| Параметр | Значение |
|---|---|
| IKM | `dhOutput` |
| **Salt** | **`rootKey`** |
| **Info** | **`"zenthril-ratchet-v1:root"`** |
| Output length | 64 байта |
| Output split | `newRootKey[0:32]`, `newChainKey[32:64]` |

### 3.6 Symmetric chain — `client/src/crypto/ratchet.ts:21`

| Параметр | Значение |
|---|---|
| IKM | `chainKey` |
| **Salt** | **`"zenthril-ratchet-v1"`** (`:5`) |
| **Info** | **`"chain"`** (`:6`) |
| Output length | 76 байт |
| **Output split** | **`newChainKey[0:32]`, `messageKey[32:64]`, `messageNonce[64:76]`** (`:28-30`) |

### 3.7 DH ratchet turn — `client/src/crypto/ratchet.ts:38`

Структура совпадает с Go: DH → root ratchet → receive chain; генерация пары →
DH → root ratchet → send chain.

### 3.8 Константы протокола

| Константа | Значение | Место |
|---|---|---|
| `PAIRWISE_SESSION_PROTOCOL_VERSION` | `1` | `pairwiseSession.ts:9` |
| `PAIRWISE_SESSION_CIPHER_SUITE` | `"X3DH-HKDF-SHA256-DR-v1"` | `pairwiseSession.ts:10` |

## 4. Side-by-side comparison

| STEP | Go | Client | Совпадает |
|---|---|---|---|
| DH1 | `DH(IK, SPK)` | `DH(IK, SPK)` | **ДА** |
| DH2 | `DH(EK, IK)` | `DH(EK, IK)` | **ДА** |
| DH3 | `DH(EK, SPK)` | `DH(EK, SPK)` | **ДА** |
| DH4 | `DH(EK, OPK)` | `DH(EK, OPK)` | **ДА** |
| Порядок конкатенации | `DH1‖DH2‖DH3[‖DH4]` | `DH1‖DH2‖DH3[‖DH4]` | **ДА** |
| X3DH salt | 32 нулевых байта | `"zenthril.e2ee.x3dh.root.v1"` | **НЕТ** |
| X3DH info | `"Zenthril X3DH v1"` | `"zenthril.e2ee.x3dh.session.v1\|"+header` | **НЕТ** |
| X3DH output | 32 байта | 96 байт | **НЕТ** |
| Стадия initial ratchet | есть | отсутствует | **НЕТ** |
| Initial-ratchet salt | `nil` | — (внутри X3DH) | **НЕТ** |
| Initial-ratchet info | `"zenthril-ratchet-v1:init:"+sessionInfo` | — | **НЕТ** |
| sessionInfo | `"uid:dev->uid:dev"` | `canonicalHeader(header)` | **НЕТ** |
| Initial DH info | `"zenthril-ratchet-v1:root"` | `"zenthril-ratchet-v1:init-dh"` | **НЕТ** |
| Root ratchet salt | `rootKey` | `rootKey` | **ДА** |
| Root ratchet info | `"zenthril-ratchet-v1:root"` | `"zenthril-ratchet-v1:root"` | **ДА** |
| Root ratchet split | `root‖chain` | `root‖chain` | **ДА** |
| Chain salt | `nil` | `"zenthril-ratchet-v1"` | **НЕТ** |
| Chain info | `"zenthril-ratchet-v1:chain"` | `"chain"` | **НЕТ** |
| Chain split | `key‖nonce‖nextChain` | `nextChain‖key‖nonce` | **НЕТ** |
| DH ratchet turn | 2 шага | 2 шага | **ДА** |
| Ответная сторона X3DH | **отсутствует** | присутствует | **НЕТ** |
| Формирование AAD | **не найдено** | `messageAAD.ts` | **НЕТ** |

## 5. First incompatibility

**Первое расхождение — KDF на выходе X3DH**, файл `backend/internal/crypto/x25519.go:81-82`
против `client/src/features/e2ee/pairwiseSession.ts:12-13, 376-378`.

Обе реализации получают одинаковый IKM при одинаковых входных ключах, но
применяют разные salt и info, поэтому получают разный выход уже на первом шаге.

До этой точки реализации эквивалентны: порядок DH-вычислений, набор DH и их
конкатенация совпадают. Расхождение начинается исключительно в HKDF.

**Доказательство расхождения без исполнения кода:** `hkdfBytes(ikm, salt, info, n)`
при фиксированном `ikm` является детерминированной функцией от пары `(salt, info)`.
Поскольку пары различаются, выходы обязаны различаться с вероятностью 1.

## 6. All incompatibilities

Перечислены в порядке следования конвейера. I-1 является первопричиной,
остальные независимы и сохранятся после устранения I-1.

| ID | Расхождение | Go | Client | Устранение независимо от I-1? |
|---|---|---|---|---|
| I-1 | X3DH salt и info | `00×32`, `"Zenthril X3DH v1"` | `"zenthril.e2ee.x3dh.root.v1"`, `"...session.v1\|"+header` | нет |
| I-2 | Число стадий вывода корня | 2 (X3DH → init ratchet) | 1 (X3DH сразу даёт корень и цепи) | нет |
| I-3 | Состав sessionInfo | строка идентификаторов | canonical header (12 полей) | нет |
| I-4 | Info начального DH-шага | `"…:root"` | `"…:init-dh"` | нет |
| I-5 | Chain salt | `nil` | `"zenthril-ratchet-v1"` | нет |
| I-6 | Chain info | `"zenthril-ratchet-v1:chain"` | `"chain"` | нет |
| I-7 | Порядок сегментов chain | `key‖nonce‖next` | `next‖key‖nonce` | нет |
| I-8 | Ответная сторона X3DH в Go | отсутствует | присутствует | да |
| I-9 | Формирование AAD в Go | не найдено | `messageAAD.ts` | да |
| I-10 | Именование набора | `X25519-HKDF-SHA256-AES-256-GCM` (`client/src/crypto/index.ts:8`) | `X3DH-HKDF-SHA256-DR-v1` (`pairwiseSession.ts:10`) | да |

I-4, I-5, I-6, I-7 не влияют друг на друга, но все они становятся
недостижимыми, пока не устранён I-1: разные корневые ключи делают любые
последующие сравнения бессмысленными.

## 7. Consequences

1. **Сквозное шифрование не работает между реализациями.** Сессия, созданная
   клиентом, не может быть восстановлена сервером, и наоборот.
2. **Серверная часть E2EE не используется в рабочем пути.** Поскольку
   фактический producer и consumer — клиент, работающая система использует
   только клиентскую реализацию. Go-реализация обслуживает хранение состояния
   и выдачу key bundles.
3. **Зелёный тестовый набор не является доказательством.** Все тесты Go проверяют
   Go против Go, все тесты клиента — клиент против клиента. Расхождение не
   обнаруживается ни одним существующим тестом.
4. **Заявление о совместимости недопустимо.** Ни с Signal, ни с любой другой
   реализацией совместимость не доказана.
5. **Существующие сессии придётся пересоздать** при любом варианте унификации,
   поскольку изменяются корневые ключи и цепочки.

## 8. Persisted DH private key issue

Отдельная задача **CRY-203**. Здесь фиксируется только фактическое состояние.

`PairwiseSessionState` содержит `dhSendPrivate` в памяти, однако функция
`serializePairwiseSession` его **не сохраняет**. При восстановлении
устанавливается `new Uint8Array(32)`, то есть нули.

Проверено, что `performDHRatchetTurn` корректно fail-closed: добавленная ранее
проверка `hasUsableDHPrivateKey` отклоняет нулевой ключ вместо того, чтобы
выполнить X25519 с публично известным клампом.

Практическое следствие: **после перезапуска клиента DH-шаг ратчета невозможен**.
Сессия продолжает работать только для уже установленных цепочек.

Дополнительно: `initializeDHRatchet` выполнял присваивание `dhSendPrivate` по
ссылке на буфер, который `initiatePairwiseSession` обнулял в блоке `finally`.
Это уже исправлено копированием ключевого материала.

Варианты стратегии (A — шифрованное локальное хранение, B — повторный
bootstrap, C — протокол восстановления сессии) **не выбираются в этом
документе**; решение принадлежит владельцу протокола.

## 9. Current test coverage

| Проверка | Go | Client |
|---|---|---|
| Внутренняя согласованность | есть: `ratchet_test.go`, `message_test.go`, `protocol_test.go` | есть: `ratchet.test.ts`, `pairwiseSession.test.ts` |
| Alice → Bob → Alice | есть на стороне Go | есть на стороне клиента |
| Out-of-order / skipped | есть на стороне Go | 9 тестов, добавлены в раунде CRY-007 |
| Официальные test vectors (RFC 7748 / 5869 / 8439) | **отсутствуют** | **отсутствуют** |
| **Кросс-реализационные vectors** | **отсутствуют** | **отсутствуют** |
| Совместимость парных ключей | проверяется только против себя самой | проверяется только против себя самой |

`backend/internal/crypto/test_vectors_test.go` содержит проверку HKDF-детерминизма
(`hkdfBytes` против ожидаемого вывода), но не содержит векторов, разделяемых с
клиентом.

## 10. Decision required from owner

Требуется выбрать нормативную реализацию. Документ не рекомендует вариант.

### Option A — Go implementation becomes normative

**Файлы к изменению:**
- `client/src/features/e2ee/pairwiseSession.ts` — X3DH salt/info, удаление либо
  сохранение отдельной стадии initial ratchet, sessionInfo, initial-DH info
- `client/src/crypto/ratchet.ts` — chain salt/info и порядок сегментов
- `client/src/features/e2ee/pairwiseSession.ts:10` — значение cipher suite
- Требуется **реализовать** в Go ответную сторону X3DH: `X3DHService` имеет
  только `StartSession` (`protocol.go:82`)
- Требуется найти или определить место формирования AAD в Go

**Тесты к добавлению:**
- Кросс-реализационные vectors 001–010, исполняемые в обоих языках
- Ответная сторона X3DH в Go: тест принятия сессии
- Тест соответствия формирования AAD

**Риск для существующих данных:** все клиентские сессии становятся
нерасшифровываемыми. Требуется принудительный re-bootstrap.

**Версионирование:** текущий `PAIRWISE_SESSION_PROTOCOL_VERSION = 1` и
`"X3DH-HKDF-SHA256-DR-v1"` описывают клиентскую схему и должны быть
переведены на 2 с пометкой несовместимости.

### Option B — Client implementation becomes normative

**Файлы к изменению:**
- `backend/internal/crypto/x25519.go` — salt (32 нулевых байта → строка), info,
  длина выхода 32 → 96
- `backend/internal/crypto/ratchet.go:89` — удаление отдельной стадии
  `DeriveInitialRatchetState` либо слияние с X3DH; изменение sessionInfo на
  canonical header
- `backend/internal/crypto/ratchet.go:135` — info начального DH-шага на
  `"zenthril-ratchet-v1:init-dh"`
- `backend/internal/crypto/ratchet.go:291` — chain salt и info; **порядок
  сегментов** `key‖nonce‖nextChain` → `nextChain‖key‖nonce`
- `backend/internal/crypto/protocol.go` — формат canonical header должен быть
  воспроизведён побайтно, включая порядок ключей `JSON.stringify`
- Требуется реализовать формирование AAD в Go, соответствующее `messageAAD.ts`

**Тесты к добавлению:**
- Кросс-реализационные vectors 001–010
- Тест паритета canonical header: сериализация Go должна совпасть с клиентской
  побайтно
- Тест формирования AAD

**Риск для существующих данных:** все серверные состояния сессий становятся
невалидными. Учитывая, что рабочий путь проходит через клиент, фактический
ущерб для пользователей ниже.

**Версионирование:** серверные состояния сессий перестают читаться;
требуется инвалидация с явной версией схемы в записи состояния.

### Общее для обоих вариантов

1. Требуется зафиксировать protocol version и единую спецификацию: KDF, salt,
   info, кодирование, порядок полей, длины вывода, порядок DH, канонизацию
   заголовка, вывод message key и nonce, AAD.
2. Требуется написать vectors 001–010 и проверять их в **обоих** языках.
3. Изменения должны быть byte-for-byte, без «эквивалентных ключей» и
   нормализации.
4. До получения результата векторов нельзя заявлять корректность
   криптографии, совместимость или статус «сквозная проверка выполнена».
5. Название схемы при собственной реализации: «Zenthril E2EE protocol».
   «Signal Protocol» не использовать, пока совместимость с официальными
   векторами Signal не доказана.
