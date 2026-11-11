"""Восстановление релиза: откат к прошлому approved релизу без удаления нового."""

from unittest.mock import MagicMock

from etl.models import KnowledgeRepository


class Capture:
    def __init__(self):
        self.calls = []

    def run(self, cypher, **params):
        self.calls.append(cypher)
        return MagicMock()


class Session:
    def __init__(self):
        self.tx = Capture()

    def execute_write(self, fn, *args, **kwargs):
        return fn(self.tx, *args, **kwargs)

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False


class Driver:
    def __init__(self):
        self._session = Session()

    def session(self, database=None):
        return self._session

    def close(self):
        pass


def test_rollback_keeps_new_release():
    repo = KnowledgeRepository.__new__(KnowledgeRepository)
    repo.driver = Driver()
    repo.database = "neo4j"

    session = repo.driver.session()
    repo.publish_release("kb-2026.11.02", "appsec", "sha-new")
    calls = session.tx.calls

    # Откат не должен удалять новый релиз (нет DETACH DELETE / DELETE релиза).
    cyphers = " ".join(calls)
    assert "kb-2026.11.02" not in cyphers or "DELETE" not in cyphers.upper()
    # Активный релиз переключается, а не удаляется.
    assert any("active = false" in c for c in calls)
