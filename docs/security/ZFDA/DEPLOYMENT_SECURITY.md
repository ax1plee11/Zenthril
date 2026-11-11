# ZFDA 1.3 — Этап J: Production Deployment Plan (План развёртывания production)

## 1. Архитектура production

```
Internet
    │
    ▼
[Reverse Proxy: Nginx/Caddy/Traefik] :443 (TLS terminated)
    │
    ├──► [Zenthril Backend] :8080 (внутренняя сеть)
    │        ├──► PostgreSQL :5432 (внутренняя сеть, не公開)
    │        └──► Redis :6379 (внутренняя сеть, не公開)
    │
    └──► [Static Assets / CDN]
```

### Серверные компоненты

| Компонент | Размещение | Сетевой доступ |
|-----------|------------|----------------|
| Reverse Proxy | DMZ / External | :443 HTTPS |
| Backend | Internal network | :8080 (только от reverse proxy) |
| PostgreSQL | Internal network | :5432 (только от backend) |
| Redis | Internal network | :6379 (только от backend) |
| Monitoring | Internal network | :9090 Prometheus (только внутренний) |

## 2. Сетевая схема

```
┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐
│   Internet   │◄────►│  Reverse Proxy   │◄────►│   Backend API   │
│   (HTTPS)    │      │  (TLS termination)│     │  (:8080 internal)│
└─────────────┘      └──────────────────┘      └────────┬────────┘
                                                        │
                    ┌──────────────────┐               │
                    │   PostgreSQL     │◄──────────────┘
                    │   (:5432 internal)│
                    └──────────────────┘
                                                        │
                    ┌──────────────────┐               │
                    │      Redis       │◄──────────────┘
                    │   (:6379 internal)│
                    └──────────────────┘
```

### Правила брандмауэра

| Откуда | Куда | Порт | Протокол | Обоснование |
|--------|------|------|----------|-------------|
| Интернет | Reverse Proxy | 443 | HTTPS | Клиентский доступ |
| Интернет | Reverse Proxy | 80 | HTTP (редирект на 443) | Принудительный HTTPS |
| Reverse Proxy | Backend | 8080 | HTTP | API доступ |
| Backend | PostgreSQL | 5432 | TCP | База данных |
| Backend | Redis | 6379 | TCP | Кэш и сессии |
| Мониторинг | Backend | 8081 | HTTP | Health checks (только внутренняя сеть) |
| Все остальное | Все | Все | Все | Запрещено (default deny) |

## 3. TLS

| Соединение | Сертификат | Источник |
|------------|------------|----------|
| Клиент ↔ Reverse Proxy | Публичный сертификат | Let's Encrypt / коммерческий CA |
| Reverse Proxy ↔ Backend | Внутренний сертификат или отключён (внутри trusted сети) | Самоподписанный или внутренний CA |
| Backend ↔ PostgreSQL | TLS (требует настройки) | Внутренний CA |
| Backend ↔ Redis | TLS (требует настройки) | Внутренний CA |

**Конфигурация:** Предпочтительно terminating TLS на reverse proxy (Caddy/Nginx/Traefik). Бэкенд получает HTTP от trusted прокси.

## 4. Управление секретами

| Секрет | Источник в production |
|--------|----------------------|
| JWT_SECRET | Secret Manager (Vault/AWS Secrets Manager/K8s Secrets) |
| POSTGRES_PASSWORD | Secret Manager |
| REDIS_PASSWORD | Secret Manager |
| OPERATIONAL_TOKEN | Secret Manager |
| METRICS_TOKEN | Secret Manager |
| FEDERATION_TOKEN | Secret Manager (если включена федерация) |
| NODE_PRIVATE_KEY | Secret Manager / HSM |

Секреты inject'ятся в контейнеры через переменные окружения из secret manager. Не использовать .env файлы в production.

## 5. Мониторинг и логирование

| Компонент | Инструмент | Метрики/Логи |
|-----------|------------|--------------|
| Backend метрики | Prometheus | zenthril_ws_*, readiness_failures, HTTP запросы |
| Backend логи | Struktured logging (slog) | stdout → сборщик логов (ELK/Loki) |
| Infrastructure мониторинг | Prometheus + Node Exporter | CPU, Memory, Disk, Network |
| Алерты | Alertmanager | Пороговые значения метрик защиты, downtime |

### Ключевые алерты

- `zenthril_readiness_failures_total > 0` — бэкенд не готов
- `zenthril_ws_rate_limit_hits_total` резкий рост — возможная атака
- `go_memstats_alloc_bytes` рост — утечка памяти
- Postgres connection pool истощение

## 6. Резервное копирование

| Компонент | Стратегия | Retention | Offsite |
|-----------|-----------|-----------|---------|
| PostgreSQL | pg_dump + WAL archiving | 30 дней | Да |
| Redis | RDB + AOF | 7 дней | Да |
| Конфигурации | Git репозиторий | Все версии | Да (GitHub) |
| Секреты | Secret Manager с резервной копией | По 필요 | Да |

## 7. Обновления и Rollback

### Обновление приложения

```
1. Собрать новый Docker образ с тегом версии
2. Провести сканирование образа (Trivy)
3. Развернуть в staging, провести тесты
4. Обновить docker-compose с новым тегом
5. Выполнить rolling update (один контейнер за другим)
6. Проверить health endpoints после каждого шага
```

### Rollback

```
1. Откатить docker-compose к предыдущему тегу образа
2. Выполнить docker compose up -d
3. Проверить health endpoints
4. При необходимости откатить БД миграции (если не обратно-совместимы)
```

## 8. Разделение конфигураций

| Окружение | ENVIRONMENT | SECURE_COOKIES | CORS_ALLOWED_ORIGINS | EVENT_BUS_DRIVER | Federation |
|-----------|-------------|----------------|---------------------|------------------|------------|
| Development | development | false | localhost | memory | false |
| Testing | testing | false | тестовые домены | memory | false |
| Staging | staging | true | staging домены | kafka/redpanda | false |
| Production | production | true | production домены (HTTPS) | kafka/redpanda | false (alpha) |

**Production не должен использовать:**
- Тестовые секреты (change-me, example и т.д.)
- Debug endpoints (/debug/* заблокированы)
- Development-конфигурации (ENVIRONMENT=development)
- CORS wildcard (`*`)
- Memory event bus (требуется kafka/nats)

## 9. Incident Response

| Инцидент | Первая реакция | Ответственный |
|----------|---------------|---------------|
| Компрометация секрета | Ротация секрета, отзыв сессий | Администратор инфраструктуры |
| Уязвимость в зависимости | Обновление зависимости, пересборка | Команда разработки |
| DDoS атака | Включение rate limiting, блокировка IP | Администратор инфраструктуры |
| Утечка данных | Отключение системы, оценка ущерба, уведомление пользователей | Руководство проекта |
| Недоступность сервиса | Переключение на резервный узел, диагностика | Администратор инфраструктуры |

### Контакты

- Сообщение об уязвимости: ax1plee@gmail.com (из SECURITY.md)
- Время отклика: best effort, цель 48 часов

### Статус: Not Verified (план создан, требует доступа к production инфраструктуре для реализации)
