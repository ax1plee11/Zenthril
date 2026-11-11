"""Загрузчик FIRST EPSS CSV — вероятность эксплуатации и percentile по CVE."""

from __future__ import annotations

import csv
import hashlib
import io
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

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


class EPSSLoader:
    """FIRST EPSS bulk CSV. Поля: cve, epss, percentile."""

    DEFAULT_URL = "https://epss.cyentia.com/epss_scores-current.csv.gz"
    LICENSE_ID = "FIRST_EPSS_ATTRIBUTION"

    def __init__(self, http: Optional[httpx.Client] = None, timeout: float = 120.0) -> None:
        self.http = http or httpx.Client(timeout=timeout)

    def fetch(self) -> SourceFetchResult:
        resp = self.http.get(self.DEFAULT_URL)
        resp.raise_for_status()
        data = resp.content
        return SourceFetchResult(
            source="FIRST_EPSS",
            source_version="current",
            url=self.DEFAULT_URL,
            fetched_at=now_iso(),
            sha256=_hash_bytes(data),
            license_id=self.LICENSE_ID,
            records_count=0,
            raw=data,
        )

    def parse(self, raw: bytes) -> List[CanonicalRecord]:
        # CSV может быть gzip-сжатым; пробуем обычный текст, иначе gzip.
        import gzip
        try:
            text = raw.decode("utf-8")
        except UnicodeDecodeError:
            text = gzip.decompress(raw).decode("utf-8")
        reader = csv.DictReader(io.StringIO(text))
        records: List[CanonicalRecord] = []
        for row in reader:
            cve = (row.get("cve") or "").strip()
            if not cve or not cve.startswith("CVE-"):
                continue
            epss = row.get("epss")
            percentile = row.get("percentile")
            records.append(CanonicalRecord(
                external_id=cve,
                source="FIRST_EPSS",
                source_version="current",
                record_type="cve",
                title=cve,
                summary=None,
                properties={
                    "epss": float(epss) if epss else None,
                    "epss_percentile": float(percentile) if percentile else None,
                },
                provenance={"source": "FIRST_EPSS", "source_version": "current"},
            ))
        return records
