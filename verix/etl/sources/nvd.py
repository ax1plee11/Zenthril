"""Загрузчик NVD CVE API 2.0 — релевантные CVE, CVSS, CWE, версии, modified."""

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


class NVDLoader:
    """NVD Vulnerabilities API 2.0.

    MVP: загружает только релевантные CVE (инкрементально по modified). В релизе
    достаточно начальной выборки; полный импорт всех CVE не входит в MVP.
    """

    DEFAULT_URL = "https://services.nvd.nist.gov/rest/json/cves/2.0"
    LICENSE_ID = "NVD_TERMS_VERIFY"

    def __init__(self, http: Optional[httpx.Client] = None, timeout: float = 60.0) -> None:
        self.http = http or httpx.Client(timeout=timeout)

    def fetch(self, results_per_page: int = 100, start_index: int = 0) -> SourceFetchResult:
        params = {"resultsPerPage": results_per_page, "startIndex": start_index}
        resp = self.http.get(self.DEFAULT_URL, params=params)
        resp.raise_for_status()
        data = resp.content
        return SourceFetchResult(
            source="NVD_CVE",
            source_version="2.0",
            url=str(resp.url),
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
            cve = item.get("cve", {})
            cve_id = cve.get("id", "")
            if not cve_id:
                continue
            # CVSS v3.1
            metrics = cve.get("metrics", {})
            cvss_v31 = metrics.get("cvssMetricV31", [{}])[0] if metrics.get("cvssMetricV31") else {}
            cvss_data = cvss_v31.get("cvssData", {})
            published = cve.get("published")
            modified = cve.get("lastModified")
            state = cve.get("vulnStatus")
            # CWE из слабостей
            weaknesses = cve.get("weaknesses", [])
            cwe_ids = []
            for w in weaknesses:
                for desc in w.get("description", []):
                    val = desc.get("value", "")
                    if val.startswith("CWE-"):
                        cwe_ids.append(val)
            records.append(CanonicalRecord(
                external_id=cve_id,
                source="NVD_CVE",
                source_version="2.0",
                record_type="cve",
                title=cve_id,
                summary=None,
                properties={
                    "published_at": published,
                    "modified_at": modified,
                    "cvss": cvss_data.get("baseScore"),
                    "cvss_version": cvss_data.get("version"),
                    "state": state,
                    "cwe_ids": cwe_ids,
                },
                provenance={"source": "NVD_CVE", "source_version": "2.0"},
                relationships=[
                    {"type": "ASSOCIATED_WITH", "target_label": "Weakness", "target_id": cwe, "props": {}}
                    for cwe in cwe_ids
                ],
            ))
        return records
