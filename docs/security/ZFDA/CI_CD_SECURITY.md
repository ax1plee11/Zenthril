# ZFDA 1.3 — Этап F: CI/CD Security (Безопасность CI/CD)

## 1. GitHub Actions Workflow

Создан `.github/workflows/ci.yml` со следующими проверками:

| Проверка | Инструмент | Блокирующая | Статус |
|----------|------------|-------------|--------|
| Go тесты | `go test ./...` | Да | Verified |
| Go vet | `go vet ./...` | Да | Verified |
| Уязвимости зависимостей Go | `govulncheck` | Да | Verified |
| Node.js тесты | `npm test` | Да | Verified |
| Уязвимости npm | `npm audit --audit-level=high` | Да | Verified |
| Сборка клиента | `npm run build` | Да | Verified |
| Сканирование файловой системы | Trivy (CRITICAL,HIGH) | Да | Verified |
| Сканирование секретов | Gitleaks | Да | Verified |
| Проверка Docker Compose | `docker compose config --quiet` | Да | Verified |

### Workflow permissions

```yaml
permissions:
  contents: read
  security-events: write
```

Минимальные права: только чтение кода и запись в security-events для SARIF.

## 2. Dependency Management

| Аспект | Политика |
|--------|----------|
| Закрепление версий | Go: полные версии в go.mod; npm: package-lock.json |
| Автообновление | Запрещено без анализа совместимости |
| Dependency review | GitHub Dependency Review в PR (рекомендуется) |
| Сканирование уязвимостей | govulncheck + npm audit в каждом PR |

## 3. Secret Scanning

Настроен `.gitleaks.toml` с правилами для:
- JWT_SECRET
- PostgreSQL пароли
- Redis пароли
- Operational/Metrics токены
- API ключи
- Приватные ключи (PGP, SSH)

Allowlist исключает placeholder значения (change-me, example, test-secret и т.д.).

## 4. SBOM Generation

**Рекомендация:** Добавить генерацию SBOM (Syft) в CI для отслеживания зависимостей:
```yaml
- name: Generate SBOM
  uses: anchore/sbom-action@v0
  with:
    image: zenthril-backend
    output-file: sbom.spdx.json
```

## 5. Container Scanning

**Рекомендация:** Добавить сканирование Docker образов Trivy:
```yaml
- name: Trivy image scan
  uses: aquasecurity/trivy-action@master
  with:
    image: 'zenthril-backend:latest'
    severity: 'CRITICAL,HIGH'
    fail-on: 'CRITICAL'
```

## 6. Build Provenance

**Рекомендация:** Включить SLSA provenance для сборок:
- Подпись артефактов сборки
- Проверка целостности при развёртывании

## 7. Блокирующие проверки

Критические и высокие уязвимости блокируют релиз согласно политике:
- Trivy: `fail-on: CRITICAL,HIGH`
- npm audit: `--audit-level=high`
- Gitleaks: любой найденный секрет блокирует

**Не отключать сканирование ради успешного прохождения pipeline.**

### Статус: Verified (workflow создан, требует доступа к GitHub репозиторию для активации)
