# E2EE Architecture Audit

**Статус:** read-only архитектурный аудит. Производственный код не изменён.
**Дата:** 01.10.2026
**Ревизия:** `afc34c6`
**Связанные документы:** `CRY-101_PROTOCOL_DECISION.md`, `CRY-101_IMPACT_MIGRATION_ANALYSIS.md`

## 1. Scope

Цель — установить по коду, какой E2EE-путь реально используется в Zenthril,
и классифицироватьGo E2EE-реализацию по достижимости. Документ не принимает
архитектурных решений.

Метод: сплошная трассировка от обработчика отправки сообщения в UI до запроса
в БД и обратно, затем поиск всех импортов и всех SQL-ссылок.

## 2. Actual Production E2EE Path

```
ChatView.handleSend                         client/src/components/ChatView.tsx:332
  ↓ prepareChannelMessage                   messageEnvelopes.ts:31
  ↓ loadDeviceKeyBundle                     deviceKeyStore.ts
  ↓ crypto.subtle.generateKey (AES-256)     messageEnvelopes.ts:38
  ↓ buildMessageAADInput                    messageAAD.ts
  ↓ encrypt(plaintext, contentKey, aad)     crypto/index.ts:160
  ↓ api.messages.recipients(channelId)      messageEnvelopes.ts:45
  ↓ найти или создать pairwise-сессию       messageEnvelopes.ts:51-58
  ↓ nextSendMessageKey                      pairwiseSession.ts
  ↓ importRatchetMessageKey                 messageEnvelopes.ts:60
  ↓ encrypt(contentKey, wrappingKey)        messageEnvelopes.ts:61
  ↓ POST /api/v1/channels/{id}/messages     ChatView.tsx:340
  ↓ message/service.go:117 INSERT           backend
  ↓ message_recipient_envelopes             backend/migrations/013
  ↓ prepared.persist() → OS keyring         ChatView.tsx:341

Обратный путь:
ChatView:246 → decryptChannelMessage       messageEnvelopes.ts:90
  ↓ найти конверт своего устройства          messageEnvelopes.ts:96
  ↓ loadPairwiseSession / acceptPairwiseSession  messageEnvelopes.ts:107-112
  ↓ performDHRatchetTurn при новом dhPublicKey    messageEnvelopes.ts:116-118
  ↓ nextReceiveMessageKey(state, ratchetCounter)  messageEnvelopes.ts:122
  ↓ decrypt(envelope.payload, wrappingKey)        messageEnvelopes.ts:127
  ↓ importKey + decrypt(payload, contentKey)     messageEnvelopes.ts:131-134
  ↓ storeDeviceKeyBundle                          messageEnvelopes.ts:135
```

**Вся E2EE-криптография выполняется на клиенте.** Сервер участвует только как
транспорт и хранилище opaque-значений.

## 3. Message Encryption Path

| Стадия | Файл:строка | Данные | Версия / набор |
|---|---|---|---|
| Ввод текста | `ChatView.tsx:333` `handleSend` | plaintext | — |
| Генерация content key | `messageEnvelopes.ts:38` | AES-256, WebCrypto, `extractable=false` | — |
| AAD | `messageEnvelopes.ts:43` → `messageAAD.ts` | channelId, senderUserId, senderDeviceId, sessionId, clientMessageId | — |
| Шифрование содержимого | `messageEnvelopes.ts:44` → `crypto/index.ts:160` | ciphertext, iv, tag | `CIPHER_SUITE_V2 = "X25519-HKDF-SHA256-AES-256-GCM"` (`crypto/index.ts:8`) |
| Получение получателей | `messageEnvelopes.ts:45` | список device_id по каналу | — |
| Поиск сессии | `messageEnvelopes.ts:51` `findPairwiseSessionForPeer` | sessionId | `version: 1` |
| Создание сессии | `messageEnvelopes.ts:54-57` | key bundle + X3DH | `X3DH-HKDF-SHA256-DR-v1` |
| Chain KDF | `messageEnvelopes.ts:59` `nextSendMessageKey` → `crypto/ratchet.ts:21` | messageKey, messageNonce | — |
| Обёртывание content key | `messageEnvelopes.ts:61` → `crypto/index.ts:160` | wrapped, nonce = messageNonce | — |
| Формирование конверта | `messageEnvelopes.ts:69-77` | session_id, ratchet_counter, dh_public_key, bootstrap_header | — |
| Сохранение состояния | `messageEnvelopes.ts:81`, `ChatView.tsx:341` | rootKey, chain keys, counters | — |

**Важное наблюдение.** Content key создаётся один раз на сообщение и оборачивается
отдельно для каждого устройства-получателя. Это соответствует схеме, при которой
устройство добавляется позже и не получает доступ к прошлым сообщениям.

## 4. Message Decryption Path

| Стадия | Файл:строка | Поведение при отказе |
|---|---|---|
| Выбор конверта | `messageEnvelopes.ts:96` | `return null` |
| Проверка счётчика | `messageEnvelopes.ts:102` | `return null` |
| Восстановление сессии | `messageEnvelopes.ts:107` | `return null` |
| Bootstrap | `messageEnvelopes.ts:108-112` | исключение пробрасывается |
| **DH-шаг** | `messageEnvelopes.ts:116-118` | **исключение пробрасывается** |
| Chain KDF + skipped keys | `messageEnvelopes.ts:122` | `return null` (`:123-125`) |
| Распаковка content key | `messageEnvelopes.ts:127` | исключение пробрасывается |
| Расшифровка содержимого | `messageEnvelopes.ts:134` | исключение пробрасывается |
| Сохранение состояния | `messageEnvelopes.ts:135` | — |

Вызывающий код: `ChatView.tsx:246` внутри `try` (`:243`) и `ChatView.tsx:441`
внутри `Promise.allSettled` (`:438`). При `text === null` оба места выбрасывают
`Error("No recipient envelope for this device")`.

## 5. X3DH Path

| Роль | Реализация | Место |
|---|---|---|
| Initiator | `initiatePairwiseSession` | `pairwiseSession.ts:181` |
| Responder | `acceptPairwiseSession` | `pairwiseSession.ts:229` |
| Проверка подписи SPK | `verifyPeerSignedPreKey` | `pairwiseSession.ts:468` |
| Получение чужого bundle | `api.keyBundles.claim` | `messageEnvelopes.ts:54` |

Обе роли реализованы **только на клиенте**. В Go существует только инициатор:
`X3DHService.StartSession` (`protocol.go:82`), и у него нет вызовов.

Порядок DH совпадает в обеих реализациях: `DH1=DH(IK,SPK)`, `DH2=DH(EK,IK)`,
`DH3=DH(EK,SPK)`, `DH4=DH(EK,OPK)`.

## 6. Double Ratchet Path

| Операция | Реализация | Место |
|---|---|---|
| Chain advance | `advanceRatchet` | `client/src/crypto/ratchet.ts:21` |
| Root ratchet | `rootRatchet` | `client/src/crypto/ratchet.ts:74` |
| DH-шаг | `dhRatchetTurn` + `performDHRatchetTurn` | `ratchet.ts:38`, `pairwiseSession.ts` |
| Skipped keys | `nextReceiveMessageKey` | `pairwiseSession.ts` |
| Fail-closed при отсутствии DH-ключа | `hasUsableDHPrivateKey` | `pairwiseSession.ts` |
| Шифрование сообщения | `encrypt` | `client/src/crypto/index.ts:160` |
| Расшифровка сообщения | `decrypt` | `client/src/crypto/index.ts:236` |

Вся криптография выполняется на WebCrypto (`crypto.subtle`) и
`@noble/hashes`, `@noble/curves`.

## 7. Session Persistence

| Свойство | Значение | Место |
|---|---|---|
| Где | `StoredDeviceKeyBundle.pairwiseSessions` | `types.ts` |
| Как | `storeDeviceKeyBundle` → Tauri `invoke("store_device_key_bundle")` | `deviceKeyStore.ts:161` |
| Резерв | `localStorageAdapter`, помечен `"insecure-localstorage"` | `deviceKeyStore.ts:175-177` |
| Сериализация | `serializePairwiseSession` | `pairwiseSession.ts:90` |
| Версионирование | поле `version: PAIRWISE_SESSION_PROTOCOL_VERSION` | `pairwiseSession.ts:97` |
| Валидация | `validateState` | `pairwiseSession.ts:506` |
| Восстановление | `restorePairwiseSession` | `pairwiseSession.ts:113` |

Сохраняемые поля: `version`, `sessionId`, `peerUserId`, `peerDeviceId`,
`rootKey`, `sendChainKey`, `receiveChainKey`, `sendCounter`, `receiveCounter`,
`dhSendPublic`, `dhRecvPublic`, `previousCounter`, `skippedMessageKeys`.

**Не сохраняется:** `dhSendPrivate`.

## 8. Backend E2EE Code

Пакет `backend/internal/crypto/` содержит:

| Файл | Размер | Содержимое |
|---|---|---|
| `x25519.go` | 2 831 | X3DH shared secret |
| `protocol.go` | 7 383 | `X3DHService`, `SessionState`, `KeyStore` |
| `ratchet.go` | 11 926 | `RootRatchet`, `DHRatchetTurn`, chain KDF |
| `message.go` | 5 129 | `EncryptMessage`, `DecryptMessage` |
| `multidevice.go` | 8 534 | `MultiDeviceService`, конверты |
| `session_store.go` | 16 836 | `SessionStore` над PostgreSQL |

Собственные тесты: `message_test.go`, `multidevice_test.go`, `protocol_test.go`,
`ratchet_test.go`, `test_vectors_test.go`.

## 9. Backend E2EE Reachability

### 9.1 Цепочка импортов

```
поиск по всем *.go вне internal/crypto:
  "zenthril-backend/internal/crypto"
  → 0 совпадений
```

**Ни один файл вне пакета не импортирует `internal/crypto` — включая тесты
других пакетов.** Пакет достижим исключительно из собственных тестов.

### 9.2 Цепочка SQL

```
таблицы device_sessions, device_session_skipped_keys
  ← SQL-ссылки только в internal/crypto/session_store.go
      (строки 150, 315, 334, 398, 422, 465, 480)
  ← session_store.go не импортируется никем
  ← СЛЕДСТВИЕ: недостижимо
```

### 9.3 Проверка точек подключения

| Проверка | Результат |
|---|---|
| Конструкторы в `main.go` | `security.NewGuard`, `spam.NewGuard`, `device.NewService`, `message.NewService` — конструкторов `internal/crypto` нет |
| DI-контейнер `internal/app/container.go` | `internal/crypto` отсутствует |
| HTTP-обработчики | Не используют `internal/crypto` |
| WebSocket `hub` | Не использует `internal/crypto` |
| Фоновые воркеры `cmd/worker` | Не используют `internal/crypto` |
| `cmd/sfu` | Не использует `internal/crypto` |

### 9.4 Классификация

| Категория | Содержимое | Основание |
|---|---|---|
| **A. Production reachable** | Ничего | 0 импортов вне пакета |
| **B. Test-only** | `x25519.go`, `protocol.go`, `ratchet.go`, `message.go`, `multidevice.go` через собственные 5 тестовых файлов | тесты компилируются и проходят |
| **C. Unreachable / dead** | `session_store.go` | не покрыт ни одним тестом и не импортируется |
| **D. Planned / future** | Не доказано | намерение не документировано в коде; таблицы `014` и комментарии намекают, но утверждать нельзя |
| **E. Legacy** | Не доказано | история репозитория не анализировалась для этой классификации |

## 10. E2EE Database Tables

| Таблица | Миграция | Модель | Репозиторий | INSERT | SELECT | UPDATE | DELETE | FK | Индексы |
|---|---|---|---|---|---|---|---|---|---|
| `device_sessions` | `014_device_sessions.sql` | нет в `models/` | нет | только `session_store.go:315` | только `session_store.go:150,334` | только `session_store.go:398` | только `session_store.go:422,465,480` | в миграции | в миграции |
| `device_session_skipped_keys` | `014_device_sessions.sql` | нет в `models/` | нет | только `session_store.go` | — | — | — | в миграции | в миграции |

**Все SQL-ссылки ведут в недостижимый код.** Реальных обращений к этим таблицам
нет.

Таблицы не удаляются. Документируются как **unused, зарезервированные под
серверное персистентное E2EE-состояние**.

## 11. Protocol Versioning

| Механизм | Где записывается | Где читается | Где валидируется | Диспетчер |
|---|---|---|---|---|
| `StoredPairwiseSession.version` | `pairwiseSession.ts:97` | `restorePairwiseSession` | `validateState` `:506` | нет |
| `X3DHSessionHeader.version` | `pairwiseSession.ts:193` | `acceptPairwiseSession` | `validateHeader` `:497` | нет |
| `X3DHSessionHeader.cipherSuite` | `pairwiseSession.ts:194` | `validateHeader` `:497` | `validateHeader` `:497` | нет |
| `messages.protocol_version` | клиент → `toApiPayload` → БД | `message/service.go:120` | **сервером не валидируется** | нет |
| `messages.cipher_suite` | клиент → БД | `message/service.go:120` | **сервером не валидируется** | нет |

Поведение при несовпадении: `validateHeader` и `validateState` **отвергают**
значение (`throw`), а не выбирают альтернативную реализацию.

**Диспетчер версий отсутствует.** Сравнение происходит по точному равенству.

Техническая возможность добавить v2 без изменения формата хранения:
существует. `sessionId` различает сессии, `protocol_version` и `cipher_suite`
уже являются колонками, `version` уже присутствует в персистированной сессии.
Требуется только код выбора реализации по значению поля. Реализация не
выполнялась.

## 12. Client/Server Responsibility

| Операция | Client | Go | Фактический production path |
|---|---|---|---|
| X3DH initiator | `initiatePairwiseSession` | `X3DHService.StartSession` (0 вызовов) | **только client** |
| X3DH responder | `acceptPairwiseSession` | отсутствует | **только client** |
| Root KDF | `rootRatchet` | `RootRatchet` (0 вызовов) | **только client** |
| Chain KDF | `advanceRatchet` | `deriveMessageAndNextChain` (0 вызовов) | **только client** |
| Шифрование сообщения | `crypto/index.ts:160` | `EncryptMessage` (0 вызовов) | **только client** |
| Расшифровка сообщения | `crypto/index.ts:236` | `DecryptMessage` (0 вызовов) | **только client** |
| DH ratchet | `dhRatchetTurn` | `DHRatchetTurn` (0 вызовов) | **только client** |
| Skipped keys | `nextReceiveMessageKey` | `NextRecvMessageKey` (0 вызовов) | **только client** |
| Персистентность сессии | `deviceKeyStore.ts` → OS keyring | `SessionStore` (0 вызовов) | **только client** |

### 12.1 Что видит сервер

| Объект | Видит ли сервер |
|---|---|
| Plaintext сообщения | **Нет** |
| Root key | **Нет** |
| Chain key | **Нет** |
| Message key | **Нет** |
| Content key | **Нет** — хранится только обёрнутым в `payload` |
| Private key устройства | **Нет** — комментарий миграции `013` это фиксирует |
| Шифротекст, IV, тег | Да, `messages.ciphertext`, `iv`, `tag` |
| Идентификаторы устройств и сессий | Да |
| `protocol_version`, `cipher_suite` | Да, но без проверки |

Сервер выполняет только `INSERT` и `SELECT` opaque-значений.

## 13. Dead/Unused E2EE Components

| Компонент | Тип | Основание |
|---|---|---|
| `internal/crypto/x25519.go` | test-only | 0 внешних импортов |
| `internal/crypto/protocol.go` | test-only | 0 внешних импортов |
| `internal/crypto/ratchet.go` | test-only | 0 внешних импортов |
| `internal/crypto/message.go` | test-only | 0 внешних импортов |
| `internal/crypto/multidevice.go` | test-only | 0 внешних импортов |
| `internal/crypto/session_store.go` | **dead** | 0 импортов, 0 тестов |
| таблица `device_sessions` | unused | SQL только из dead-кода |
| таблица `device_session_skipped_keys` | unused | SQL только из dead-кода |
| `encryptLegacyForAlphaCompatibility` (`crypto/index.ts:171`) | **test-only** | вызовов нет; используется только в `crypto.test.ts:136` |

Дополнительно проверено:

| Компонент | Место | Поведение |
|---|---|---|
| `canUseLegacyChannelKeys` | `crypto/index.ts:316` | возвращает `!env.PROD` |
| `assertLegacyChannelKeysAllowed` | `crypto/index.ts:320` | в production выбрасывает `ChannelSessionDistributionUnavailableError` |
| Обработка в UI | `ChatView.tsx:355`, `:394` | показывается уведомление вместо отправки |

Это подтверждает fail-closed решение: локальный одноразовый ключ канала,
который не обеспечивает защиту общего канала, запрещён в production-сборке.
Отправка завершается уведомлением, а не деградацией до небезопасного режима.

Ничего не удалено.

## 14. CRY-203 Impact

### 14.1 Что происходит после перезапуска клиента

| Аспект | Фактическое поведение |
|---|---|
| Восстановление rootKey и цепей | Работает — поля сериализуются |
| Восстановление счётчиков | Работает |
| Восстановление skipped keys | Работает |
| Восстановление `dhSendPublic`, `dhRecvPublic` | Работает |
| Восстановление `dhSendPrivate` | **Не работает** — поле не сериализуется, restore даёт нули |
| Расшифровка по существующей цепи | Работает |
| DH-шаг при новом `dh_public_key` | **Fail-closed: исключение** |

### 14.2 Локальное поведение при DH-шаге после перезапуска

`messageEnvelopes.ts:116-118` вызывает `performDHRatchetTurn`, если
`envelope.dhPublicKey` отличается от `state.dhRecvPublic`. Вызов находится
**вне** `try`-блока (`:121`), поэтому исключение fail-closed пробрасывается из
`decryptChannelMessage`.

- Для новых сообщений: `ChatView.tsx:246` находится внутри `try` (`:243`).
  Поведение обработчика требует проверки в компоненте.
- Для истории: `ChatView.tsx:441` использует `Promise.allSettled`, поэтому
  отказ отдельного сообщения не прерывает загрузку списка.

**Следствие:** после перезапуска первое сообщение, требующее DH-шага, приведёт к
отказу расшифровки вместо её выполнения. Состояние сессии при этом не портится
необратимо: fail-closed срабатывает до изменения состояния.

### 14.3 Где состояние сессии должно переживать перезапуск

Фактически переживает: `StoredDeviceKeyBundle` в OS keyring через
`store_device_key_bundle`. Альтернативных хранилищ для pairwise-сессий не найдено.

Шифрование at-rest на уровне приложения для бандла не выполняется; защита
делегирована механизму Tauri. Комментарий `deviceKeyStore.ts:83` указывает, что
production веб-сборки не должны сохранять приватный материал в localStorage.

## 15. Open Architectural Questions

1. Почему `internal/crypto` содержит 52 КБ кода, недоступного из рабочего пути?
2. Представляет ли `session_store.go` незавершённую миграцию на серверное
   E2EE-состояние или устаревшую попытку? История репозитория не анализировалась.
3. Должна ли таблица `device_sessions` использоваться, и если да, то для чего:
   серверного relay состояния, оффлайн-доставки или восстановления?
4. Должен ли сервер вообще участвовать в E2EE, учитывая, что вся схема
   спроектирована как клиентская?
5. Требуется ли `encryptLegacyForAlphaCompatibility` в production, и что он
   делает с версионированием?

## 16. Owner Decisions

1. **Должен ли Go E2EE оставаться частью архитектуры?** Установлено, что он
   недостижим из рабочего пути. Решение о судьбе кода, таблиц и тестов —
   за владельцем.
2. **Если остаётся — какой ролью?** Production path, future path или
   compatibility path. Каждая роль требует разных действий: production требует
   реализации responder, AAD и подключения к маршрутам; future требует
   определения сроков; compatibility требует доказательства совместимости.
3. **Сохранять ли серверные таблицы сессий?** Они не удаляются автоматически.
4. **Реализовывать ли Go-ответную сторону X3DH?** Отсутствует.
5. **Нужен ли диспетчер версий?** Сейчас несовместимая версия отвергается.
6. **Какая сторона отвечает за персистентность pairwise-сессии?** Фактически —
   клиент, через OS keyring.
7. **Как выполняется восстановление сессии после перезапуска?** Фактически —
   частично: цепи восстанавливаются, DH-шаг невозможен.
