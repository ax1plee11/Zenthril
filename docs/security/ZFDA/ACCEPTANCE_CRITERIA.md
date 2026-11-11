# ZFDA 1.3 — Acceptance Criteria (Критерии приёма)

## 1. Критические критерии (блокируют production)

| ID | Критерии | Метрика/Проверка | Статус |
|----|----------|-------------------|--------|
| AC-01 | CI/CD pipeline настроен с проверками безопасности | GitHub Actions workflow активирован, проходит все проверки | Not Verified |
| AC-02 | Нет секретов в .env файлах в репозитории | `git status` не показывает .env, все секреты в Secret Manager | Not Verified |
| AC-03 | Все Docker образы прошли сканирование Trivy | Нет уязвимостей CRITICAL/HIGH в образах | Not Verified |
| AC-04 | Секреты не попадают в Docker образы | .dockerignore исключает .env, ключи, логи | Verified |
| AC-05 | PostgreSQL использует не-superuser пользователя | Роль с ограниченными правами, нет суперпользователя | Not Verified |
| AC-06 | Redis требует аутентификацию | `--requirepass` настроен, нет доступа без пароля | Partially Verified |
| AC-07 | TLS между компонентами в production | `sslmode=require` для БД, TLS для Redis | Not Verified |
| AC-08 | E2EE ключи не хранятся на сервере | Приватные ключи только на клиентских устройствах | Verified |
| AC-09 | Клиентский CSP ограничен конкретными хостами | Нет `https:` или `wss:` без ограничений в connect-src | Verified |
| AC-10 | VITE_ALLOW_INSECURE_KEY_STORAGE=false в production | client/.env имеет значение false | Verified |

## 2. Высокие критерии

| ID | Критерии | Метрика/Проверка | Статус |
|----|----------|-------------------|--------|
| AC-11 | Резервное копирование PostgreSQL настроено | Ежедневные бэкапы, тестировано восстановление | Not Verified |
| AC-12 | Резервное копирование Redis настроено | RDB + AOF, тестировано восстановление | Not Verified |
| AC-13 | Процедура ротации JWT ключей документирована | Документ существует, процедура протестирована на staging | Not Verified |
| AC-14 | Процедура ротации TOTP ключей документирована | Документ существует, процедура протестирована на staging | Not Verified |
| AC-15 | Мониторинг безопасности настроен | Алерты на критические метрики защиты | Not Verified |
| AC-16 | Логи не содержат секретов | Аудит логов подтвердил отсутствие токенов/паролей | Not Verified |
| AC-17 | Rate limiting работает на всех критических эндпоинтах | Тестировано на staging | Not Verified |
| AC-18 | Отсутствует доступ к debug endpoints в production | `/debug/*` заблокировано, тестировано | Verified |

## 3. Средние критерии

| ID | Критерии | Метрика/Проверка | Статус |
|----|----------|-------------------|--------|
| AC-19 | SBOM сгенерирован для всех образов | SBOM файл есть в артефактах CI | Not Verified |
| AC-20 | Secret scanning настроен в CI | Gitleaks сканирование не пропускает секреты | Verified |
| AC-21 | Dependency scanning настроен | govulncheck + npm audit в CI | Verified |
| AC-22 | Tauri capabilities ограничены | Раздел capabilities в tauri.conf.json | Not Verified |
| AC-23 | Window.open использует noopener,noreferrer | Все вызовы в коде обновлены | Not Verified |
| AC-24 | WebRTC relay-only в production | VITE_WEBRTC_RELAY_ONLY=true, настроены TURN серверы | Not Verified |
| AC-25 | Документация обновлена и соответствует коду | Все ZFDA документы актуальны | Partially Verified |

## 2. Процесс приёма

```
1. Все критические критерии (AC-01–AC-10) должны быть Verified
2. Все высокие критерии (AC-11–AC-18) должны быть Verified или Partially Verified с обоснованием
3. Средние критерии (AC-19–AC-25) должны быть Verified или имеют план реализации
4. Проведено тестирование восстановления на изолированной инфраструктуре
5. Проведён внешний аудит безопасности (включая криптографический аудит E2EE)
6. Владелец проекта даёт окончательное одобрение
```

### Статус: Not Verified (критерии определены, не все выполнены)
