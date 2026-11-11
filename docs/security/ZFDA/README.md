# ZFDA — Zenthril Fortress Defense Architecture

ZFDA — фундаментальный архитектурный принцип защиты приложения Zenthril. Вся архитектура строится по принципу многоуровневой, взаимосвязанной и отказоустойчивой защиты (Defense in Depth, Zero Trust, Least Privilege, Separation of Duties, Compartmentalization, Fail Secure, Assume Breach, Secure by Default, Defense Against Lateral Movement, Recovery by Design).

## Десять рубежей защиты

| Уровень | Назначение | Статус реализации | Краткое описание |
|---------|------------|-------------------|------------------|
| 1 | Perimeter Defense | Реализован частично | CORS, Origin allowlist, security headers, rate limiting, body limits |
| 2 | Identity Defense | Реализован | Argon2id, JWT HS256, refresh rotation, brute-force, **MFA/TOTP enforcement**, MFA challenge flow |
| 3 | Device Defense | Реализован частично | Регистрация устройств, Ed25519 подпись, fingerprint, лимит 10, отзыв |
| 4 | Application Defense | Реализован частично | Параметризованные SQL, валидация входа, WebSocket анти-флудинг, RBAC v2 |
| 5 | Network Defense | Реализован частично | TLS, WebSocket Origin, изоляция БД/Redis в Docker сети |
| 6 | Server Defense | Реализован частично | Docker limits, health checki, graceful shutdown, fail-closed конфиг |
| 7 | Cryptographic Defense | Исправлен | X25519, HKDF-SHA256, AES-256-GCM, E2EE, кросс-реализационные тестовые векторы (F-09 исправлен) |
| 8 | Data Defense | Реализован частично | Параметризованные SQL, Redis изоляция, security log без секретов, **TOTP секреты зашифрованы (AES-256-GCM)** |
| 9 | Detection & Intelligence | Реализован | Autonomous Defense: ThreatDetector, RiskEngine, PolicyEngine, ResponseEngine, RecoveryEngine — подключены к реальным событиям |
| 10 | Recovery & Resilience | Реализован частично | Graceful shutdown, readiness probes, RecoveryEngine, **TOTP recovery codes (one-time, Argon2id)** |

## Навигация

- [ARCHITECTURE.md](ARCHITECTURE.md) — целевая архитектура защиты
- [THREAT_MODEL.md](THREAT_MODEL.md) — модель угроз
- [ASSET_INVENTORY.md](ASSET_INVENTORY.md) — инвентарь активов
- [ATTACK_SURFACE.md](ATTACK_SURFACE.md) — поверхность атак
- [TRUST_BOUNDARIES.md](TRUST_BOUNDARIES.md) — границы доверия
- [SECURITY_POLICY.md](SECURITY_POLICY.md) — политика безопасности
- [IDENTITY_AND_ACCESS.md](IDENTITY_AND_ACCESS.md) — идентификация и доступ
- [DEVICE_SECURITY.md](DEVICE_SECURITY.md) — безопасность устройств
- [APPLICATION_SECURITY.md](APPLICATION_SECURITY.md) — безопасность приложения
- [NETWORK_SECURITY.md](NETWORK_SECURITY.md) — сетевая безопасность
- [INFRASTRUCTURE_SECURITY.md](INFRASTRUCTURE_SECURITY.md) — безопасность инфраструктуры
- [CRYPTOGRAPHY.md](CRYPTOGRAPHY.md) — криптографическая защита
- [DATA_SECURITY.md](DATA_SECURITY.md) — защита данных
- [AUTONOMOUS_DEFENSE.md](AUTONOMOUS_DEFENSE.md) — автономная защита
- [INCIDENT_RESPONSE.md](INCIDENT_RESPONSE.md) — реагирование на инциденты
- [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) — аварийное восстановление
- [SECURITY_TESTING.md](SECURITY_TESTING.md) — тестирование безопасности
- [RISK_REGISTER.md](RISK_REGISTER.md) — реестр рисков
- [ACCEPTANCE_CRITERIA.md](ACCEPTANCE_CRITERIA.md) — критерии приемки
- [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) — дорожная карта внедрения
- [ZFDA_1.1_SECURITY_VERIFICATION_REPORT.md](ZFDA_1.1_SECURITY_VERIFICATION_REPORT.md) — отчёт проверки ZFDA 1.1
- [ZFDA_1.2_SECURITY_COMPLETION_REPORT.md](ZFDA_1.2_SECURITY_COMPLETION_REPORT.md) — отчёт завершения ZFDA 1.2

## Стандарты и источники

- NIST Cybersecurity Framework 2.0 (проверено: 2026-10-30)
- NIST Zero Trust Architecture (проверено: 2026-10-30)
- NIST SP 800-53 (проверено: 2026-10-30)
- NIST SP 800-61 (проверено: 2026-10-30)
- OWASP ASVS (проверено: 2026-10-30)
- OWASP API Security Top 10 (проверено: 2026-10-30)
- CIS Controls (проверено: 2026-10-30)
- ISO/IEC 27001 и 27002 (проверено: 2026-10-30)
- RFC 8446 (TLS 1.3)
- RFC 7748 (X25519/X448)
- RFC 5869 (HKDF)
- FIDO2 и WebAuthn

Директория `docs/security/ZFDA/` соответствует фактическому состоянию кода на момент внедрения. Не описываются планируемые функции как уже реализованные.

## Версии ZFDA

- **ZFDA 1.0** — Базовая реализация: криптография (X3DH, Double Ratchet), MFA заглушки, Autonomous Defense классы
- **ZFDA 1.1** — Проверка безопасности: исправлена несовместимость крипто клиент/Go (F-09), реализован MFA/TOTP, добавлены тесты
- **ZFDA 1.2** — Завершение: MFA enforcement при входе, безопасная инициализация TOTP ключа, интеграция Autonomous Defense к реальным событиям, TOTP recovery codes, интеграционные тесты

---
