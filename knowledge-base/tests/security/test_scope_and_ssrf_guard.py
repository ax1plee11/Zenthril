"""Безопасность: защита от SSRF и проверка scope при активном тестировании (раздел 10)."""

import ipaddress
import re

# Разрешённые схемы и хосты для активного сканирования (пример политики).
ALLOWED_SCHEMES = {"http", "https"}
BLOCKED_HOST_PATTERNS = [
    r"^localhost$",
    r"^127\.",
    r"^0\.0\.0\.0$",
    r"^::1$",
    r"^169\.254\.",        # link-local
    r"^10\.",              # private
    r"^172\.(1[6-9]|2[0-9]|3[01])\.",  # private
    r"^192\.168\.",        # private
    r"metadata",           # cloud metadata
    r"internal",
]


def is_allowed_target(url: str, consent_id: str | None = None) -> bool:
    if not consent_id:
        return False
    try:
        from urllib.parse import urlparse
    except ImportError:
        from urlparse import urlparse
    parsed = urlparse(url)
    if parsed.scheme.lower() not in ALLOWED_SCHEMES:
        return False
    host = parsed.hostname or ""
    if any(re.search(p, host, re.IGNORECASE) for p in BLOCKED_HOST_PATTERNS):
        return False
    try:
        ip = ipaddress.ip_address(host)
        if ip.is_private or ip.is_loopback or ip.is_link_local or ip.is_multicast:
            return False
    except ValueError:
        pass  # доменное имя — допустимо, если не в блок-листе
    return True


def test_localhost_blocked():
    assert not is_allowed_target("http://localhost:8080/scan", consent_id="c1")
    assert not is_allowed_target("http://127.0.0.1/scan", consent_id="c1")


def test_metadata_blocked():
    assert not is_allowed_target("http://169.254.169.254/latest/meta-data/", consent_id="c1")


def test_private_ip_blocked():
    assert not is_allowed_target("http://10.0.0.1/scan", consent_id="c1")


def test_allowed_public_target():
    assert is_allowed_target("https://example.com/scan", consent_id="c1")


def test_no_consent_rejected():
    assert not is_allowed_target("https://example.com/scan", consent_id=None)
