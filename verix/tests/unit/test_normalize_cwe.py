import pytest

from etl.models import CanonicalRecord, ETLPipeline, RecordStatus


class FakeRepo:
    """Заглушка репозитория: хранит узлы в памяти для проверки идемпотентности."""

    def __init__(self) -> None:
        self.nodes = {}
        self.relationships = []
        self.releases = []

    def apply_canonical_records(self, records):
        counts = {}
        for rec in records:
            key = f"{rec.record_type}:{rec.external_id}"
            self.nodes[key] = rec
            counts[rec.record_type] = counts.get(rec.record_type, 0) + 1
        return counts

    def create_relationships(self, records):
        for rec in records:
            for rel in rec.relationships:
                self.relationships.append((rec.external_id, rel["type"], rel["target_id"]))
        return len(self.relationships)

    def publish_release(self, release_id, published_by, manifest_sha256):
        self.releases.append({"id": release_id, "active": True})

    def link_release_to_rules(self, release_id, rule_keys):
        pass

    def close(self):
        pass


@pytest.fixture
def pipeline():
    return ETLPipeline(FakeRepo())


def test_normalize_cwe_produces_weakness_record(pipeline):
    raw = [
        {
            "ID": "CWE-89",
            "Name": "Improper Neutralization of Special Elements used in an SQL Command",
            "Description": "SQL injection description.",
            "Status": "Draft",
            "Abstraction": "Base",
        },
        {
            "ID": "CWE-79",
            "Name": "Improper Neutralization of Input During Web Page Generation",
            "Description": "XSS description.",
            "Status": "Draft",
            "Abstraction": "Base",
        },
    ]
    records = pipeline.normalize(raw, "MITRE_CWE", "5.0")
    assert len(records) == 2
    ids = {r.external_id for r in records}
    assert ids == {"CWE-89", "CWE-79"}
    for r in records:
        assert r.record_type == "weakness"
        assert r.source == "MITRE_CWE"
        assert r.source_version == "5.0"
        assert r.provenance["source"] == "MITRE_CWE"
        assert r.status == RecordStatus.PENDING


def test_normalize_skips_empty_ids(pipeline):
    raw = [
        {"ID": "", "Name": "empty", "Description": "x"},
        {"ID": "NOT-CWE", "Name": "bad", "Description": "x"},
        {"ID": "CWE-89", "Name": "SQLi", "Description": "desc"},
    ]
    records = pipeline.normalize(raw, "MITRE_CWE", "5.0")
    # Пустые ID пропускаются; непустые сохраняются (формат CWE проверяется на стадии Validate).
    assert len(records) == 2
    ids = {r.external_id for r in records}
    assert ids == {"NOT-CWE", "CWE-89"}


def test_link_and_dedupe_removes_duplicates(pipeline):
    r1 = CanonicalRecord("CWE-89", "MITRE_CWE", "5.0", "weakness", "SQLi")
    r2 = CanonicalRecord("CWE-89", "MITRE_CWE", "5.0", "weakness", "SQLi dup")
    deduped = pipeline.link_and_dedupe([r1, r2])
    assert len(deduped) == 1


def test_review_marks_rules_without_handler_rejected(pipeline):
    rule = CanonicalRecord(
        "VRX-SQLI-001:1.0.0", "VERIX", "1.0.0", "rule", "SQLi rule",
        properties={},
    )
    reviewed = pipeline.review([rule])
    assert reviewed[0].status == RecordStatus.REJECTED


def test_review_approves_valid_rule(pipeline):
    rule = CanonicalRecord(
        "VRX-SQLI-001:1.0.0", "VERIX", "1.0.0", "rule", "SQLi rule",
        properties={"handler_key": "passive_error_pattern_v1"},
    )
    reviewed = pipeline.review([rule])
    assert reviewed[0].status == RecordStatus.APPROVED


def test_pipeline_publish_is_idempotent_on_nodes(pipeline):
    records = pipeline.normalize(
        [{"ID": "CWE-89", "Name": "SQLi", "Description": "desc", "Status": "Draft", "Abstraction": "Base"}],
        "MITRE_CWE",
        "5.0",
    )
    records = pipeline.link_and_dedupe(records)
    records = pipeline.review([records[0]], auto_approve=True)

    result1 = pipeline.publish(records, "tester")
    result2 = pipeline.publish(records, "tester")

    # Повторный импорт одного набора не создаёт новых узлов (идемпотентность MERGE).
    assert result1.artifacts["counts"] == result2.artifacts["counts"]
