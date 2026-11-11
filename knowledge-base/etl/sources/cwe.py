"""Загрузчики официальных источников (раздел 03 ТЗ): MITRE CWE и OWASP Top 10:2025."""

from __future__ import annotations

import csv
import hashlib
import io
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

import httpx

from ..pipeline import CanonicalRecord, SourceManifest, sha256_file, now_iso


@dataclass
class SourceFetchResult:
    source: str
    source_version: str
    url: str
    fetched_at: str
    sha256: str
    license_id: str
    records_count: int
    raw: bytes


def _hash_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


class CWELoader:
    """MITRE CWE 5.0 — ID, имена, описание, отношения Parent/Child, состояния, ссылки."""

    DEFAULT_URL = "https://cwe.mitre.org/data/xml/cwec_v5.0.xml.zip"
    LICENSE_ID = "MITRE_CWE_TERMS_VERIFY"

    def __init__(self, http: Optional[httpx.Client] = None, timeout: float = 30.0) -> None:
        self.http = http or httpx.Client(timeout=timeout)

    def fetch(self) -> SourceFetchResult:
        # MVP: используем официальный CSV дамп CWE (более простой для парсинга, чем XML zip).
        url = "https://cwe.mitre.org/data/csv/5.0/cwec_v5.0.csv"
        resp = self.http.get(url)
        resp.raise_for_status()
        data = resp.content
        return SourceFetchResult(
            source="MITRE_CWE",
            source_version="5.0",
            url=url,
            fetched_at=now_iso(),
            sha256=_hash_bytes(data),
            license_id=self.LICENSE_ID,
            records_count=0,
            raw=data,
        )

    def parse(self, raw: bytes) -> List[CanonicalRecord]:
        text = raw.decode("utf-8-sig")
        reader = csv.DictReader(io.StringIO(text))
        records: List[CanonicalRecord] = []
        for row in reader:
            cwe_id = (row.get("ID") or "").strip()
            if not cwe_id or not cwe_id.startswith("CWE-"):
                continue
            name = (row.get("Name") or "").strip()
            desc = (row.get("Description") or "").strip()
            status = (row.get("Status") or "").strip()
            abstraction = (row.get("Abstraction") or "").strip()
            records.append(CanonicalRecord(
                external_id=cwe_id,
                source="MITRE_CWE",
                source_version="5.0",
                record_type="weakness",
                title=name or cwe_id,
                summary=desc or None,
                properties={
                    "status": status,
                    "abstraction": abstraction,
                    "source_version": "5.0",
                },
                provenance={"source": "MITRE_CWE", "source_version": "5.0"},
            ))
        return records


class OWASPLoader:
    """OWASP Top 10:2025 — категории рисков и официальные CWE-сопоставления."""

    DEFAULT_URL = "https://raw.githubusercontent.com/OWASP/Top10/2025/data/owasp-top-10-2025.json"
    LICENSE_ID = "OWASP_FREE_AND_OPEN"

    def __init__(self, http: Optional[httpx.Client] = None, timeout: float = 30.0) -> None:
        self.http = http or httpx.Client(timeout=timeout)

    def fetch(self) -> SourceFetchResult:
        url = self.DEFAULT_URL
        resp = self.http.get(url)
        resp.raise_for_status()
        data = resp.content
        return SourceFetchResult(
            source="OWASP_TOP10",
            source_version="2025",
            url=url,
            fetched_at=now_iso(),
            sha256=_hash_bytes(data),
            license_id=self.LICENSE_ID,
            records_count=0,
            raw=data,
        )

    def parse(self, raw: bytes) -> List[CanonicalRecord]:
        import json
        data = json.loads(raw.decode("utf-8"))
        categories = data if isinstance(data, list) else data.get("categories", [])
        records: List[CanonicalRecord] = []
        for cat in categories:
            cat_id = str(cat.get("id") or cat.get("code") or "")
            if not cat_id:
                continue
            name = str(cat.get("name") or cat.get("title") or cat_id)
            desc = cat.get("description") or None
            cwe_mapping = cat.get("cwe_mapping") or cat.get("cwe") or []
            records.append(CanonicalRecord(
                external_id=cat_id,
                source="OWASP_TOP10",
                source_version="2025",
                record_type="risk_category",
                title=name,
                summary=desc,
                properties={
                    "edition": "2025",
                    "source_url": "https://top10.owasp.org/2025/",
                    "cwe_mapping": cwe_mapping,
                },
                provenance={"source": "OWASP_TOP10", "source_version": "2025"},
                relationships=[
                    {"type": "IN_CATEGORY", "target_label": "Weakness", "target_id": str(cwe), "props": {"edition": "2025"}}
                    for cwe in cwe_mapping if isinstance(cwe, str) and cwe.startswith("CWE-")
                ],
            ))
        return records
