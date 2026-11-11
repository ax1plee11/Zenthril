# ZFDA IDENTITY_AND_ACCESS — Идентификация и управление доступом

## Аутентификация

- **Пароли:** Argon2id хеширование с политикой сложности (`backend/auth/password.go`).
- **Access токены:** JWT HS256, TTL 15 минут, `alg:none` отвергается, HMAC-SHA256 подпись (`backend/auth/jwt.go`).
- **Refresh токены:** JWT HS256, TTL 30 дней, ротация при использовании, детекция replay (`backend/auth/service.go`).
- **Blacklist access токенов:** при выходе (logout), срок жизни до естественного истечения.
- **JWT signing key:** конфигурируется через `SetJWTSigningKey()` при запуске приложения.

## MFA / TOTP Enforcement

**Статус:** Реализован и enforced (ZFDA 1.2).

### MFA Flow при входе

1. Пользователь отправляет `POST /auth/login` с username + password
2. Сервер проверяет `IsTOTPEnabled(username)`:
   - **Если MFA включён:** Возвращает `mfa_required: true` + `challenge_id` (токены НЕ выдаются)
   - **Если MFA выключен:** Валидирует пароль → выдаёт access + refresh токены
3. Пользователь отправляет `POST /auth/mfa/verify` с challenge_id + TOTP код + username + password
4. Сервер проверяет challenge (существует, не истек, не использован) → проверяет TOTP код → потребляет challenge
5. При успешной проверке: повторная аутентификация паролем → выдача полноценных токенов

### MFA Endpoints

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/auth/mfa/totp/start` | POST | Генерация TOTP секрета (rate limited: 5/час) |
| `/auth/mfa/totp/confirm` | POST | Подтверждение TOTP и включение MFA (rate limited: 10/час) |
| `/auth/mfa/totp/disable` | POST | Отключение MFA с повторной проверкой TOTP (rate limited: 5/час) |
| `/auth/mfa/verify` | POST | Проверка MFA challenge при входе (rate limited: 10/час) |
| `/auth/mfa/status` | GET | Проверка статуса MFA для пользователя |

### Защита MFA

- TOTP по RFC 6238 (SHA1, 6 цифр, 30-секундные шаги, ±1 шаг устойчивость)
- TOTP секреты шифруются AES-256-GCM при хранении в Redis
- TOTP encryption key конфигурируется через `SetTOTPEncryptionKey()` при запуске
- Без ключа все TOTP операции возвращают ошибку (fail-secure, нет plaintext fallback)
- TOTP коды с ведущими нулями корректно обрабатываются (`%06d` форматирование)
- MFA не обходится через refresh token (клиент должен проверять `/auth/mfa/status`)

### MFA и Passkeys

WebAuthn/FIDO2 passkeys ещё не реализованы. TOTP является текущим вторым фактором.

## Защита от перебора и credential stuffing

- BruteForceProtect: 10 неудачных попыток входа → блокировка IP на 15 минут (`backend/security/guard.go`).
- IPRateLimit: 1000 запросов в секунду → блокировка на 60 секунд.
- AuthRateLimit: независимые бюджеты по scopes (register: 10/час, logout: 30/мин, refresh: 30/мин).
- MFA endpoints: отдельные rate limits (TOTPStart: 5/час, TOTPConfirm: 10/час, MFAVerify: 10/час).

## Управление сессиями

- Токены клиента хранятся в памяти процесса, не в `localStorage`.
- Refresh токены в HttpOnly куки (`SameSite=Strict`, `Secure`).
- Privacy-first startup: сохраненные сессии открываются офлайн до явного Connect.
- Лимит 10 активных устройств на пользователя с сериализацией регистрации (`FOR UPDATE`).

## Отзыв доступа

- Logout:ブラックлист access токена, отзыв refresh токена.
- LogoutAll: отзыв всех refresh токенов пользователя.
- Глобальная блокировка: аккаунт не может подключаться к WebSocket.
- Отзыв устройства: помечает устройство как отозванное, удаляет one-time prekeys.

## TOTP Recovery Codes

**Статус:** Реализованы (ZFDA 1.2).

- 10 криптографически случайных кодов (8 символов, base32) генерируются при включении MFA
- Коды показываются пользователю только один раз при создании
- Хранятся только Argon2id хеши кодов (не plaintext)
- Каждый код используется только один раз (атомарно помечается использованным)
- Ограничение: 5 попыток проверки в час
- Событие использования записывается в security audit log
- После использования всех кодов, набор деактивируется
- Генерация нового набора инвалидирует старый

Источники: NIST SP 800-63, OWASP ASVS, RFC 6238 (TOTP), FIDO2/WebAuthn. Проверено: 2026-10-30.