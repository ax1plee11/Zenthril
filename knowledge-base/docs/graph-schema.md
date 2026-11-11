# Graph Schema — модель данных Neo4j

## Метки узлов (Node Labels)

| Метка | Ключ | Поля |
|---|---|---|
| Weakness | id (CWE-89) | title, summary, status, abstraction, source_version, reviewed_at, created_at, updated_at |
| RiskCategory | id (OWASP-2025-A05) | title, edition, source_url, created_at, updated_at |
| Vulnerability | id (CVE-YYYY-NNNN) | published_at, modified_at, cvss, cvss_version, state, created_at, updated_at |
| TestMethod | id (WSTG ID) | title, source_version, reference_url, created_at, updated_at |
| RuleRevision | key (rule_id:version) | handler_key, status, safety_class, budget, checksum, created_at, updated_at |
| DetectionSignal | id (VRX-SIGNAL-...) | signal_type, description, verification_level, created_at, updated_at |
| Mitigation | id (VRX-FIX-...) | title, recommendation, language, tech_scope, created_at, updated_at |
| Technology | id (tech:family) | name, family, version_scope, created_at, updated_at |
| SourceRecord | key (source:external_id:version) | url, license_id, retrieved_at, sha256, created_at, updated_at |
| KnowledgeRelease | id (kb-YYYY.MM.N) | published_at, approved_by, manifest_sha256, active, created_at, retired_at |

## Ограничения (UNIQUE, Community Edition)
```cypher
CREATE CONSTRAINT weakness_id IF NOT EXISTS FOR (n:Weakness) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT risk_category_id IF NOT EXISTS FOR (n:RiskCategory) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT cve_id IF NOT EXISTS FOR (n:Vulnerability) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT rule_revision_key IF NOT EXISTS FOR (n:RuleRevision) REQUIRE n.key IS UNIQUE;
CREATE CONSTRAINT source_record_key IF NOT EXISTS FOR (n:SourceRecord) REQUIRE n.key IS UNIQUE;
CREATE CONSTRAINT kb_release_id IF NOT EXISTS FOR (n:KnowledgeRelease) REQUIRE n.id IS UNIQUE;
```

## Отношения

| Откуда →куда | Тип | Смысл |
|---|---|---|
| Weakness → RiskCategory | IN_CATEGORY | Тип ошибки в категории OWASP редакции |
| Vulnerability → Weakness | ASSOCIATED_WITH | CVE классифицирована как CWE |
| TestMethod → Weakness | ASSESSES | Методика оценивает класс ошибки |
| RuleRevision → Weakness | DETECTS | Правило проверяет гипотезу о CWE |
| RuleRevision → DetectionSignal | EXPECTS | Сигналы для подтверждения |
| RuleRevision → Technology | APPLIES_TO | Где применимо (с ограничениями версии) |
| Mitigation → Weakness | ADDRESSES | Снижение риска / устранение причины |
| SourceRecord → * | SUPPORTS | Происхождение утверждения + ссылка |
| KnowledgeRelease → RuleRevision | INCLUDES | Версия правила внутри релиза |

## Инварианты
- Все связи ведут к существующим сущностям (нет битых связей).
- CWE/CVE/OWASP не смешиваются.
- У RuleRevision определён безопасный handler и лимиты.
- Опубликованные объекты имеют SourceRecord или документ ревью.
- Повторный импорт одного snapshot не меняет число узлов (идемпотентность MERGE).
