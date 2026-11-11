"""Загрузчик CISA KEV JSON — признак реально наблюдаемой эксплуатации для CVE."""

from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from typing import Any, List, Optional

import httpx

from ..pipeline import CanonicalRecord, now_iso


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


class CISAKEVLoader:
    """CISA Known Exploited Vulnerabilities Catalog (CC0)."""

    DEFAULT_URL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
    LICENSE_ID = "CISA_KEV_CC0"

    def __init__(self, http: Optional[httpx.Client] = None, timeout: float = 60.0) -> None:
        self.http = http or httpx.Client(timeout=timeout)

    def fetch(self) -> SourceFetchResult:
        resp = self.http.get(self.DEFAULT_URL)
        resp.raise_for_status()
        data = resp.content
        return SourceFetchResult(
            source="CISA_KEV",
            source_version="1.0",
            url=self.DEFAULT_URL,
            fetched_at=now_iso(),
            sha256=_hash_bytes(data),
            license_id=self.LICENSE_ID,
            records_count=0,
            raw=data,
        )

    def parse(self, raw: bytes) -> List[CanonicalRecord]:
        data = json.loads(raw.decode("utf-8"))
        vulns = data.get("vulnerabilities", [])
        records: List[CanonicalRecord] = []
        for item in vulns:
            cve_id = item.get("cveID", "")
            if not cve_id:
                continue
            records.append(CanonicalRecord(
                external_id=cve_id,
                source="CISA_KEV",
                source_version="1.0",
                record_type="cve",
                title=cve_id,
                summary=item.get("shortDescription") or None,
                properties={
                    "state": "kevin_catalog",
                    "required_action": item.get("requiredAction") or None,
                    "due_date": item.get("dueDate") or None,
                    "known_ransomware": item.get("knownRansomwareCampaignUse"),
                    "cwe": item.get("cweID") or None,
                    "vendor_project": item.get("vendorProject") or None,
                    "product": item.get("product") or None,
                    "date_added": item.get("dateAdded") or None,
                },
                provenance={"source": "CISA_KEV", "source_version": "1.0"},
                relationships=[
                    {"type": "ASSOCIATED_WITH", "target_label": "Weakness", "target_id": str(item.get("cweID")), "props": {}}
                ] if item.get("cweID") and str(item.get("cweID")).startswith("CWE-") else [],
            ))
        return records
