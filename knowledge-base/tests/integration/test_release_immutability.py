"""Неизменяемость опубликованного релиза: старый релиз не меняется после нового."""

from unittest.mock import MagicMock

from etl.models import KnowledgeRepository


class RecordingTx:
    def __init__(self):
        self.calls = []

    def run(self, cypher, **params):
        self.calls.append((cypher, params))
        return MagicMock()


class RecordingSession:
    def __init__(self):
        self.tx = RecordingTx()

    def execute_write(self, fn, *args, **kwargs):
        return fn(self.tx, *args, **kwargs)

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False


class FakeDriver:
    def __init__(self):
        self._session = RecordingSession()

    def session(self, database=None):
        return self._session

    def close(self):
        pass


def test_publish_deactivates_old_release():
    repo = KnowledgeRepository.__new__(KnowledgeRepository)
    repo.driver = FakeDriver()
    repo.database = "neo4j"

    session = repo.driver.session()
    repo.publish_release("kb-2026.11.01", "appsec", "sha-old")
    tx = session.tx

    # Проверяем, что есть операция деактивации прошлого релиза (не удаления).
    cyphers = [c for c, _ in tx.calls]
    assert any("old.active = false" in c for c in cyphers)
    assert any("active = true" in c for c in cyphers)
    assert not any("DETACH DELETE" in c or "DELETE" in c for c in cyphers)
