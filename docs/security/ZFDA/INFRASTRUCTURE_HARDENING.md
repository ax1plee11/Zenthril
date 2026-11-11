# ZFDA 1.3 — Этап B: Infrastructure Security (Инфраструктурная безопасность)

## 1. PostgreSQL

### Конфигурация (docker-compose.yml)

| Параметр | Значение | Статус |
|----------|----------|--------|
| Образ | `postgres:16-alpine@sha256:...` (закреплён) | Verified |
| Пользователь | `999:999` (не root) | Verified |
| Сетевая изоляция | Внутренняя сеть `zenthril-internal` | Verified |
| Порт | Не опубликован наружу (только `expose`) | Verified |
| Health check | `pg_isready` каждые 5с | Verified |
| Read-only FS | `read_only: true` с `tmpfs` для данных | Verified |
| Capabilities | `cap_drop: ALL` | Verified |
| TLS | `sslmode=disable` (внутри доверенной сети Docker) | Partially |

### Рекомендации (не выполнено — требует доступа к production серверу)

1. **Создать отдельного не-superuser пользователя** для бэкенда:
   ```sql
   CREATE ROLE zenthril_app WITH LOGIN PASSWORD '<strong-password>';
   GRANT CONNECT ON DATABASE zenthril TO zenthril_app;
   GRANT USAGE ON SCHEMA public TO zenthril_app;
   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO zenthril_app;
   ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE TO zenthril_app;
   REVOKE ALL ON DATABASE zenthril FROM PUBLIC;
   ```

2. **Включить TLS** для соединений между бэкендом и БД в production:
   - `sslmode=require` в DB_URL
   - Настроить сертификаты PostgreSQL

3. **Включить аудит PostgreSQL** (`pgaudit` extension)

4. **Регулярные бэкапы** — см. Этап E

### Статус: Partially Verified

---

## 2. Redis

### Конфигурация (docker-compose.yml)

| Параметр | Значение | Статус |
|----------|----------|--------|
| Образ | `redis:7-alpine@sha256:...` (закреплён) | Verified |
| Пользователь | `999:999` (не root) | Verified |
| Аутентификация | `--requirepass ${REDIS_PASSWORD}` | Verified |
| Сетевая изоляция | Внутренняя сеть `zenthril-internal` | Verified |
| Порт | Не опубликован наружу | Verified |
| Persistence | AOF с reglas сохранения | Verified |
| Read-only FS | `read_only: true` с `tmpfs` для /data | Verified |
| Capabilities | `cap_drop: ALL` | Verified |

### Рекомендации

1. **Настроить Redis ACL** для разделения доступа (разные пользователи для разных операций)
2. **Включить TLS** если Redis используется через недоверенную сеть
3. **Настроить мониторинг памяти** и лимиты (`maxmemory-policy`)
4. **Регулярное резервное копирование** AOF/RDB файлов

### Статус: Partially Verified

---

## 3. Docker

### Конфигурация (docker-compose.yml)

| Параметр | Значение | Статус |
|----------|----------|--------|
| Multi-stage build | Да (backend/Dockerfile) | Verified |
| Non-root user | Да (`USER zenthril`) | Verified |
| .dockerignore | Создан (исключает .env, ключи, логи) | Verified |
| Read-only FS | Да (`read_only: true`) | Verified |
| Capabilities | `cap_drop: ALL` | Verified |
| No-new-privileges | Да | Verified |
| Resource limits | `mem_limit`, `cpus` на всех контейнерах | Verified |
| Network segmentation | Внутренняя сеть `zenthril-internal` (internal: true) | Verified |
| Pinned image versions | Да (sha256 дайджесты) | Verified |
| Health checks | Да на всех сервисах | Verified |
| Нет секретов в образах | Да (.dockerignore исключает .env) | Verified |

### Статус: Verified (для конфигурации, требует развёртывания для верификации)

---

## 4. Операционная система и сервер

### Рекомендации (для production сервера)

**OS Hardening:**
- Использовать минимальный дистрибутив (Alpine Linux или hardened Debian)
- Отключить ненужные службы и порты
- Настроить автоматические обновления безопасности (`unattended-upgrades`)
- Включить AppArmor/SELinux
- Настроить auditd для логирования системных событий

**SSH:**
- Отключить парольную аутентификацию (только ключи)
- Изменить стандартный порт SSH
- Использовать fail2ban для защиты от bruteforce
- Ограничить доступ по IP (allowlist)

**Firewall:**
- Закрыть все порты кроме 80/443 (через reverse proxy)
- PostgreSQL (5432) и Redis (6379) недоступны извне
- Внутренняя сеть Docker изолирована

**Минимизация пакетов:**
- Удалить компиляторы, отладочные инструменты из production
- Установить только необходимые пакеты

**Изоляция сервисов:**
- Каждый сервис в отдельном контейнере
- Нет совместного использования volumes между недоверенными контейнерами

### Статус: Not Verified (требует доступа к production серверу)

---

## Итог Этапа B

Все конфигурации инфраструктуры обновлены с учётом принципов ZFDA. Каждый компонент имеет собственную защиту:
- PostgreSQL: не root пользователь, внутренняя сеть, read-only FS, нет 공개 портов
- Redis: аутентификация по паролю, ACL рекомендовано, внутренняя сеть
- Docker: multi-stage, non-root, .dockerignore, capabilities drops, resource limits

Следующий этап: C — Secrets Management.
