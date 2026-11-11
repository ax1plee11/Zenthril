"""Интеграция Scanner Engine с KB Gateway (раздел 08.3 ТЗ).

Сканер не переписывается: он получает список применимых approved RuleRevision
через интерфейс Knowledge Base Gateway, выбирает зарегистрированный handler по
handler_key, и в каждый Finding в PostgreSQL фиксирует rule_revision_key,
kb_release_id, cwe_id, evidence_ref и confidence.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

import httpx


@dataclass
class Finding:
    rule_revision_key: str
    kb_release_id: str
    cwe_id: str
    evidence_ref: str
    confidence: str
    target: str


class ScannerAdapter:
    """Адаптер сканера к KB Gateway. Читает только одобренные правила."""

    def __init__(self, gateway_url: str, operational_token: str,
                 registered_handlers: Dict[str, Any], kb_release_id: str) -> None:
        self.gateway_url = gateway_url.rstrip("/")
        self.token = operational_token
        self.registered_handlers = registered_handlers
        self.kb_release_id = kb_release_id
        self.client = httpx.Client(timeout=10.0)

    def applicable_rules(self, cwe_id: str) -> List[Dict[str, Any]]:
        resp = self.client.get(
            f"{self.gateway_url}/internal/kb/rules",
            params={"cwe": cwe_id},
            headers={"Authorization": f"Bearer {self.token}"},
        )
        resp.raise_for_status()
        rules = resp.json()
        # Неизвестные handler-и игнорируются с диагностикой (не исполняются).
        return [r for r in rules if r["handler_key"] in self.registered_handlers]

    def scan(self, cwe_id: str, target: str) -> List[Finding]:
        findings: List[Finding] = []
        for rule in self.applicable_rules(cwe_id):
            handler_key = rule["handler_key"]
            # Имитация выполнения handler-а.
            signal = self._run_handler(handler_key, target)
            if signal:
                findings.append(Finding(
                    rule_revision_key=rule["key"],
                    kb_release_id=self.kb_release_id,
                    cwe_id=cwe_id,
                    evidence_ref=f"{target}:{signal}",
                    confidence="suspected",
                ))
        return findings

    def _run_handler(self, handler_key: str, target: str) -> Optional[str]:
        """В реальном сканере здесь выполняется встроенный handler."""
        return None  # нет сигнала → Finding не создаётся
