// VERIX Knowledge Graph — схема Neo4j (раздел 05-06 ТЗ)
// Применяется при инициализации контейнера через /var/lib/neo4j/init/schema.cypher
// Совместимо с Neo4j Community Edition (UNIQUE, без NODE KEY).

// 06. Ограничения уникальности (первая миграция)
CREATE CONSTRAINT weakness_id IF NOT EXISTS
FOR (n:Weakness) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT risk_category_id IF NOT EXISTS
FOR (n:RiskCategory) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT cve_id IF NOT EXISTS
FOR (n:Vulnerability) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT rule_revision_key IF NOT EXISTS
FOR (n:RuleRevision) REQUIRE n.key IS UNIQUE;

CREATE CONSTRAINT detection_signal_id IF NOT EXISTS
FOR (n:DetectionSignal) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT mitigation_id IF NOT EXISTS
FOR (n:Mitigation) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT technology_id IF NOT EXISTS
FOR (n:Technology) REQUIRE n.id IS UNIQUE;

CREATE CONSTRAINT source_record_key IF NOT EXISTS
FOR (n:SourceRecord) REQUIRE n.key IS UNIQUE;

CREATE CONSTRAINT kb_release_id IF NOT EXISTS
FOR (n:KnowledgeRelease) REQUIRE n.id IS UNIQUE;

// Индексы для частых запросов чтения (раздел 09.2 метрики)
CREATE INDEX weakness_title IF NOT EXISTS FOR (n:Weakness) ON (n.title);
CREATE INDEX rule_revision_status IF NOT EXISTS FOR (n:RuleRevision) ON (n.status);
CREATE INDEX kb_release_published IF NOT EXISTS FOR (n:KnowledgeRelease) ON (n.published_at);

// 06.1 Канонический импорт одной записи (Пример: CWE + OWASP категорию)
// MERGE (w:Weakness {id: $cwe_id})
// ON CREATE SET w.created_at = datetime()
// SET w.title = $title, w.summary = $summary, w.source_version = $source_version, w.updated_at = datetime();
// MERGE (c:RiskCategory {id: $category_id})
// ON CREATE SET c.title = $category_title;
// MATCH (w:Weakness {id: $cwe_id}) MATCH (c:RiskCategory {id: $category_id})
// MERGE (w)-[:IN_CATEGORY {edition: $edition}]->(c);
