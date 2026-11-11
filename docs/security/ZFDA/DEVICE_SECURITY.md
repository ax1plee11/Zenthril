# ZFDA DEVICE_SECURITY — Безопасность устройств

## Идентификация устройств

- Устройство идентифицируется UUID (генерируется сервером или валидируется как UUID).
- Fingerprint устройства: SHA-256(userID + deviceID + identity_signing_public_key + identity_dh_public_key) (`backend/device/service.go`).
- Оба долгосрочных публичных ключа включаются в отпечаток — замена любого ключа обнаруживается.

## Регистрация и отзыв устройств

- Регистрация требует: identity_public_key (32 байта), identity_dh_public_key (X25519), signed_pre_key, подпись signed prekey (Ed25519), опциональные one-time prekeys.
- Подпись signed prekey проверяется через `ed25519.Verify` перед публикацией.
- Low-order X25519 точки отвергаются при регистрации.
- Лимит 10 активных устройств на пользователя, сериализовано через `SELECT ... FOR UPDATE`.
- Отзыв устройства: `trust_state = 'revoked'`, `revoked_at = NOW()`, удаление one-time prekeys.

## Контроль активных сессий

- Каждое устройство имеет собственную идентификацию.
- Список устройств владельца: полная информация включая ключи.
- Публичный список устройств другого пользователя: только device_id, fingerprint, trust_state, требуется общая гильдия.
- Claim key bundle: потребление одного one-time prekey, `FOR UPDATE SKIP LOCKED`, проверка общей гильдии.

## Обнаружение подозрительных изменений

- Trust state устройства отслеживается (unverified, revoked).
- Изменение ключей устройства сбрасывает trust_state в 'unverified'.

## Безопасное локальное хранение секретов

- Клиент: OS keychain через Rust `keyring` (Windows Credential Manager, macOS Keychain, Linux Secret Service).
- Приватные E2EE ключи не хранятся в `localStorage` (кроме явного insecure режима для тестирования).
- Проверка userId бандлов ключей при загрузке.

Проверено: 2026-10-30.