# CRY-101 Impact & Migration Analysis

**Статус:** исследовательский этап CRY-101.1. Производственный код не изменён.
**Дата:** 01.10.2026
**Ревизия:** `c027376`
**Предшествующий документ:** `docs/security/CRY-101_PROTOCOL_DECISION.md`

> Документ не выбирает нормативную схему. Раздел 17 содержит вопросы владельцу.

## 1. Scope

Цель этапа — установить фактические последствия Option A и Option B на основании
существующего кода и схемы хранения данных, а не на основании несовместимости
KDF как таковой.

Проверено чтением кода:

| Область | Источник |
|---|---|
| Серверная E2EE-реализация | `backend/internal/crypto/` |
| Клиентская E2EE-реализация | `client/src/crypto/`, `client/src/features/e2ee/` |
| Схема БД | `backend/migrations/001`…`015` |
| Живой путь сообщений | `backend/message/service.go`, `backend/message/handler.go` |
| Клиентское хранилище | `client/src/features/e2ee/deviceKeyStore.ts` |

## 2. Current Protocol State

| Параметр | Значение | Место |
|---|---|---|
| Client protocol version | `1` | `client/src/features/e2ee/pairwiseSession.ts:9` |
| Client cipher suite | `"X3DH-HKDF-SHA256-DR-v1"` | `pairwiseSession.ts:10` |
| Go ratchet constants | `zenthril-ratchet-v1:init:`, `:root`, `:chain` | `backend/internal/crypto/ratchet.go` |
| Go X3DH info | `"Zenthril X3DH v1"` | `backend/internal/crypto/x25519.go:82` |
| Message envelope version | `protocol_version INTEGER NOT NULL DEFAULT 1` | `backend/migrations/008_message_crypto_envelope.sql:3` |

Несовместимость детально описана в `CRY-101_PROTOCOL_DECISION.md`. Здесь
устанавливается, какая часть этой несовместимости имеет реальные последствия.

## 3. Session Storage Architecture

### 3.1 Что хранится на сервере — ФАКТ

| Объект | Таблица | Кто пишет | Кто читает |
|---|---|---|---|
| Состояние E2EE-сессии | `device_sessions` | **никто** | **никто** |
| Skipped message keys | `device_session_skipped_keys` | **никто** | **никто** |
| Сессии refresh | Redis `refresh:` | `auth/service.go` | `auth/service.go` |
| Билеты WebSocket | Redis `ws:ticket:` | `auth/service.go:344` | `auth/service.go:408` |
| Ключи устройств | таблица `devices` | `device/service.go` | `device/service.go`, `key-bundles/claim` |

### 3.2 Установлено: серверная E2EE-реализация не входит в рабочий путь

Поиск по всему backend обнаружил **нулевое число вызовов**:

```
NewSessionStore          — вызовов нет
NewX3DHService           — вызовов нет
crypto.SessionState      — использований нет
SaveSession              — вызовов нет
LoadSession              — вызовов нет
NextSendMessageKey       — вызовов нет
NextRecvMessageKey       — вызовов нет
DHRatchetTurn            — вызовов нет
```

Таблицы `device_sessions` и `device_session_skipped_keys`, созданные миграцией
`014_device_sessions.sql`, **не читаются и не записываются** ни одним запросом
в `backend/`.

**Следствие.** Серверных состояний E2EE-сессий в эксплуатации не существует.
Пакет `backend/internal/crypto` является изолированным кодом без потребителей.

### 3.3 Что хранится на клиенте

`StoredDeviceKeyBundle`, персистируемая через `deviceKeyStore.ts`:

| Хранилище | Механизм | Статус |
|---|---|---|
| Tauri (десктоп) | `invoke("store_device_key_bundle")` → OS keyring | основной путь |
| Браузер, production | `localStorageAdapter` помечен `"insecure-localstorage"` | заблокирован комментарием на `deviceKeyStore.ts:83` |
| Браузер, development | тот же адаптер | допускается флагом окружения |

### 3.4 Поля персистированной сессии

`serializePairwiseSession` (`pairwiseSession.ts:90-111`) записывает:

`version`, `sessionId`, `peerUserId`, `peerDeviceId`, `rootKey`, `sendChainKey`,
`receiveChainKey`, `sendCounter`, `receiveCounter`, `dhSendPublic`,
`dhRecvPublic`, `previousCounter`, `skippedMessageKeys`.

**`dhSendPrivate` не сериализуется.** Подробно — раздел 15.

## 4. Message Storage Architecture

### 4.1 `messages`

| Колонка | Тип | Источник | Содержимое |
|---|---|---|---|
| `ciphertext` | TEXT | клиент | base64 шифротекста |
| `iv` | TEXT | клиент | base64 вектора инициализации |
| `key_id` | TEXT | клиент | идентификатор ключа, максимум 128 символов (`handler.go:21`) |
| `tag` | TEXT | клиент | base64 тега AES-GCM |
| `protocol_version` | INTEGER NOT NULL DEFAULT 1 | клиент | версия конверта, миграция `008` |
| `sender_device_id` | TEXT | клиент | миграция `011` |
| `session_id` | TEXT | клиент | миграция `011` |
| `client_message_id` | TEXT | клиент | миграция `011` |
| `cipher_suite` | TEXT | клиент | миграция `011` |

### 4.2 `message_recipient_envelopes`

| Колонка | Тип | Содержимое |
|---|---|---|
| `message_id`, `recipient_user_id`, `recipient_device_id` | UUID | адресация |
| `session_id` | TEXT | идентификатор сессии |
| `ratchet_counter` | BIGINT NOT NULL, CHECK >= 0 | позиция в цепочке |
| `bootstrap_header` | JSONB | заголовок X3DH, включая `version` и `cipherSuite` |
| `payload` | JSONB NOT NULL | opaque, обёрнутый content key |
| `dh_public_key` | TEXT | для DH-шага, миграция `015` |

Комментарий миграции `013` фиксирует: «no private device keys or plaintext
content keys are stored here».

### 4.3 Ответ на вопросы о версии сообщения

| Вопрос | Ответ |
|---|---|
| Хранится ли ciphertext в БД | Да, `messages.ciphertext` |
| Хранится ли версия протокола рядом | Да, `messages.protocol_version` и `messages.cipher_suite` |
| Можно ли определить версию шифрования сообщения | Да, по `protocol_version` и `cipher_suite` |
| Может ли новый клиент расшифровать старое сообщение | Только если сохранил старую сессию; сама версия известна |
| Требуется ли повторное шифрование | Не требуется для сохранения читаемости: `protocol_version` позволяет не трогать старые строки |

## 5. Client-side State

| Состояние | Локация | Версионировано |
|---|---|---|
| `StoredDeviceKeyBundle` | `deviceKeyStore.ts` → OS keyring | `pairwiseSessions[].version` |
| `PairwiseSessionState` в памяти | `pairwiseSession.ts` | `version: 1`, проверяется в `validateState` (`:506`) |
| Skipped message keys | внутри сессии, сериализуются | вместе с сессией |
| `dhSendPrivate` | **не сериализуется** | — |

## 6. Server-side State

| Утверждение | Статус |
|---|---|
| Сервер хранит состояние E2EE-сессии | **Нет** — таблицы `device_sessions`, `device_session_skipped_keys` пусты и не используются |
| Сервер использует root/chain keys | **Нет** — ни один вызов `internal/crypto` из сервисов |
| Сервер хранит только метаданные конверта | Да, и это всё: сессия целиком клиентская |
| Сервер может расшифровать сообщение | **Нет** — content key обёрнут per-device, сервер хранит только opaque `payload` |
| Session cache в Redis | **Не найдено** |
| Redis state по E2EE | **Не найдено** — `refresh:`, `security:`, `ws:` |

### 6.1 Запрошенная цепочка

```
DATABASE / REDIS
    ↓  device_sessions, device_session_skipped_keys — ПУСТЫ, не запрашиваются
LOAD SESSION
    ↓  SessionStore.LoadSession — вызовов нет
DESERIALIZE
    ↓  не выполняется в рабочем пути
USE IN RATCHET
    ↓  NextSendMessageKey / DHRatchetTurn — вызовов нет
MESSAGE DECRYPTION
    ↓  сервер не участвует
```

**Точка возникновения несовместимости на сервере не существует**, поскольку
серверный конвейер не выполняется. Несовместимость проявляется исключительно
между двумя путями внутри клиента (создание сессии против чтения) и между
клиентом и мёртвым Go-кодом.

## 7. Existing Session Lifecycle

| Стадия | Реализация | Место |
|---|---|---|
| Bootstrap | `initiatePairwiseSession` / `acceptPairwiseSession` | `pairwiseSession.ts:181, 229` |
| Выдача чужого bundle | `POST /api/v1/key-bundles/claim` | `main.go:337` |
| Отправка | `nextSendMessageKey` | `pairwiseSession.ts` |
| Приём | `acceptPairwiseSession` при наличии `bootstrap_header`, иначе `loadPairwiseSession` | `messageEnvelopes.ts:107-109` |
| DH-шаг | `performDHRatchetTurn` при расхождении `dh_public_key` | `messageEnvelopes.ts` |
| Сохранение | `storeDeviceKeyBundle(savePairwiseSession(...))` | `messageEnvelopes.ts:81` |
| Восстановление | `restorePairwiseSession` | `pairwiseSession.ts:113` |

## 8. Existing Message Lifecycle

1. Клиент получает или создаёт сессию.
2. `encryptChannelMessage` шифрует содержимое, оборачивает content key для
   каждого получателя, формирует `recipientEnvelopes`.
3. `POST` в `message/service.go:117-128` сохраняет строку `messages` со всеми
   метаданными версии и один `message_recipient_envelopes` на устройство.
4. Получатель читает историю, находит свой конверт по `recipientDeviceId`,
   при наличии `bootstrap_header` выполняет `acceptPairwiseSession`, иначе
   `loadPairwiseSession`, затем `nextReceiveMessageKey(state, ratchetCounter)`.

## 9. Upgrade Scenario Matrix

Термины: **OLD** — клиент с текущей клиентской схемой; **NEW** — клиент с
изменённой схемой.

| Сценарий | Новая сессия | Существующая сессия | Отправка | Приём и расшифровка | DH-шаг |
|---|---|---|---|---|---|
| **OLD ↔ OLD** | Работает | Работает | Работает | Работает | Работает |
| **OLD ↔ NEW** | Сессия, созданная OLD, читается NEW только если сохранена её сессия | Ключи не совпадут, если схема изменилась | Симметрично | **Отказ** при расхождении KDF | **Отказ** |
| **NEW ↔ OLD** | Симметрично предыдущей строке | — | — | **Отказ** | **Отказ** |
| **NEW ↔ NEW** | Работает | Работает | Работает | Работает | Работает |

Дополнительно по всем сценариям:

| Аспект | OLD ↔ OLD | Смешанные версии | NEW ↔ NEW |
|---|---|---|---|
| Персистентность | Работает | Сессия NEW может перезаписать запись OLD | Работает |
| Восстановление после перезапуска | Работает, но DH-шаг невозможен (CRY-203) | То же | То же |
| История сообщений | Читается | Старые строки читаются только при наличии старой сессии | Старые строки читаются только при наличии старой сессии |

Механизма, обеспечивающего совместимость версий при рукопожатии, **не найдено**.
Заголовок `bootstrap_header` содержит `version` и `cipherSuite`, и клиент
проверяет `header.version !== PAIRWISE_SESSION_PROTOCOL_VERSION` (`:497`), то есть
**несовместимая версия отвергается, а не обслуживается**.

## 10. Option A Impact — Go normative

| Область | Последствие |
|---|---|
| X3DH initiator | Клиент переписывается под Go: salt 32 нулевых байта, info `"Zenthril X3DH v1"`, выход 32 байта |
| X3DH responder | **Требуется реализовать в Go.** `X3DHService` имеет только `StartSession` (`protocol.go:82`) |
| Root KDF | Клиент переписывается на двухстадийную схему с `sessionInfo` вида `uid:dev->uid:dev` |
| Ratchet | Клиент переписывается на 4 изменения: chain salt, chain info, порядок сегментов, info начального DH-шага |
| AAD | **Требуется найти или создать формирование AAD в Go.** В Go не найдено |
| Existing sessions | **Все клиентские сессии становятся нерасшифровываемыми.** Пересоздание через X3DH |
| Existing messages | Строки сохраняются, но конверты не расшифровываются без старой сессии |
| Server state | Изменений не требуется: состояний нет |
| Client state | Полная перезапись сессий |
| Migration | Требуется принудительный re-bootstrap всех пар устройств |
| Backward compatibility | **Отсутствует** — механизма поддержки двух версий нет |
| Required code changes | `pairwiseSession.ts` (4 участка), `crypto/ratchet.ts`, отсутствующий responder в Go, формирование AAD |
| Protocol versioning | Версия должна быть поднята до 2; поле `protocol_version` в БД уже существует |
| Risk areas | Отсутствие работающей эталонной реализации: Go-код не покрыт production-путём и не проверен векторами |

## 11. Option B Impact — Client normative

| Область | Последствие |
|---|---|
| X3DH initiator | Клиент не изменяется |
| X3DH responder | Без изменений |
| Root KDF | Без изменений |
| Ratchet | Без изменений |
| AAD | **Требуется реализовать формирование AAD в Go**, соответствующее `messageAAD.ts` |
| Existing sessions | Не затрагиваются |
| Existing messages | Не затрагиваются |
| Server state | Изменений не требуется: состояний нет |
| Client state | Не затрагивается |
| Migration | Не требуется для данных |
| Backward compatibility | Сохраняется trivially |
| Required code changes | `x25519.go`, `ratchet.go` (4 места), `protocol.go` (воспроизведение canonical header), формирование AAD |
| Protocol versioning | Версия остаётся 1 для клиента; при подключении Go к рабочему пути потребуется версия |
| Risk areas | Go остаётся неиспользуемым кодом; правки Go не влияют на эксплуатацию и не будут проверены векторами, пока не подключены к пути |

## 12. Existing Data Impact

### A. Данные, гарантированно не затрагиваемые

| Объект | Файл / таблица | Причина |
|---|---|---|
| `users`, `channels`, `guilds`, роли | миграции `001`, `010` | не содержат криптографии |
| `devices`, `device_one_time_prekeys` | миграция `006` | публичные ключи; схема выдачи не меняется |
| Refresh-сессии | Redis `refresh:` | не E2EE |
| WS-билеты | Redis `ws:ticket:` | не E2EE |
| `security_log`, `admin_audit_log` | миграции `001`, `004` | не E2EE |
| `federation_nodes`, `federation_messages` | миграция `009` | не содержат клиентских E2EE-состояний |
| `global_bans`, `banned_guilds`, `user_ip_log` | миграции `002`, `005` | не E2EE |
| Аватары | клиентский `ProfileModal` | локально, не связаны с протоколом |

### B. Данные, потенциально затрагиваемые

| Объект | Файл / таблица | Комментарий |
|---|---|---|
| Клиентские сессии | `deviceKeyStore.ts` → `pairwiseSessions` | содержат `rootKey` и цепи; версия 1 |
| `messages.protocol_version` | миграция `008` | значения клиентские, не проверяются сервером |
| `messages.cipher_suite` | миграция `011` | аналогично |
| `message_recipient_envelopes.bootstrap_header` | миграция `013` | JSONB с `version` и `cipherSuite` |

### C. Данные, которые могут стать нерасшифровываемыми

| Объект | Условие |
|---|---|
| Шифротекст `messages` | при изменении корневого ключа сессии и отсутствии сохранённой старой сессии |
| `message_recipient_envelopes.payload` | содержит обёрнутый content key; без старой сессии не unwrap |
| `message_recipient_envelopes.bootstrap_header` | при отказе нового клиента от `version: 1` (`:497`) |
| Исторические сообщения в UI клиента | если клиент удалил старые сессии |

### D. Данные, для которых возможна миграция

| Действие | Обоснование |
|---|---|
| Перечислить сообщения по версии | `SELECT ... WHERE protocol_version = 1` |
| Перечислить сообщения по набору шифров | `cipher_suite` |
| Перешифровать существующие сообщения новой версией | Требует plaintext, то есть расшифровки на клиенте |
| Пометить старые строки как архивные | Колонка `protocol_version` уже существует |
| Инвалидировать клиентские сессии | Удаление записей из `StoredDeviceKeyBundle` |

### E. Данные, для которых миграция невозможна без участия клиента

| Объект | Причина |
|---|---|
| `messages.ciphertext` | Сервер не имеет content key; plaintext недоступен серверу принципиально |
| `message_recipient_envelopes.payload` | Обёрнут per-device; сервер хранит только opaque значение |
| Любая повторная криптографическая операция над историей | Требует ключевого материала, существующего только на устройствах |

**Ответ на вопрос о повторном шифровании без plaintext:** невозможно.
**Ответ на вопрос о роли сервера в такой миграции:** сервер может только
идентифицировать строки по `protocol_version`; сама миграция принципиально
остаётся клиентской.

## 13. Migration Possibilities

| Сценарий миграции | Осуществимость | Условие |
|---|---|---|
| Перевод только клиента | Реализуемо | Смена KDF в `pairwiseSession.ts` и `crypto/ratchet.ts` |
| Перевод клиента с сохранением старых сессий | Реализуемо только при поддержке двух версий | Механизма нет |
| Перешифровка истории | Реализуемо | Только клиентом, с последовательным расшифрованием |
| Одновременная работа v1 и v2 | Не реализовано | Требует диспетчера версий в `initiatePairwiseSession`, `acceptPairwiseSession` и `decryptChannelMessage` |
| Полная инвалидация | Реализуемо | Удаление клиентских сессий, потеря читаемости истории |

## 14. Versioned Protocol Possibility

| Вопрос | Ответ |
|---|---|
| Где хранить версию протокола сессии | Уже есть: `StoredPairwiseSession.version` (`pairwiseSession.ts:97`) |
| Где хранить версию протокола сообщения | Уже есть: `messages.protocol_version` (миграция `008`) |
| Где передавать версию при рукопожатии | Уже есть: `X3DHSessionHeader.version` и `cipherSuite` (`:193-194`) |
| Где проверяется версия | `validateHeader` (`:497`) и `validateState` (`:506`) — обе **отвергают** несовпадение |
| Можно ли поддерживать v1 и v2 одновременно | Сейчас нет. Требуется диспетчер |
| Можно ли создать v2-сессию не разрушая v1 | Технически возможно: `sessionId` различает сессии |
| Можно ли отправлять сообщения разных версий | Возможно на уровне строк БД: версия хранится в каждой строке |
| Изменения схемы БД | **Не требуются** — `protocol_version` и `cipher_suite` уже существуют |
| Изменения клиентской сериализации | Требуется добавить `version` в `StoredPairwiseSession` — уже присутствует |
| Изменения конверта сообщения | Требуется только заполнять `protocol_version` новым значением |
| Требуется ли миграция данных | Нет: старая версия остаётся в своей строке |

**Существенный вывод:** инфраструктура версионирования в проекте **уже
присутствует** — на уровне сессии, на уровне сообщения и на уровне рукопожатия.
Отсутствует только диспетчер версий, который выбирал бы реализацию по полю
`version`, вместо того чтобы отвергать несовпадение.

## 15. CRY-203 Current State

| Аспект | Факт | Место |
|---|---|---|
| Где создаётся `dhSendPrivate` | `initializeDHRatchet` получает пару от вызывающего | `pairwiseSession.ts:406` |
| Инициатор | `x25519.keygen()` | `pairwiseSession.ts:190` |
| Получатель | secret key signed prekey | `pairwiseSession.ts:265` |
| Где используется | `dhRatchetTurn` → `x25519.getSharedSecret` | `client/src/crypto/ratchet.ts:54, 60` |
| Сериализуется ли | **Нет** — в `serializePairwiseSession` поля нет | `pairwiseSession.ts:96-110` |
| Почему restore даёт нули | `restorePairwiseSession` устанавливает `new Uint8Array(32)` | `pairwiseSession.ts:113` |
| Где fail-closed | `performDHRatchetTurn` → `hasUsableDHPrivateKey` → `throw` | `pairwiseSession.ts` |
| Уже исправлено | `initializeDHRatchet` копирует ключевой материал, а не сохраняет ссылку | коммит `50d9a0b` |
| Предложенные варианты | A, B, C, D — зафиксированы в `docs/security/OPEN_TASKS.md` | — |
| Выбранный вариант | **Нет** | — |

Механизма rekey, session reset или пересоздания сессии **не найдено в коде**
(проверка: `deletePairwiseSession`, `resetSession`, `rekey`, `rotateSession`,
`clearPairwise`, `forgetSession`, `removeSession` — отсутствуют; совпадения
объясняются подстрокой `PreKey` в `oneTimePreKey`).

Единственный способ получить новую сессию — удаление `StoredDeviceKeyBundle`
целиком или перерегистрация устройства.

## 16. Unknowns / Not Proven

| Пункт | Статус |
|---|---|
| Фактическая длина сессий в данных | Не измерено: нет доступа к production БД |
| Наличие сохранённых v1-сессий у реальных пользователей | Не проверено |
| Фактическое использование `protocol_version` ≠ 1 | Не проверено: значения задаются клиентом и сервером не валидируются |
| Поведение клиента при `decryptChannelMessage === null` | Возвращает `null`; поведение UI при отказе не проверено |
| Кто устанавливает `protocol_version` на клиенте | Не проверено до конца: требуется трассировка `EncryptedPayload` |
| Работоспособность `restorePairwiseSession` для реальных бандлов | Не проверено на данных пользователей |
| Совместимость с Signal | **Не заявляется.** Векторов Signal нет |

## 17. Owner Decisions Required

1. **Нормативная схема.** Option A или Option B. Установлено, что Option A
   затрагивает рабочий клиентский путь, а Option B — нерабочий серверный код;
   выбор между ними не является технически нейтральным и определяет объём
   и риск работ.
2. **Политика версий.** Оставить только одну версию с полной инвалидацией либо
   реализовать диспетчер версий для параллельной работы v1 и v2.
3. **Судьба существующей истории.** Терпеть нечитаемость, выполнить клиентскую
   перешифровку или удалить.
4. **CRY-203.** Стратегия хранения приватного DH-ключа: A, B, C или иное.
5. **Режим отказа клиента.** Текущий код отвергает несовместимую версию.
   Требуется ли информирование пользователя вместо молчаливого отказа.
6. **Валидация версии на сервере.** Допустимо ли принимать от клиента
   произвольное значение `protocol_version` без проверки.
7. **Включение Go в рабочий путь.** Требуется ли это вообще, учитывая, что
   серверная E2EE-реализация сейчас не используется. Вопрос не связан с
   выбором схемы, но определяет, нужен ли Option B вообще.

## 18. Conclusion

1. **Серверных состояний E2EE-сессий не существует.** Пакет
   `backend/internal/crypto` не имеет ни одного вызова из сервисов; таблицы
   `device_sessions` и `device_session_skipped_keys` не используются. Любая
   формулировка о «серверных состояниях сессий, которые перестанут читаться»
   не подтверждается кодом.
2. **Версионирование протокола уже реализовано** на трёх уровнях: сессия,
   сообщение, рукопожатие. Отсутствует не схема версий, а диспетчер, который
   обслуживал бы более одной версии одновременно.
3. **Повторное шифрование истории принципиально невозможно на сервере.** Content
   key обёрнут per-device, сервер хранит только opaque значение. Миграция
   истории — исключительно клиентская операция.
4. **Асимметрия вариантов установлена фактически.** Option A затрагивает
   работающий клиентский путь и требует создания отсутствующей в Go ответной
   стороны X3DH. Option B затрагивает код, не входящий в рабочий путь, и не
   влияет на эксплуатацию. Документ не определяет, какой вариант выбрать.
5. **Ни один вариант не может быть реализован без утверждённых cross-side
   test vectors.** До их создания любая реализация будет непроверяемой.
