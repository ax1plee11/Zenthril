# Фундамент E2EE Zenthril

Мови: [Русский](E2EE_FOUNDATION.ru.md) | [English](E2EE_FOUNDATION.md) | [Українська](E2EE_FOUNDATION.uk.md)

Цей документ фіксує production-шлях до E2EE у стилі Signal. Поточна реалізація
є фундаментом для device keys і X3DH key bundles. Це ще не повний і не
аудитований протокол end-to-end encryption.

## Поточний етап

Реалізовані backend-примітиви:

- реєстр пристроїв для кожного користувача;
- публічний identity key пристрою;
- signed prekey і підпис signed prekey;
- завантаження та споживання one-time prekeys;
- fingerprint пристрою для майбутнього UX перевірки безпеки;
- authenticated endpoint для отримання X3DH-style key bundle.

Реалізовані client-примітиви:

- локальна генерація device key bundle;
- Ed25519 identity signing key для перевірки signed prekey;
- X25519 signed prekey і one-time prekeys;
- best-effort реєстрація пристрою після login/register;
- UI керування активними пристроями та відкликанням старих пристроїв;
- deterministic safety number для pairwise-перевірки пристроїв;
- приватний key material залишається на клієнті та не надсилається в backend
  registration requests.

## API

Authenticated routes:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices/`
- `GET /api/v1/users/{userId}/devices`
- `DELETE /api/v1/devices/{deviceId}`
- `POST /api/v1/key-bundles/claim`

`POST /api/v1/devices/register` зберігає або ротує device key bundle для
поточного користувача. Клієнт має генерувати ключі локально та надсилати тільки
публічний key material.

`POST /api/v1/key-bundles/claim` повертає bundle цільового пристрою та атомарно
споживає не більше одного one-time prekey. Endpoint навмисно використовує
`POST`, бо отримання bundle змінює стан сервера.

`DELETE /api/v1/devices/{deviceId}` відкликає один із пристроїв поточного
користувача та видаляє його one-time prekeys, щоб нові сесії не могли бути
встановлені з цим пристроєм.

## Наступні кроки

1. Перевести Tauri-сховище device keys на OS keychain або Stronghold.
2. Додати відображення safety number і ручну verification UX.
3. Реалізувати X3DH shared secret derivation з test vectors.
4. Додати Double Ratchet session state для direct messages.
5. Додати академічну threat model і розділ з обмеженнями протоколу.

## Security Notes

- Сервер ніколи не має отримувати private keys.
- One-time prekeys споживаються з row-level locking, щоб уникнути подвійної
  видачі одного ключа.
- Device fingerprints допомагають у verification UX, але самі по собі не є
  authentication-механізмом.
- Поточний Tauri storage adapter використовує Tauri store plugin як локальний
  persistence layer. Перед production-grade заявами про зберігання приватних
  ключів його потрібно замінити на OS keychain або Stronghold.
- Safety numbers є локальними UX-примітивами перевірки. Вони не замінюють
  signature verification, trust decisions або майбутні per-user trust records.
- Протокол має залишатися experimental до review і тестування на відтворюваних
  vectors.
