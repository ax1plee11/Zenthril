#!/bin/bash
set -euo pipefail

echo "Starting Zenthril development stack..."

docker compose up -d postgres redis

echo "Waiting for local services..."
sleep 5

cd "/workspaces/$(basename "$PWD")/backend"
go run . &
BACKEND_PID=$!
echo "Backend started with PID ${BACKEND_PID}"

cleanup() {
  echo "Stopping backend..."
  kill "${BACKEND_PID}" 2>/dev/null || true
}
trap cleanup EXIT

cd "/workspaces/$(basename "$PWD")/client"
npm run dev -- --host 0.0.0.0
