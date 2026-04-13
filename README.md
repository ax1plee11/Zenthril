# Zenthril

Децентрализованный мессенджер с федеративной архитектурой и сквозным шифрованием.

## Стек

- **Бэкенд**: Go + PostgreSQL + Redis
- **Клиент**: Tauri + Vite + React/TypeScript
- **Шифрование**: X25519 + AES-256-GCM (E2EE)
- **Аутентификация**: JWT + Argon2id

## Документы

- [SECURITY.md](SECURITY.md) — как сообщать об уязвимостях
- [docs/PRIVACY.md](docs/PRIVACY.md) — черновик политики конфиденциальности для публичного сервиса
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — публичный хостинг: TLS, `VITE_API_BASE`, CORS/WS, бэкапы

## Качество кода (локально)

**Бэкенд** (`backend/`):

```bash
go vet ./...
go test ./... -count=1
# go test -race ./...   # на Linux/macOS и Windows amd64; на win/386 недоступен
```

Линтер: [golangci-lint](https://golangci-lint.run/) с конфигом `backend/.golangci.yml` (тот же запускается в CI).

**Клиент** (`client/`):

```bash
npm run lint
npm run test
npm run test:coverage
npm run build
```

## Структура

```
zenthril/
├── backend/          # Go-сервер (узел федеративной сети)
│   ├── config/       # Конфигурация через env-переменные
│   ├── migrations/   # SQL-миграции PostgreSQL
│   └── main.go       # Точка входа HTTP-сервера
├── client/           # Tauri + Vite + React/TypeScript десктоп-клиент
├── docker-compose.yml
└── .env.example
```

## Быстрый старт

```bash
cp .env.example .env
docker compose up -d
cd client
npm i
cp .env.example .env
npm run dev -- --host 0.0.0.0 --port 1420
```

Открой `http://localhost:1420/`. Бэкенд будет доступен на `http://localhost:8080/`.

## Переменные окружения

### Бэкенд (корень репозитория)

- `DB_URL` (обязательно)
- `REDIS_URL` (по умолчанию `redis://localhost:6379`)
- `JWT_SECRET` (обязательно)
- `HTTP_ADDR` (по умолчанию `:8080`)
- `CORS_ALLOWED_ORIGINS` (опционально)
- `WS_ALLOWED_ORIGINS` (опционально)
- `ADMIN_USER_IDS` (опционально, UUID через запятую) — доступ к `/api/v1/admin/*` (в т.ч. global ban)

### Клиент (папка `client/`)

Скопируй `client/.env.example` → `client/.env`.

- `VITE_API_BASE` (для прод-сборки) — origin бэкенда, например `https://api.example.com` (см. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md))
- `VITE_TENOR_KEY` (опционально) — поиск/нормализация Tenor
- `VITE_GIPHY_KEY` (опционально) — поиск/нормализация Giphy

## Tauri (Windows)

Для сборки десктоп-приложения на Windows нужны **Visual Studio Build Tools** (наличие `link.exe`), затем:

```bash
cd client/src-tauri
cargo build --release
```
