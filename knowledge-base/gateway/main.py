"""FastAPI KB Gateway — read-only шлюз к графе знаний VERIX (раздел 08 ТЗ).

Контракт:
GET  /internal/kb/releases/active   — текущий опубликованный релиз + checksum
GET  /internal/kb/rules?cwe=...     — одобренные версии правил для CWE
GET  /internal/kb/weaknesses/{id}   — классификация и источники для объяснения
GET  /internal/kb/remediations?cwe= — контекстные рекомендации
POST /internal/kb/ingest/runs       — запуск ручного ETL (админ, роль-ограниченный)
POST /internal/kb/releases/{id}/publish — атомарное утверждение релиза

Сканер не знает Cypher; весь доступ через сервисный слой.
"""

from __future__ import annotations

import os
from typing import Any, Dict, List, Optional

from fastapi import FastAPI, Header, HTTPException, Query, status
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from etl.models import KnowledgeRepository


app = FastAPI(title="VERIX KB Gateway", version="1.0.0")


@app.get("/")
def health() -> Dict[str, str]:
    return {"service": "VERIX KB Gateway", "version": "1.0.0", "status": "ok"}


def _env(key: str, default: Optional[str] = None) -> Optional[str]:
    return os.environ.get(key, default)


def _repo() -> KnowledgeRepository:
    uri = _env("NEO4J_URI")
    user = _env("NEO4J_USER")
    password = _env("NEO4J_PASSWORD")
    if not uri or not user or not password:
        raise HTTPException(status_code=503, detail="Neo4j не сконфигурирован")
    return KnowledgeRepository(uri, user, password, _env("NEO4J_DATABASE", "neo4j"))


KB_ADMIN_TOKEN = _env("KB_ADMIN_TOKEN", "")


def _require_admin(authorization: Optional[str]) -> None:
    if not KB_ADMIN_TOKEN or authorization != f"Bearer {KB_ADMIN_TOKEN}":
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="требуется админ-токен")


# --- Schemas ---------------------------------------------------------------

class ReleaseInfo(BaseModel):
    id: str
    published_at: Optional[str] = None
    approved_by: Optional[str] = None
    manifest_sha256: Optional[str] = None
    active: bool = True


class RuleRevisionOut(BaseModel):
    key: str
    handler_key: str
    status: str
    safety_class: str


class WeaknessOut(BaseModel):
    id: str
    title: Optional[str]
    summary: Optional[str]
    status: Optional[str]
    source_version: Optional[str]
    categories: List[str] = []
    related_cves: List[str] = []
    source_urls: List[str] = []


class RemediationOut(BaseModel):
    id: str
    title: Optional[str]
    recommendation: Optional[str]
    language: Optional[str]
    tech_scope: Optional[str]


# --- Endpoints -------------------------------------------------------------

@app.get("/internal/kb/releases/active", response_model=ReleaseInfo)
def get_active_release() -> ReleaseInfo:
    repo = _repo()
    try:
        with repo.driver.session(database=repo.database) as session:
            rows = session.run(
                """
                MATCH (rel:KnowledgeRelease {active: true})
                RETURN rel.id AS id, rel.published_at AS published_at,
                       rel.approved_by AS approved_by, rel.manifest_sha256 AS manifest_sha256
                ORDER BY rel.published_at DESC LIMIT 1
                """
            ).data()
        if not rows:
            raise HTTPException(status_code=404, detail="опубликованный релиз не найден")
        row = rows[0]
        return ReleaseInfo(
            id=row["id"],
            published_at=str(row["published_at"]) if row["published_at"] else None,
            approved_by=row["approved_by"],
            manifest_sha256=row["manifest_sha256"],
            active=True,
        )
    finally:
        repo.close()


@app.get("/internal/kb/rules", response_model=List[RuleRevisionOut])
def get_rules(cwe: str = Query(..., description="CWE ID, например CWE-89")) -> List[RuleRevisionOut]:
    repo = _repo()
    try:
        records, _, _ = repo.driver.execute_query(
            """
            MATCH (r:RuleRevision)-[:DETECTS]->(w:Weakness {id: $cwe_id})
            WHERE r.status = 'approved'
            RETURN r.key AS key, r.handler_key AS handler_key, r.status AS status,
                   r.safety_class AS safety_class
            ORDER BY r.key
            """,
            cwe_id=cwe,
            database_=repo.database,
        )
        return [RuleRevisionOut(**dict(r)) for r in records]
    finally:
        repo.close()


@app.get("/internal/kb/weaknesses/{weakness_id}", response_model=WeaknessOut)
def get_weakness(weakness_id: str) -> WeaknessOut:
    repo = _repo()
    try:
        rows = repo.driver.execute_query(
            """
            MATCH (w:Weakness {id: $id})
            OPTIONAL MATCH (w)-[:IN_CATEGORY]->(c:RiskCategory)
            OPTIONAL MATCH (v:Vulnerability)-[:ASSOCIATED_WITH]->(w)
            OPTIONAL MATCH (w)<-[:SUPPORTS]-(s:SourceRecord)
            RETURN w.id AS id, w.title AS title, w.summary AS summary, w.status AS status,
                   w.source_version AS source_version,
                   collect(DISTINCT c.id) AS categories,
                   collect(DISTINCT v.id) AS related_cves,
                   collect(DISTINCT s.url) AS source_urls
            """,
            id=weakness_id,
            database_=repo.database,
        ).records
        if not rows:
            raise HTTPException(status_code=404, detail="уязвимость не найдена")
        row = dict(rows[0])
        return WeaknessOut(
            id=row["id"],
            title=row["title"],
            summary=row["summary"],
            status=row["status"],
            source_version=row["source_version"],
            categories=[x for x in (row["categories"] or []) if x],
            related_cves=[x for x in (row["related_cves"] or []) if x],
            source_urls=[x for x in (row["source_urls"] or []) if x],
        )
    finally:
        repo.close()


@app.get("/internal/kb/remediations", response_model=List[RemediationOut])
def get_remediations(cwe: str = Query(..., description="CWE ID, например CWE-89")) -> List[RemediationOut]:
    repo = _repo()
    try:
        rows = repo.driver.execute_query(
            """
            MATCH (w:Weakness {id: $cwe})<-[:ADDRESSES]-(m:Mitigation)
            RETURN m.id AS id, m.title AS title, m.recommendation AS recommendation,
                   m.language AS language, m.tech_scope AS tech_scope
            ORDER BY m.id
            """,
            cwe=cwe,
            database_=repo.database,
        ).records
        return [RemediationOut(**dict(r)) for r in rows]
    finally:
        repo.close()


@app.post("/internal/kb/ingest/runs", status_code=status.HTTP_202_ACCEPTED)
def run_ingest(authorization: Optional[str] = Header(None)) -> Dict[str, Any]:
    _require_admin(authorization)
    # В реальном сервисе запускается фоновая задача ETL; здесь возвращаем подтверждение.
    return {"status": "accepted", "run_id": f"ingest-{__import__('datetime').datetime.utcnow().strftime('%Y%m%d%H%M%S')}"}


@app.post("/internal/kb/releases/{release_id}/publish", status_code=status.HTTP_200_OK)
def publish_release(release_id: str, authorization: Optional[str] = Header(None)) -> Dict[str, Any]:
    _require_admin(authorization)
    repo = _repo()
    try:
        published_by = "kb-admin"
        manifest_sha = __import__("hashlib").sha256(release_id.encode()).hexdigest()
        repo.publish_release(release_id, published_by, manifest_sha)
        return {"status": "published", "release_id": release_id, "manifest_sha256": manifest_sha}
    finally:
        repo.close()

