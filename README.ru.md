# Zenthril

**[English](README.md)** | **[Русский](README.ru.md)** | **[Українська](README.uk.md)**

---

Децентрализованный мессенджер с федеративной архитектурой и сквозным шифрованием.

## Технологический стек

- **Backend**: Go + PostgreSQL + Redis
- **Client**: Tauri + Vite + React/TypeScript
- **Шифрование**: X25519 + AES-256-GCM (E2EE)
- **Аутентификация**: JWT + Argon2id
- **UI**: Tailwind CSS + shadcn/ui + Glass Minimal Design
- **i18n**: Поддержка нескольких языков (EN, RU, UK)

## Возможности

✨ **Современный дизайн**
- Glass Minimal UI в стиле Apple/Notion
- Тёмная тема с эффектами glassmorphism
- Плавные анимации и переходы
- Адаптивная вёрстка

🌍 **Интернационализация**
- Автоматическое определение языка
- Поддержка английского, русского, украинского
- Легко добавить новые языки

🔒 **Безопасность и приватность**
- Сквозное шифрование (E2EE)
- Федеративная архитектура
- Без отслеживания и сбора данных
- Открытый исходный код

💬 **Коммуникация**
- Текстовые каналы и личные сообщения
- Голосовые каналы (WebRTC)
- Поддержка GIF (Tenor/Giphy)
- Обмен сообщениями в реальном времени (WebSocket)

## Документация

- [SECURITY.md](SECURITY.md) — Как сообщать об уязвимостях
- [docs/PRIVACY.md](docs/PRIVACY.md) — Черновик политики конфиденциальности
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — Публичный хостинг: TLS, `VITE_API_BASE`, CORS/WS, бэкапы
- [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md) — Как собрать desktop-приложение (.exe)

## Качество кода (локально)

**Backend** (`backend/`):

```bash
go vet ./...
go test ./... -count=1
# go test -race ./...   # на Linux/macOS и Windows amd64; на win/386 недоступен
```

Линтер: [golangci-lint](https://golangci-lint.run/) с конфигом `backend/.golangci.yml` (тот же запускается в CI).

**Client** (`client/`):

```bash
npm run lint
npm run test
npm run test:coverage
npm run build
```

## Структура проекта

```
zenthril/
├── backend/          # Go-сервер (узел федеративной сети)
│   ├── config/       # Конфигурация через переменные окружения
│   ├── migrations/   # SQL-миграции PostgreSQL
│   └── main.go       # Точка входа HTTP-сервера
├── client/           # Tauri + Vite + React/TypeScript desktop-клиент
│   ├── src/
│   │   ├── components/  # React-компоненты
│   │   ├── i18n/        # Интернационализация (EN, RU, UK)
│   │   └── store/       # Управление состоянием
│   └── src-tauri/    # Tauri (Rust) desktop-обёртка
├── docs/             # Документация
├── docker-compose.yml
└── .env.example
```

## Быстрый старт

```bash
# 1. Скопировать переменные окружения
cp .env.example .env

# 2. Запустить backend-сервисы (PostgreSQL + Redis)
docker compose up -d

# 3. Установить зависимости клиента
cd client
npm install

# 4. Скопировать переменные окружения клиента
cp .env.example .env

# 5. Запустить dev-сервер
npm run dev -- --host 0.0.0.0 --port 1420
```

Открой `http://localhost:1420/`. Backend будет доступен на `http://localhost:8080/`.

## Переменные окружения

### Backend (корень репозитория)

- `DB_URL` (обязательно) — Строка подключения к PostgreSQL
- `REDIS_URL` (по умолчанию: `redis://localhost:6379`)
- `JWT_SECRET` (обязательно) — Секретный ключ для JWT-токенов
- `HTTP_ADDR` (по умолчанию: `:8080`)
- `CORS_ALLOWED_ORIGINS` (опционально) — Разрешённые CORS origins
- `WS_ALLOWED_ORIGINS` (опционально) — Разрешённые WebSocket origins
- `ADMIN_USER_IDS` (опционально, UUID через запятую) — Доступ к `/api/v1/admin/*` (включая global ban)

### Client (папка `client/`)

Скопируй `client/.env.example` → `client/.env`.

- `VITE_API_BASE` (для production-сборки) — Origin backend'а, например `https://api.example.com` (см. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md))
- `VITE_TENOR_KEY` (опционально) — API-ключ Tenor для поиска GIF
- `VITE_GIPHY_KEY` (опционально) — API-ключ Giphy для поиска GIF

## Сборка desktop-приложения (Windows)

Для сборки desktop-приложения на Windows нужны **Visual Studio Build Tools** (с `link.exe`):

1. Скачай [Visual Studio Build Tools 2022](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)
2. Выбери "Desktop development with C++"
3. Установи (~6 ГБ)
4. Перезапусти терминал

Затем собери:

```bash
cd client
npm run tauri build
```

Файл `.exe` будет в `client/src-tauri/target/release/bundle/`.

Подробные инструкции в [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md).

## Скриншоты

### Glass Minimal дизайн
![Экран авторизации](https://via.placeholder.com/800x500?text=Auth+Screen+with+Language+Switcher)

### Поддержка нескольких языков
- 🇬🇧 English
- 🇷🇺 Русский
- 🇺🇦 Українська

Язык автоматически определяется из настроек браузера и может быть изменён вручную.

## Участие в разработке

Мы приветствуем вклад в проект! Пожалуйста, прочитайте [SECURITY.md](SECURITY.md) перед сообщением об уязвимостях.

## Лицензия

Этот проект с открытым исходным кодом. См. файл LICENSE для деталей.

## Благодарности

- Дизайн вдохновлён Apple и Notion
- UI-компоненты от [shadcn/ui](https://ui.shadcn.com/)
- Иконки от [Lucide](https://lucide.dev/)
