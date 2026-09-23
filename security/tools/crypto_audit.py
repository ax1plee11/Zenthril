#!/usr/bin/env python3
"""
Zenthril Crypto Audit Tool
Performs automated cryptographic weakness scanning across the codebase.
Based on Anthropic Cybersecurity Skills: performing-cryptographic-audit-of-application
"""

import os
import re
import sys
from dataclasses import dataclass
from enum import Enum
from pathlib import Path


class Severity(Enum):
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"
    INFO = "INFO"


@dataclass
class Finding:
    severity: Severity
    category: str
    description: str
    file_path: str
    line_number: int
    remediation: str


class CryptoAuditor:
    def __init__(self, root_path: str):
        self.root = Path(root_path)
        self.findings: list[Finding] = []
        self.scanned_files = 0

    def scan(self) -> list[Finding]:
        self._scan_weak_algorithms()
        self._scan_hardcoded_secrets()
        self._scan_insecure_modes()
        self._scan_weak_kdf()
        self._scan_poor_entropy()
        self._scan_deprecated_protocols()
        self._scan_missing_auth()
        self._scan_crypto_misuse()
        return self.findings

    def _scan_weak_algorithms(self):
        patterns = {
            r'\bMD5\b': (Severity.HIGH, "Weak Hashing", "MD5 is cryptographically broken"),
            r'\bSHA-?1\b': (Severity.HIGH, "Weak Hashing", "SHA-1 is deprecated for security use"),
            r'\bDES\b': (Severity.CRITICAL, "Insecure Encryption", "DES is broken"),
            r'\b3DES\b': (Severity.HIGH, "Insecure Encryption", "3DES is deprecated"),
            r'\bRC4\b': (Severity.CRITICAL, "Insecure Encryption", "RC4 is broken"),
            r'\bBlowfish\b': (Severity.MEDIUM, "Insecure Encryption", "Blowfish has 64-bit block size"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _scan_hardcoded_secrets(self):
        patterns = [
            (r'(password|passwd|pwd)\s*=\s*["\'][^"\']{8,}["\']', Severity.CRITICAL, "Hardcoded Secret", "Hardcoded password detected"),
            (r'(secret|api_key|apikey)\s*=\s*["\'][^"\']{16,}["\']', Severity.CRITICAL, "Hardcoded Secret", "Hardcoded API key detected"),
            (r'(jwt_secret|JWT_SECRET)\s*=\s*["\'][^"\']{8,}["\']', Severity.CRITICAL, "Hardcoded Secret", "Hardcoded JWT secret detected"),
        ]
        for pattern, severity, category, desc in patterns:
            self._search_pattern(pattern, severity, category, desc, exclude_tests=True)

    def _scan_insecure_modes(self):
        patterns = {
            r'\bECB\b': (Severity.HIGH, "Bad Cipher Mode", "ECB mode reveals patterns"),
            r'\bCBC\b': (Severity.MEDIUM, "Cipher Mode", "CBC mode requires proper IV handling"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _scan_weak_kdf(self):
        patterns = {
            r'PBKDF2.*iterations?\s*=\s*[0-9]{1,4999}': (Severity.HIGH, "Weak KDF", "PBKDF2 with < 10000 iterations is weak"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _scan_poor_entropy(self):
        patterns = {
            r'\bMath\.random\b': (Severity.HIGH, "Poor Entropy", "Math.random is not cryptographically secure"),
            r'\brandom\.random\b': (Severity.HIGH, "Poor Entropy", "random.random is not cryptographically secure"),
            r'\btime\(\)': (Severity.MEDIUM, "Poor Entropy", "time() based values are predictable"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _scan_deprecated_protocols(self):
        patterns = {
            r'\bSSLv3\b': (Severity.CRITICAL, "Deprecated Protocol", "SSLv3 is broken"),
            r'\bTLSv?1\.0\b': (Severity.HIGH, "Deprecated Protocol", "TLS 1.0 is deprecated"),
            r'\bTLSv?1\.1\b': (Severity.HIGH, "Deprecated Protocol", "TLS 1.1 is deprecated"),
            r'ws://': (Severity.MEDIUM, "Insecure Transport", "Unencrypted WebSocket (ws://) detected"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _scan_missing_auth(self):
        ws_patterns = [
            (r'func.*ServeWS', Severity.HIGH, "Missing Auth", "WebSocket handler; verify authentication is enforced"),
            (r'NoopSessionValidator', Severity.CRITICAL, "Missing Auth", "NoopSessionValidator must not reach production"),
        ]
        for pattern, severity, category, desc in ws_patterns:
            self._search_pattern(pattern, severity, category, desc, extensions=[".go"])

    def _scan_crypto_misuse(self):
        patterns = {
            r'AES.*ECB': (Severity.HIGH, "Crypto Misuse", "AES-ECB is insecure"),
        }
        for pattern, (severity, category, desc) in patterns.items():
            self._search_pattern(pattern, severity, category, desc)

    def _search_pattern(self, pattern: str, severity: Severity, category: str, description: str,
                        extensions: list[str] | None = None, exclude_tests: bool = False):
        regex = re.compile(pattern, re.IGNORECASE)
        for file_path in self.root.rglob("*"):
            if not file_path.is_file():
                continue
            if extensions and file_path.suffix not in extensions:
                continue
            if exclude_tests and ("test" in file_path.name or "spec" in file_path.name):
                continue
            if file_path.suffix not in {".ts", ".tsx", ".go", ".js", ".json", ".yaml", ".yml", ".env", ".md"}:
                continue
            try:
                content = file_path.read_text(encoding="utf-8", errors="ignore")
            except Exception:
                continue
            if not content.strip():
                continue
            for match in regex.finditer(content):
                line_no = content[:match.start()].count("\n") + 1
                context = content[max(0, match.start() - 40):match.end() + 40].replace("\n", " ")
                self.findings.append(Finding(
                    severity=severity,
                    category=category,
                    description=f"{description} (context: ...{context}...)",
                    file_path=str(file_path.relative_to(self.root)),
                    line_number=line_no,
                    remediation=f"Review and replace insecure pattern with a secure alternative",
                ))
            self.scanned_files += 1

    def print_report(self):
        critical = [f for f in self.findings if f.severity == Severity.CRITICAL]
        high = [f for f in self.findings if f.severity == Severity.HIGH]
        medium = [f for f in self.findings if f.severity == Severity.MEDIUM]
        low = [f for f in self.findings if f.severity == Severity.LOW]

        print(f"\n=== Zenthril Crypto Audit Report ===")
        print(f"Scanned files: {self.scanned_files}")
        print(f"Total findings: {len(self.findings)}")
        print(f"  CRITICAL: {len(critical)}")
        print(f"  HIGH: {len(high)}")
        print(f"  MEDIUM: {len(medium)}")
        print(f"  LOW: {len(low)}\n")

        for severity in [Severity.CRITICAL, Severity.HIGH, Severity.MEDIUM, Severity.LOW]:
            items = [f for f in self.findings if f.severity == severity]
            if not items:
                continue
            print(f"--- {severity.value} ---")
            for finding in items:
                print(f"[{finding.category}] {finding.file_path}:{finding.line_number}")
                print(f"  {finding.description}")
                print(f"  Remediation: {finding.remediation}\n")


def main():
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    auditor = CryptoAuditor(project_root)
    findings = auditor.scan()
    auditor.print_report()

    critical_and_high = [f for f in findings if f.severity in (Severity.CRITICAL, Severity.HIGH)]
    if critical_and_high:
        print(f"\n[FAIL] Found {len(critical_and_high)} CRITICAL/HIGH findings")
        sys.exit(1)
    print("\n[PASS] No CRITICAL/HIGH crypto weaknesses detected")
    sys.exit(0)


if __name__ == "__main__":
    main()
