# Фундамент E2EE Zenthril

Языки: [Русский](E2EE_FOUNDATION.ru.md) | [English](E2EE_FOUNDATION.md) | [Українська](E2EE_FOUNDATION.uk.md)

Этот документ фиксирует production-путь к E2EE в стиле Signal. Текущая
реализация является фундаментом для device keys и X3DH key bundles. Это еще не
полный и не прошедший аудит протокол end-to-end encryption.

## Текущий этап

Реализованные backend-примитивы:

- реестр устройств для каждого пользователя;
- публичный identity key устройства;
- signed prekey и подпись signed prekey;
- загрузка и расходование one-time prekeys;
- fingerprint устройства для будущего UX проверки безопасности;
- authenticated endpoint для получения X3DH-style key bundle.

Реализованные client-примитивы:

- локальная генерация device key bundle;
- Ed25519 identity signing key для проверки signed prekey;
- X25519 signed prekey и one-time prekeys;
- best-effort регистрация устройства после login/register;
- UI управления активными устройствами и отзывом старых устройств;
- deterministic safety number для pairwise-проверки устройств;
- приватный key material остается на клиенте и не отправляется в backend
  registration requests.

## API

Authenticated routes:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices/`
- `GET /api/v1/users/{userId}/devices`
- `DELETE /api/v1/devices/{deviceId}`
- `POST /api/v1/key-bundles/claim`

`POST /api/v1/devices/register` сохраняет или ротирует device key bundle для
текущего пользователя. Клиент должен генерировать ключи локально и отправлять
только публичный key material.

`POST /api/v1/key-bundles/claim` возвращает bundle целевого устройства и
атомарно расходует не более одного one-time prekey. Endpoint намеренно использует
`POST`, потому что получение bundle изменяет состояние сервера.

`DELETE /api/v1/devices/{deviceId}` отзывает одно из устройств текущего
пользователя и удаляет его one-time prekeys, чтобы новые сессии не могли быть
установлены с этим устройством.

## Следующие шаги

1. Перевести Tauri-хранилище device keys на OS keychain или Stronghold.
2. Добавить отображение safety number и ручную verification UX.
3. Реализовать X3DH shared secret derivation с test vectors.
4. Добавить Double Ratchet session state для direct messages.
5. Добавить академическую threat model и раздел с ограничениями протокола.

## Security Notes

- Сервер никогда не должен получать private keys.
- One-time prekeys расходуются с row-level locking, чтобы избежать двойной
  выдачи одного ключа.
- Device fingerprints помогают в verification UX, но сами по себе не являются
  authentication-механизмом.
- Текущий Tauri storage adapter использует Tauri store plugin как локальный
  persistence layer. Перед production-grade заявлениями о хранении приватных
  ключей его нужно заменить на OS keychain или Stronghold.
- Safety numbers являются локальными UX-примитивами проверки. Они не заменяют
  signature verification, trust decisions или будущие per-user trust records.
- Протокол должен оставаться experimental до review и тестирования на
  воспроизводимых vectors.
