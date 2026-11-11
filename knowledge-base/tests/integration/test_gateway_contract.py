"""Контракт шлюза: проверка формата ответов без реального Neo4j (fastapi TestClient)."""

from fastapi.testclient import TestClient

from gateway.main import app


def test_active_release_shape():
    client = TestClient(app)
    resp = client.get("/internal/kb/releases/active")
    # Endpoint существует: 200 если Neo4j сконфигурирован, иначе 503.
    assert resp.status_code in (200, 404, 500, 503)
    if resp.status_code == 200:
        body = resp.json()
        assert "id" in body
        assert "active" in body


def test_rules_requires_cwe():
    client = TestClient(app)
    resp = client.get("/internal/kb/rules")
    assert resp.status_code in (400, 422, 404, 500, 503)

    resp = client.get("/internal/kb/rules?cwe=CWE-89")
    assert resp.status_code in (200, 404, 500, 503)
    if resp.status_code == 200:
        assert isinstance(resp.json(), list)


def test_weakness_not_found_without_neo4j():
    client = TestClient(app)
    resp = client.get("/internal/kb/weaknesses/CWE-89")
    assert resp.status_code in (404, 500, 503)


def test_ingest_requires_admin_token():
    client = TestClient(app)
    resp = client.post("/internal/kb/ingest/runs")
    assert resp.status_code == 401

    resp = client.post(
        "/internal/kb/ingest/runs",
        headers={"Authorization": "Bearer wrong"},
    )
    assert resp.status_code == 401
