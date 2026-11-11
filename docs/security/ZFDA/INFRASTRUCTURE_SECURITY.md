# ZFDA INFRASTRUCTURE_SECURITY — Безопасность инфраструктуры

## Hardening операционной системы

- Нет документации по hardening ОС.
- Нет установки обновлений безопасности в образах по умолчанию.

## Принцип минимальных привилегий

- Пользователь PostgreSQL с ограниченными правами.
- Административный доступ только через `ADMIN_USER_IDS`.
- Операционные и metrics endpointы требуют bearer токен в production.

## Изоляция процессов и сервисов

- Docker контейнеры с лимитами памяти и CPU (postgres: 512мБ/0.75 CPU, redis: 256мБ/0.5 CPU, backend: 512мБ/1.0 CPU).
- Нет пользователя non-root в Dockerfile (требуется проверка).
- Нет сетевой изоляции между контейнерами (одна сеть Compose).

## Безопасная конфигурация контейнеров

- `docker-compose.yml` с health checkami.
- Отсутствие опубликованных портов для БД и Redis.
- `docker-compose.debug.yml` для локальной отладки с公開анием портов.
- Нет сканирования контейнеров как блокирующего гейта (Trivy с continue-on-error, дефект F-04).

## Управление секретами

- Секреты через переменные окружения (.env).
- Нет HashiCorp Vault или внешнего менеджера секретов.
- Нет ротации секретов.
- Fail-closed валидация placeholder секретов в production.

## Защита CI/CD

- `.github/workflows/ci.yml` — сборка, тесты, линтинг, бенчмарки.
- `.github/workflows/supply-chain.yml` — govulncheck (гейт без continue-on-error), CodeQL (гейт без continue-on-error).
- `.github/workflows/build-desktop.yml` — сборка десктопа.
- `.github/workflows/deploy-pages.yml` — деплой документации.
- Нет подписывания коммитов, нет проверки подписейリリース.

## Контроль административного доступа

- `adminOnly` middleware для `/admin` endpointов.
- Глобальная блокировка/разблокировка пользователей.
- Админ не может быть выдан через API ролей гильдии.

## Защита от lateral movement

- Нет изоляции между компонентами внутри backend монолита.
- Общий JWT_SECRET для всех токенов.
- Общий Redis для всех сессий и токенов.
- Общий PostgreSQL для всех данных.

Проверено: 2026-10-30.