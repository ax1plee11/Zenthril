#!/bin/bash
# Запуск всего проекта в Codespaces

echo "🚀 Starting Zenthril..."

# Поднимаем PostgreSQL + Redis через Docker
docker compose up -d postgres redis

echo "⏳ Waiting for DB..."
sleep 5

# Запускаем бэкенд в фоне
cd /workspaces/$(basename $PWD)/backend
go run . &
BACKEND_PID=$!
echo "✅ Backend started (PID $BACKEND_PID)"

# Запускаем фронтенд
cd /workspaces/$(basename $PWD)/client
npm run dev -- --host 0.0.0.0

# При выходе останавливаем бэкенд
kill $BACKEND_PID
