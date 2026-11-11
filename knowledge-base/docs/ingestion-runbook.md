# Ingestion Runbook — от источника до графа

Шесть стадий ETL (раздел 04 ТЗ): Acquire → Validate → Normalize → Link & dedupe → Review → Publish.

## 01 Acquire
Скачать официальный датасет или получить через API, записать headers/время/хэш.
- MITRE CWE 5.0: https://cwe.mitre.org/data/downloads.html
- OWASP Top 10:2025: https://top10.owasp.org/2025/A05_2025-Injection/
- CISA KEV JSON: https://www.cisa.gov/known-exploited-vulnerabilities-catalog
- FIRST EPSS CSV: https://www.first.org/epss/data.html

Артефакт: raw snapshot + manifest.json (источник, версия, URL, fetched_at, sha256, license_id).

## 02 Validate
Проверить тип, схему, ID, даты, размер, отсутствие неожиданного HTML/архива.
При ошибке — отказ, запись не попадает в граф.

## 03 Normalize
Единый JSON: CWE-89, CVE-..., OWASP-2025-A05, UTF-8, UTC, языки.
Результат: Canonical records v1.

## 04 Link & dedupe
MERGE по стабильному ключу, разрешить алиасы, проверить связи.
Повторный импорт не должен менять число узлов.

## 05 Review
Проверить лицензии, смысл отношений, безопасность и корректность правил.
Решение: approved / rejected.

## 06 Publish
Сформировать immutable KnowledgeRelease, обновить указатель active.
До публикации клиент видит прошлый утверждённый релиз.

## Ежедневные операции
- Сохранять контролируемый экспорт проверенных данных/манифестов.
- Тестировать восстановление на отдельной БД не реже одного раза в месяц.
- Собирать метрики: latency, ошибки driver, объём базы, длительность ETL, возраст источников, число активных правил.

## Запуск ETL (админ API)
```
POST /internal/kb/ingest/runs   Authorization: Bearer <KB_ADMIN_TOKEN>
POST /internal/kb/releases/{id}/publish   Authorization: Bearer <KB_ADMIN_TOKEN>
```
