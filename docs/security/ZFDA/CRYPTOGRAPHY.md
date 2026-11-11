# ZFDA CRYPTOGRAPHY — Криптографическая защита

## Сохранение настоящего E2EE

- Сервер не получает приватные ключи пользователей.
- Сервер не может расшифровывать пользовательские сообщения.
- Шифрование на стороне клиента: X25519 key agreement, HKDF-SHA256 derivation, AES-256-GCM encryption.

## Ключевые алгоритмы (проверенные стандарты)

| Алгоритм | Стандарт | Назначение |
|----------|----------|------------|
| X25519 | RFC 7748 | Key agreement |
| HKDF-SHA256 | RFC 5869 | Key derivation |
| AES-256-GCM | NIST | Symmetric encryption |
| Ed25519 | RFC 8032 | Digital signatures |

## Создание и хранение ключей

- Клиент: приватные ключи в OS keychain (Windows Credential Manager / macOS Keychain / Linux Secret Service).
- Сервер: публичные ключи устройств в PostgreSQL (подписанные prekeys).
- Собственные криптографические алгоритмы запрещены.

## Ротация и отзыв ключей

- Устройства могут быть отозваны (отзыв ключевого бандла).
- One-time prekeys потребляются одноразово.
- Нет автоматической ротации долгоживущих ключей.
- TOTP encryption key: ротация через `SetTOTPEncryptionKey()`; старые зашифрованные секреты становятся нерасшифруемыми (fail-secure).

## Защита криптографических операций

- Криптографические операции изолированы на стороне клиента.
- Серверная бизнес-логика не обрабатывает приватные ключи.
- Backend валидирует формат ключей и подписи при регистрации устройств.
- JWT токены подписываются HMAC-SHA256 (`SetJWTSigningKey` при запуске).

## Проверка криптографических протоколов

- X3DH реализована на клиенте и сервере (унифицировано).
- Double Ratchet реализован (DH ratchet, symmetric chain ratchet, skipped message keys, session healing).
- Group Sender Keys foundation (до 10 устройств, fail-closed).
- **Кросс-реализационные тестовые векторы добавлены** (ZFDA 1.1, исправлен дефект F-09).
- Нет независимого криптографического аудита (требуется для production).

## Кросс-реализационная совместимость (Go ↔ TypeScript)

После унификации в ZFDA 1.1:

| Компонент | Salt | Info | Layout |
|-----------|------|------|--------|
| Chain ratchet | "zenthril-ratchet-v1" | "chain" | newChainKey[0:32], messageKey[32:64], nonce[64:76] |
| Initial state | "zenthril.e2ee.x3dh.root.v1" | "zenthril.e2ee.x3dh.session.v1:"+info | root[0:32], chain1[32:64], chain2[64:96] |
| DH ratchet | "zenthril-ratchet-v1:init-dh" | rootKey | newRootKey[0:32], newChainKey[32:64] |

## Защита от повторного воспроизведения

- WebSocket тикеты: одноразовые (GETDEL).
- Refresh токены: детекция replay с отзывом всех токенов.
- One-time prekeys: потребление одноразово.
- Нет глобального nonce/sequence для сообщений.

## Дефект F-09 — ИСПРАВЛЕН

Клиент и Go backend теперь выпускают совместимый криптографический материал:
- Chain ratchet: унифицированы salt, info, раскладка сегментов.
- Инициализация сессии: унифицированы IKM, salt, info.
- Добавлены 4 кросс-реализационных тестовых вектора в `backend/internal/crypto/test_vectors_test.go`.

Проверено: 2026-10-30.