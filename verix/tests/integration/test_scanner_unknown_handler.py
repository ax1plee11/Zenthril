"""Сканер не исполняет неизвестные handler (раздел 11.3 тест integration/test_scanner_unknown_handler.py)."""

from __future__ import annotations

from typing import Any, Dict, List

# Зарегистрированные handler-и сканера (allowlist).
REGISTERED_HANDLERS: Dict[str, Any] = {
    "passive_error_pattern_v1": {"mode": "passive", "safety_class": "passive"},
    "active_login_probe_v1": {"mode": "active", "safety_class": "low-impact"},
}


class ScannerEngine:
    """Упрощённая модель Scanner Engine: выбирает handler по handler_key."""

    def __init__(self, registered_handlers: Dict[str, Any]) -> None:
        self.registered = registered_handlers
        self.executed: List[str] = []
        self.diagnostics: List[str] = []

    def run_rule(self, handler_key: str, cwe_id: str) -> Dict[str, Any]:
        if handler_key not in self.registered:
            self.diagnostics.append(
                f"неизвестный обработчик {handler_key} для {cwe_id} — игнорируется, не исполняется"
            )
            return {"skipped": True, "reason": "unknown_handler"}
        self.executed.append(handler_key)
        return {"skipped": False, "handler": handler_key}


def test_scanner_unknown_handler_ignored():
    engine = ScannerEngine(REGISTERED_HANDLERS)
    # Известный handler исполняется.
    res = engine.run_rule("passive_error_pattern_v1", "CWE-89")
    assert res["skipped"] is False
    # Неизвестный handler игнорируется с диагностикой, НЕ исполняется.
    res = engine.run_rule("nonexistent_handler_v9", "CWE-79")
    assert res["skipped"] is True
    assert any("неизвестный обработчик" in d for d in engine.diagnostics)
    assert "nonexistent_handler_v9" not in engine.executed
