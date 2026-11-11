"""Интеграционные тесты с заглушкой Neo4j: идемпотентность MERGE и неизменяемость релиза."""

from unittest.mock import MagicMock

from etl.models import CanonicalRecord, ETLPipeline, KnowledgeRepository


class FakeResult:
    def __init__(self, data):
        self._data = data
        self.counters = MagicMock(relationships_created=len(data))

    def data(self):
        return self._data

    def consume(self):
        return self


class FakeSession:
    def __init__(self):
        self.writes = []

    def execute_write(self, fn, *args, **kwargs):
        tx = MagicMock()
        tx.run = MagicMock(return_value=FakeResult([]))
        result = fn(tx, *args, **kwargs)
        self.writes.append(fn.__name__)
        return result

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False


class FakeDriver:
    def __init__(self):
        self.session_calls = []
        self._session = FakeSession()

    def session(self, database=None):
        self.session_calls.append(database)
        return self._session

    def execute_query(self, query, **kwargs):
        return ([], MagicMock(), MagicMock())

    def close(self):
        pass


def test_merge_idempotent_node_count():
    repo = KnowledgeRepository.__new__(KnowledgeRepository)
    repo.driver = FakeDriver()
    repo.database = "neo4j"

    rec = CanonicalRecord("CWE-89", "MITRE_CWE", "5.0", "weakness", "SQLi")

    c1 = repo.apply_canonical_records([rec])
    c2 = repo.apply_canonical_records([rec])

    # Повторный импорт той же записи не меняет число узлов (идемпотентность MERGE).
    assert c1 == c2 == {"weakness": 1}


def test_publish_sets_active_release_once():
    repo = KnowledgeRepository.__new__(KnowledgeRepository)
    repo.driver = FakeDriver()
    repo.database = "neo4j"

    session = repo.driver.session()
    repo.publish_release("kb-2026.11.05", "appsec", "sha256:abc")
    tx_writes = session.writes
    assert "_publish_release_tx" in tx_writes
