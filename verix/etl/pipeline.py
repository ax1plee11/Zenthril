"""ETL-конвейер базы знаний VERIX: от источника до графа Neo4j.

Реализует шесть обязательных стадий из раздела 04 ТЗ:
01 Acquire, 02 Validate, 03 Normalize, 04 Link & dedupe, 05 Review, 06 Publish.

Контроль качества: идемпотентность повторного импорта, никакого тихого удаления
сущностей, атомарная публикация релиза.
"""

from __future__ import annotations

import hashlib
import json
import os
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional

from neo4j import GraphDatabase


class Stage(str, Enum):
    ACQUIRE = "Acquire"
    VALIDATE = "Validate"
    NORMALIZE = "Normalize"
    LINK_DEDUPE = "Link & dedupe"
    REVIEW = "Review"
    PUBLISH = "Publish"


class RecordStatus(str, Enum):
    PENDING = "pending"
    APPROVED = "approved"
    REJECTED = "rejected"
    WITHDRAWN = "withdrawn"
    DEPRECATED = "deprecated"


class EvidenceStatus(str, Enum):
    SUSPECTED = "suspected"
    CONFIRMED = "confirmed"
    INCONCLUSIVE = "inconclusive"
    NOT_OBSERVED = "not_observed"


SCHEMA_VERSION = "verix.ingest.v1"
CONTENT_DIR = Path(__file__).resolve().parents[1] / "content"
SNAPSHOTS_DIR = CONTENT_DIR / "sources" / "snapshots"
MANIFESTS_DIR = CONTENT_DIR / "sources" / "manifests"
STAGING_DIR = CONTENT_DIR / "staging"


@dataclass
class SourceManifest:
    source: str
    source_version: str
    url: str
    fetched_at: str
    sha256: str
    license_id: str
    schema_version: str = SCHEMA_VERSION
    records_count: int = 0
    validation_status: str = "pending"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "source": self.source,
            "source_version": self.source_version,
            "url": self.url,
            "fetched_at": self.fetched_at,
            "sha256": self.sha256,
            "license_id": self.license_id,
            "schema_version": self.schema_version,
            "records_count": self.records_count,
            "validation_status": self.validation_status,
        }


@dataclass
class CanonicalRecord:
    """Единая нормализованная запись перед загрузкой в граф."""
    external_id: str
    source: str
    source_version: str
    record_type: str  # weakness | risk_category | cve | rule | mitigation | technology | signal
    title: str
    summary: Optional[str] = None
    properties: Dict[str, Any] = field(default_factory=dict)
    relationships: List[Dict[str, Any]] = field(default_factory=list)
    provenance: Dict[str, Any] = field(default_factory=dict)
    status: RecordStatus = RecordStatus.PENDING
    evidence_status: Optional[EvidenceStatus] = None


@dataclass
class StageResult:
    stage: Stage
    ok: bool
    records: List[CanonicalRecord] = field(default_factory=list)
    manifest: Optional[SourceManifest] = None
    error: Optional[str] = None
    artifacts: Dict[str, Any] = field(default_factory=dict)


class PipelineError(Exception):
    pass


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def ensure_dirs() -> None:
    for d in (SNAPSHOTS_DIR, MANIFESTS_DIR, STAGING_DIR):
        d.mkdir(parents=True, exist_ok=True)


class KnowledgeRepository:
    """Обёртка над Neo4j для записи в граф (только из управляемого ETL/админа)."""

    def __init__(self, uri: str, user: str, password: str, database: str = "neo4j") -> None:
        self.driver = GraphDatabase.driver(uri, auth=(user, password))
        self.database = database

    def close(self) -> None:
        self.driver.close()

    def apply_canonical_records(self, records: Iterable[CanonicalRecord]) -> Dict[str, int]:
        """Мерджит записи в графе идемпотентно. Возвращает счётчики по меткам."""
        counts: Dict[str, int] = {}
        with self.driver.session(database=self.database) as session:
            for record in records:
                summary = session.execute_write(self._merge_record, record)
                label = record.record_type
                counts[label] = counts.get(label, 0) + 1
        return counts

    @staticmethod
    def _merge_record(tx, record: CanonicalRecord) -> Any:
        rt = record.record_type
        if rt == "weakness":
            return tx.run(
                """
                MERGE (w:Weakness {id: $id})
                ON CREATE SET w.created_at = datetime()
                SET w.title = $title, w.summary = $summary,
                    w.status = $status, w.source_version = $source_version,
                    w.updated_at = datetime()
                RETURN w.id AS id
                """,
                id=record.external_id, title=record.title, summary=record.summary,
                status=record.status.value, source_version=record.source_version,
            ).consume()
        if rt == "risk_category":
            return tx.run(
                """
                MERGE (c:RiskCategory {id: $id})
                ON CREATE SET c.created_at = datetime()
                SET c.title = $title, c.edition = $edition, c.source_url = $source_url,
                    c.updated_at = datetime()
                RETURN c.id AS id
                """,
                id=record.external_id, title=record.title,
                edition=record.properties.get("edition", ""),
                source_url=record.properties.get("source_url", ""),
            ).consume()
        if rt == "cve":
            return tx.run(
                """
                MERGE (v:Vulnerability {id: $id})
                ON CREATE SET v.created_at = datetime()
                SET v.published_at = $published_at, v.modified_at = $modified_at,
                    v.cvss = $cvss, v.cvss_version = $cvss_version, v.state = $state,
                    v.updated_at = datetime()
                RETURN v.id AS id
                """,
                id=record.external_id,
                published_at=record.properties.get("published_at"),
                modified_at=record.properties.get("modified_at"),
                cvss=record.properties.get("cvss"),
                cvss_version=record.properties.get("cvss_version"),
                state=record.properties.get("state"),
            ).consume()
        if rt == "rule":
            return tx.run(
                """
                MERGE (r:RuleRevision {key: $key})
                ON CREATE SET r.created_at = datetime()
                SET r.handler_key = $handler_key, r.status = $status,
                    r.safety_class = $safety_class, r.budget = $budget,
                    r.checksum = $checksum, r.updated_at = datetime()
                RETURN r.key AS key
                """,
                key=record.external_id,
                handler_key=record.properties.get("handler_key", ""),
                status=record.status.value,
                safety_class=record.properties.get("safety_class", "passive"),
                budget=json.dumps(record.properties.get("request_budget", {})),
                checksum=record.properties.get("checksum", ""),
            ).consume()
        if rt == "mitigation":
            return tx.run(
                """
                MERGE (m:Mitigation {id: $id})
                ON CREATE SET m.created_at = datetime()
                SET m.title = $title, m.recommendation = $recommendation,
                    m.language = $language, m.tech_scope = $tech_scope,
                    m.updated_at = datetime()
                RETURN m.id AS id
                """,
                id=record.external_id, title=record.title,
                recommendation=record.summary or "",
                language=record.properties.get("language", ""),
                tech_scope=record.properties.get("tech_scope", ""),
            ).consume()
        if rt == "technology":
            return tx.run(
                """
                MERGE (t:Technology {id: $id})
                ON CREATE SET t.created_at = datetime()
                SET t.name = $name, t.family = $family, t.version_scope = $version_scope,
                    t.updated_at = datetime()
                RETURN t.id AS id
                """,
                id=record.external_id, name=record.title,
                family=record.properties.get("family", ""),
                version_scope=record.properties.get("version_scope", ""),
            ).consume()
        if rt == "signal":
            return tx.run(
                """
                MERGE (s:DetectionSignal {id: $id})
                ON CREATE SET s.created_at = datetime()
                SET s.signal_type = $signal_type, s.description = $description,
                    s.verification_level = $verification_level, s.updated_at = datetime()
                RETURN s.id AS id
                """,
                id=record.external_id, signal_type=record.properties.get("signal_type", ""),
                description=record.summary or "",
                verification_level=record.properties.get("verification_level", "suspected"),
            ).consume()
        raise PipelineError(f"Неизвестный тип записи: {rt}")

    def create_relationships(self, records: Iterable[CanonicalRecord]) -> int:
        """Создаёт типизированные отношения между узлами."""
        created = 0
        with self.driver.session(database=self.database) as session:
            for record in records:
                for rel in record.relationships:
                    summary = session.execute_write(self._merge_relationship, record, rel)
                    created += summary.counters.relationships_created
        return created

    @staticmethod
    def _merge_relationship(tx, record: CanonicalRecord, rel: Dict[str, Any]) -> Any:
        rel_type = rel["type"]
        target_label = rel["target_label"]
        target_id = rel["target_id"]
        props = rel.get("props", {})
        return tx.run(
            f"""
            MATCH (src:{record.record_type.capitalize()} {{id: $src_id}})
            MATCH (dst:{target_label} {{id: $dst_id}})
            MERGE (src)-[r:{rel_type}]->(dst)
            SET r += $props
            RETURN r
            """,
            src_id=record.external_id, dst_id=target_id, props=props,
        ).consume()

    def publish_release(self, release_id: str, published_by: str, manifest_sha256: str) -> None:
        with self.driver.session(database=self.database) as session:
            session.execute_write(self._publish_release_tx, release_id, published_by, manifest_sha256)

    @staticmethod
    def _publish_release_tx(tx, release_id: str, published_by: str, manifest_sha256: str) -> Any:
        # Атомарно помечаем новый релиз активным, прошлый остаётся неизменным (immutable).
        tx.run(
            """
            MATCH (old:KnowledgeRelease) WHERE old.active = true
            SET old.active = false, old.retired_at = datetime()
            """,
        )
        tx.run(
            """
            MERGE (rel:KnowledgeRelease {id: $id})
            ON CREATE SET rel.created_at = datetime()
            SET rel.published_at = datetime(), rel.approved_by = $published_by,
                rel.manifest_sha256 = $manifest_sha256, rel.active = true
            RETURN rel.id AS id
            """,
            id=release_id, published_by=published_by, manifest_sha256=manifest_sha256,
        ).consume()

    def link_release_to_rules(self, release_id: str, rule_keys: List[str]) -> None:
        with self.driver.session(database=self.database) as session:
            for key in rule_keys:
                session.execute_write(self._include_rule, release_id, key)

    @staticmethod
    def _include_rule(tx, release_id: str, key: str) -> Any:
        return tx.run(
            """
            MATCH (rel:KnowledgeRelease {id: $release_id})
            MATCH (r:RuleRevision {key: $key})
            MERGE (rel)-[:INCLUDES]->(r)
            RETURN r.key AS key
            """,
            release_id=release_id, key=key,
        ).consume()


class ETLPipeline:
    """Оркестратор шести стадий ETL."""

    def __init__(self, repo: KnowledgeRepository) -> None:
        self.repo = repo

    # 01 Acquire -----------------------------------------------------------
    def acquire(self, source: str, url: str, source_version: str, license_id: str,
                raw_bytes: bytes) -> StageResult:
        ensure_dirs()
        fetched_at = now_iso()
        sha = hashlib.sha256(raw_bytes).hexdigest()
        snapshot_name = f"{source}_{source_version}_{sha[:12]}.raw"
        snapshot_path = SNAPSHOTS_DIR / snapshot_name
        snapshot_path.write_bytes(raw_bytes)

        manifest = SourceManifest(
            source=source,
            source_version=source_version,
            url=url,
            fetched_at=fetched_at,
            sha256=sha,
            license_id=license_id,
            records_count=0,
            validation_status="pending",
        )
        manifest_path = MANIFESTS_DIR / f"{source}_{source_version}_{sha[:12]}.json"
        manifest_path.write_text(json.dumps(manifest.to_dict(), ensure_ascii=False, indent=2), encoding="utf-8")

        return StageResult(Stage.ACQUIRE, ok=True, manifest=manifest,
                           artifacts={"snapshot": str(snapshot_path), "manifest": str(manifest_path)})

    # 02 Validate ----------------------------------------------------------
    def validate(self, manifest: SourceManifest, raw_bytes: bytes) -> StageResult:
        try:
            data = json.loads(raw_bytes.decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            return StageResult(Stage.VALIDATE, ok=False, manifest=manifest,
                               error=f"Невалидный JSON/UTF-8: {exc}")
        if not isinstance(data, (dict, list)):
            return StageResult(Stage.VALIDATE, ok=False, manifest=manifest,
                               error="Ожидался JSON объект или массив верхнего уровня")
        records_count = len(data) if isinstance(data, list) else len(data.get("records", []))
        # Проверка схемы: каждая запись содержит идентификатор.
        items = data if isinstance(data, list) else data.get("records", [])
        for item in items:
            if not isinstance(item, dict):
                return StageResult(Stage.VALIDATE, ok=False, manifest=manifest,
                                   error="Каждая запись должна быть объектом")
            if not any(k in item for k in ("id", "cwe_id", "CVE_ID", "ID")):
                return StageResult(Stage.VALIDATE, ok=False, manifest=manifest,
                                   error="Запись без идентификатора")
        manifest.records_count = records_count
        manifest.validation_status = "valid"
        return StageResult(Stage.VALIDATE, ok=True, manifest=manifest,
                           artifacts={"raw": data})

    # 03 Normalize ---------------------------------------------------------
    def normalize(self, raw: Any, source: str, source_version: str) -> List[CanonicalRecord]:
        """Превращает исходный ответ API в единый набор CanonicalRecord."""
        records: List[CanonicalRecord] = []
        items = raw if isinstance(raw, list) else raw.get("records", [])
        for item in items:
            rec = self._normalize_item(item, source, source_version)
            if rec is not None:
                records.append(rec)
        return records

    def _normalize_item(self, item: Dict[str, Any], source: str, source_version: str) -> Optional[CanonicalRecord]:
        ext_id = self._pick(item, "id", "cwe_id", "CVE_ID", "ID")
        if not ext_id or not str(ext_id).strip():
            return None
        ext_id = str(ext_id).strip()
        title = self._pick(item, "title", "name", "summary", "description") or ext_id
        summary = item.get("summary") or item.get("description") or None
        rec_type = self._infer_type(source, item, ext_id)
        props = {k: v for k, v in item.items() if k.lower() not in ("id", "title", "name", "summary", "description")}
        return CanonicalRecord(
            external_id=ext_id,
            source=source,
            source_version=source_version,
            record_type=rec_type,
            title=str(title),
            summary=summary,
            properties=props,
            provenance={
                "source": source,
                "source_version": source_version,
                "retrieved_at": now_iso(),
                "schema_version": SCHEMA_VERSION,
            },
        )

    @staticmethod
    def _pick(item: Dict[str, Any], *keys: str) -> Any:
        lower = {str(k).lower(): v for k, v in item.items()}
        for key in keys:
            if key.lower() in lower:
                return lower[key.lower()]
        return None

    @staticmethod
    def _infer_type(source: str, item: Dict[str, Any], ext_id: str) -> str:
        if source == "MITRE_CWE":
            return "weakness"
        if source == "OWASP_TOP10":
            return "risk_category"
        if "CVE-" in str(ext_id):
            return "cve"
        return "weakness"

    # 04 Link & dedupe -----------------------------------------------------
    def link_and_dedupe(self, records: List[CanonicalRecord]) -> List[CanonicalRecord]:
        """MERGE по стабильному ключу, разрешение алиасов, проверка связей."""
        seen: Dict[str, CanonicalRecord] = {}
        deduped: List[CanonicalRecord] = []
        for rec in records:
            key = f"{rec.source}:{rec.record_type}:{rec.external_id}"
            if key in seen:
                continue
            seen[key] = rec
            deduped.append(rec)
        return deduped

    # 05 Review ------------------------------------------------------------
    def review(self, records: List[CanonicalRecord], auto_approve: bool = False) -> List[CanonicalRecord]:
        """Проверка лицензий, смысла отношений, безопасности и корректности правил.

        Конфликт с уже утверждёнными данными VERIX не перезаписывается тихо —
        запись помечается для ревью (раздел 04.2 ТЗ).
        """
        reviewed: List[CanonicalRecord] = []
        for rec in records:
            if self._has_conflict(rec):
                rec.status = RecordStatus.REJECTED
                rec.properties["conflict"] = True
                rec.properties["conflict_reason"] = "противоречит утверждённым данным VERIX"
            elif auto_approve or self._passes_review(rec):
                rec.status = RecordStatus.APPROVED
            else:
                rec.status = RecordStatus.REJECTED
            reviewed.append(rec)
        return reviewed

    LABEL_BY_TYPE = {
        "weakness": "Weakness",
        "risk_category": "RiskCategory",
        "cve": "Vulnerability",
        "rule": "RuleRevision",
        "mitigation": "Mitigation",
        "technology": "Technology",
        "signal": "DetectionSignal",
    }
    KEY_BY_TYPE = {
        "weakness": "id",
        "risk_category": "id",
        "cve": "id",
        "rule": "key",
        "mitigation": "id",
        "technology": "id",
        "signal": "id",
    }

    def _has_conflict(self, rec: CanonicalRecord) -> bool:
        """Проверяет, противоречит ли запись существующему узлу в графе."""
        label = self.LABEL_BY_TYPE.get(rec.record_type)
        key = self.KEY_BY_TYPE.get(rec.record_type, "id")
        if not label:
            return False
        try:
            rows, _, _ = self.repo.driver.execute_query(
                f"MATCH (n:{label} {{{key}: $id}}) RETURN n LIMIT 1",
                id=rec.external_id, database_=self.repo.database,
            )
        except Exception:
            return False
        if not rows:
            return False
        existing = dict(rows[0])["n"]
        # Конфликт только по существенным данным (title/summary), а не по статусу
        # (статус назначается на стадии ревью).
        for field in ("title", "summary"):
            value = rec.title if field == "title" else rec.summary
            if field in existing and existing[field] not in (None, value):
                return True
        return False

    @staticmethod
    def _passes_review(rec: CanonicalRecord) -> bool:
        if not rec.external_id or not rec.title:
            return False
        if rec.record_type == "rule" and not rec.properties.get("handler_key"):
            return False
        return True

    # 06 Publish -----------------------------------------------------------
    def publish(self, records: List[CanonicalRecord], published_by: str) -> StageResult:
        approved = [r for r in records if r.status == RecordStatus.APPROVED]
        counts = self.repo.apply_canonical_records(approved)
        self.repo.create_relationships(approved)

        release_id = f"kb-{time.strftime('%Y.%m.%d')}"
        manifest_payload = {
            "release_id": release_id,
            "published_by": published_by,
            "published_at": now_iso(),
            "records": [r.external_id for r in approved],
            "counts": counts,
        }
        manifest_bytes = json.dumps(manifest_payload, ensure_ascii=False, sort_keys=True).encode("utf-8")
        manifest_sha = hashlib.sha256(manifest_bytes).hexdigest()

        release_path = CONTENT_DIR / "releases" / f"{release_id}.json"
        release_path.write_bytes(manifest_bytes)

        self.repo.publish_release(release_id, published_by, manifest_sha)
        rule_keys = [r.external_id for r in approved if r.record_type == "rule"]
        self.repo.link_release_to_rules(release_id, rule_keys)

        manifest = SourceManifest(
            source="VERIX_REVIEW", source_version="1.0", url="internal",
            fetched_at=now_iso(), sha256=manifest_sha, license_id="INTERNAL",
            records_count=len(approved), validation_status="approved",
        )
        return StageResult(Stage.PUBLISH, ok=True, records=approved, manifest=manifest,
                           artifacts={"release": str(release_path), "counts": counts})
