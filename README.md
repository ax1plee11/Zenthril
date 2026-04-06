# Veltrix

Децентрализованный мессенджер с федеративной архитектурой и сквозным шифрованием.

## Стек

- **Бэкенд**: Go + PostgreSQL + Redis
- **Клиент**: Tauri + Vite + React/TypeScript
- **Шифрование**: X25519 + AES-256-GCM (E2EE)
- **Аутентификация**: JWT + Argon2id

## Структура

```
veltrix/
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
cd backend && go run main.go
```
