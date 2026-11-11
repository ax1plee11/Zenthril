# ZFDA 1.3 — Этап G: Tauri Client Security (Безопасность клиента)

## 1. Audit tauri.conf.json

### До исправления (проблемы)

```json
"csp": "default-src 'self'; connect-src 'self' http://localhost:8080 ws://localhost:8080 https: wss:; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'"
```

**Проблемы:**
1. `connect-src` содержит `https:` и `wss:` — разрешает соединение на **любой** внешний хост (exfilтрация данных)
2. `style-src` содержит `'unsafe-inline'` — уязвимость к XSS через инлайн стили
3. `script-src` не указан — по умолчанию `'self'` но лучше явно указать
4. Нет раздела `capabilities` — не явно ограничены возможности приложения

### После исправления

```json
"csp": "default-src 'self'; connect-src 'self' http://localhost:8080 ws://localhost:8080 https://api.zenthril.example.com wss://api.zenthril.example.com; img-src 'self' data: https://cdn.zenthril.example.com; style-src 'self'; script-src 'self'"
```

**Исправления:**
1. `connect-src` ограничен конкретными хостами (api.zenthril.example.com, cdn.zenthril.example.com)
2. Убран `'unsafe-inline'` из style-src
3. Явно указан `script-src 'self'`
4. `img-src` ограничен конкретным CDN

## 2. 其他 клиентские настройки безопасности

### client/.env

| Параметр | Было | Стало | Обоснование |
|----------|------|-------|-------------|
| `VITE_ALLOW_INSECURE_KEY_STORAGE` | `true` | `false` | Запрет небезопасного хранения E2EE ключей в localStorage production сборках |

### VITE_WEBRTC_RELAY_ONLY

По умолчанию `false` в dev, должен быть `true` в production (требует настройки TURN серверов).

## 3. Проверка IPC и Capabilities

### Проверенные аспекты:

| Аспект | Статус | Примечание |
|--------|--------|------------|
| Tauri capabilities | Not Verified | Нет явного раздела capabilities в tauri.conf.json |
| IPC границы | Partially | Использует стандартные Tauri API, требует аудита |
| tauri-plugin-shell | Risk | Позволяет выполнение shell команд — потенциальный вектор атаки |
| CSP | Verified | Исправлен (см. выше) |
| XSS | Partially | Убран unsafe-inline, требует проверки рендеринга пользовательского контента |
| Local storage токенов | Verified | Токены в процессе памяти, refresh через HttpOnly cookies |
| E2EE ключи | Partially | keyring для Tauri, localStorage только в dev |

## 4. Рекомендации (требуют реализации)

1. **Добавить раздел `capabilities`** в tauri.conf.json для явного ограничения возможностей
2. **Убрать `tauri-plugin-shell`** если не используется (сокращает поверхность атаки)
3. **Добавить `script-src` с nonce или hash** для строгого CSP
4. **Добавить `frame-ancestors 'none'`** для защиты от clickjacking
5. **Проверить все `window.open` вызовы** — добавить `noopener,noreferrer` (найдено в аудите: MessageItem.tsx:130)
6. **Запретить direkten рендеринг localhost/127.0.0.1 URL** как медиа (социальная инженерия)
7. **Проверить deep links** — валидация перед обработкой

### Статус: Partially Verified (CSP исправлен, остальные требования требуют реализации)
