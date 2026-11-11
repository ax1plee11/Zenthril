# ZFDA APPLICATION_SECURITY — Безопасность приложения

## Backend安全

- Router: Chi v5 с middleware.
- Security headers: CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer Policy, Permissions Policy (`backend/middleware/security.go`).
- CORS: строгий allowlist, без wildcard, валидация origin (`backend/middleware/cors.go`).
- Ограничение размера тела запроса: 1 МБ (`main.go`).
- Блокировка debug endpointов в production.
- Readiness/liveness проверки с скрытием деталей зависимостей в production.
- Аутентификация operational и metrics endpointов через bearer токен в production.

## API безопасность

- Все критические операции защищены middleware аутентификации (`authSvc.Middleware`).
- Параметризованные SQL запросы (pgx) — защита от инъекций.
- Валидация входных данных на всех endpointах.
- RBAC v2: битовой OR разрешений, иерархия ролей, защита системных ролей.
- Admin "god role" только через конфигурацию `ADMIN_USER_IDS`.

## WebSocket безопасность

- Strict Origin валидация (fail-closed).
- Одноразовые тикеты вместо долгоживущих JWT в query string.
- Лимиты сообщений: 120/мин на соединение, 300/мин на пользователя.
- Порог искаженных сообщений: 5 → закрытие соединения.
- Лимиты по типам сообщений (voice.signal: 30, voice.ice: 60, invite.send: 10, typing: 20).
- Проверка доступа к каналу перед подпиской и релеем сигналов.
- Инвайты требуют авторизации через `InviteAuthorizer` (не открытый релей).

## Защита от уязвимостей

| Угроза | Защита | Статус |
|--------|--------|--------|
| XSS | CSP, keine хранение ключей в localStorage | Реализовано частично |
| CSRF | Stateless JWT, нет cookie CSRF токена | Требуется реализация |
| SSRF | Нет явной защиты | Требуется реализация |
| IDOR | Проверка доступа к ресурсам на сервере | Реализовано частично |
| Injection | Параметризованные запросы pgx | Реализовано |
| Replay | WebSocket тикеты GETDEL, refresh токен replay detection | Реализовано |

## Обработка ошибок

- Безопасная обработка ошибок без утечки внутренней информации.
- Production readiness скрывает детали зависимостей от неаутентифицированных probes.
- Логи не содержат секретов или содержимого E2EE.

Проверено: 2026-10-30.