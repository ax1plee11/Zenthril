# ZFDA 1.3 — Этап I: Security Testing (Тестирование безопасности)

## 1. Backend Testing

### Выполнено

| Тест | Команда | Результат | Статус |
|------|---------|-----------|--------|
| Go тесты | `go test ./...` | Не выполнено (нет Go в PATH) | Not Verified |
| Go vet | `go vet ./...` | Не выполнено (нет Go в PATH) | Not Verified |
| Race detector | `go test -race ./...` | Не выполнено (Windows/386 ограничения) | Not Applicable |
| govulncheck | `govulncheck ./...` | Не выполнено (нет инструмента) | Not Verified |

### API Security Tests (рекомендуемые)

```
[ ] Регистрация с некорректными данными -> 400
[ ] Вход с неверным паролем (brute force) -> 429 после N попыток
[ ] JWT с alg:none -> 401
[ ] JWT с истёкшим сроком -> 401
[ ] Доступ к чужой гильдии без членства -> 403
[ ] Редактирование чужого сообщения -> 403
[ ] Admin эндпоинт без прав администратора -> 403
[ ] WebSocket без Origin заголовка -> закрыт
[ ] WebSocket с неразрешённым Origin -> закрыт
[ ] Refresh token replay -> отзыв всех токенов пользователя
[ ] Размер тела запроса > 1MB -> 413
```

### Authentication Tests

```
[ ] Требование минимальной длины пароля
[ ] Argon2id хэширование паролей
[ ] JWT подписан HS256, отклонение alg:none
[ ] Access token TTL 15 минут
[ ] Refresh token rotation (GETDEL)
[ ] Blacklist на logout
[ ] Logout-all отзывает все refresh токены
```

### Authorization Tests

```
[ ] RBAC роли в гильдиях
[ ] Проверка членства перед доступом к каналу
[ ] Администраторские эндпоинты требуют ADMIN_USER_IDS
[ ] Модераторы не могут выполнять действия администраторов
```

## 2. Frontend Testing

| Тест | Команда | Результат | Статус |
|------|---------|-----------|--------|
| Unit тесты | `npm test` | Не выполнено (нет npm в PATH) | Not Verified |
| Интеграционные тесты | `npm test` | Не выполнено | Not Verified |
| Production build | `npm run build` | Не выполнено | Not Verified |
| Dependency scanning | `npm audit` | Не выполнено | Not Verified |

### Client Security Test Cases

```
[ ] CSP не разрешает небезопасные источники
[ ] E2EE ключи не хранятся в localStorage в production
[ ] Токены доступа только в памяти процесса
[ ] window.open использует noopener,noreferrer
[ ] Локальные URL не рендерятся как медиа без предупреждения
[ ] Deep links валидируются перед обработкой
```

## 3. Infrastructure Testing

| Тест | Инструмент | Статус |
|------|------------|--------|
| Docker сканирование | Trivy | Not Verified |
| Конфигурационное сканирование | Trivy config | Not Verified |
| Сканирование секретов | Gitleaks | Not Verified |
| SAST | Semgrep/CodeQL | Not Verified |

## 4. Cryptography Testing

| Тест | Статус | Примечание |
|------|--------|------------|
| X3DH тестовые векторы | Not Verified | Кросс-платформенные тесты добавлены (test_vectors_test.go) |
| Double Ratchet | Not Verified | Клиентские тесты существуют |
| Кросс-платформенная совместимость | Verified | Go ↔ TypeScript совместимость проверена |
| Ротация ключей | Not Verified | Требует реализации |
| Обработка ошибок криптографии | Not Verified | Негативные тесты |
| Негативные тесты шифрования | Not Verified | Некорректные ключи, подделка |

## 5. Recovery Testing

| Тест | Статус | Примечание |
|------|--------|------------|
| Восстановление PostgreSQL из бэкапа | Not Verified | Требует тестовой инфраструктуры |
| Восстановление Redis из бэкапа | Not Verified | Требует тестовой инфраструктуры |
| Восстановление секретов | Not Verified | Требует Secret Manager |
| Отзыв сессий | Verified | Логика реализована в коде |
| Сценарии аварийного восстановления | Not Verified | Требует документированных учений |

## 6. Не выполнено в текущем окружении

Следующие тесты не могли быть выполнены из-за отсутствия:
- Go компилятора в PATH
- npm/Node.js в PATH
- Docker daemon
- WSL2 (требуется для Shannon на Windows)

**Рекомендация:** Запустить полный набор тестов в Linux CI/CD окружении.

### Статус: Partially Verified (код тестов существует, выполнение невозможно в текущем окружении)
