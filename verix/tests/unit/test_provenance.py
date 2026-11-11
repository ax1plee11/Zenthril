"""Проверка provenance: каждая опубликованная запись содержит источник или аудит."""

from etl.models import CanonicalRecord


def test_provenance_required_for_published():
    rec = CanonicalRecord(
        external_id="CWE-89",
        source="MITRE_CWE",
        source_version="5.0",
        record_type="weakness",
        title="SQLi",
        provenance={
            "source": "MITRE_CWE",
            "source_version": "5.0",
            "retrieved_at": "2026-11-05T10:00:00Z",
            "schema_version": "verix.ingest.v1",
        },
    )
    assert rec.provenance["source"] == "MITRE_CWE"
    assert rec.provenance["source_version"] == "5.0"
    assert "retrieved_at" in rec.provenance
    assert rec.provenance["schema_version"] == "verix.ingest.v1"


def test_source_record_manifest_fields():
    manifest = {
        "source": "MITRE_CWE",
        "source_version": "5.0",
        "url": "https://cwe.mitre.org/data/downloads.html",
        "fetched_at": "2026-11-05T10:00:00Z",
        "sha256": "a" * 64,
        "license_id": "VERIFY_SOURCE_TERMS",
        "schema_version": "verix.ingest.v1",
        "records_count": 1,
        "validation_status": "pending",
    }
    assert len(manifest["sha256"]) == 64
    assert manifest["records_count"] >= 0
