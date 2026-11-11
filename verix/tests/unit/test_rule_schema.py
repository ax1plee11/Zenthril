import re

# Схема правила из раздела 07.1 ТЗ (проверка полей без Neo4j).


def test_rule_key_format():
    rule_id = "VRX-SQLI-001"
    version = "1.0.0"
    key = f"{rule_id}:{version}"
    assert re.fullmatch(r"VRX-[A-Z0-9]+-\d+:\d+\.\d+\.\d+", key)
    assert key == "VRX-SQLI-001:1.0.0"


def test_required_rule_fields_present():
    rule = {
        "rule_id": "VRX-SQLI-001",
        "version": "1.0.0",
        "key": "VRX-SQLI-001:1.0.0",
        "weakness_ids": ["CWE-89"],
        "handler_key": "passive_error_pattern_v1",
        "mode": "passive",
        "safety_class": "passive",
        "request_budget": {"max_requests": 0, "timeout_ms": 2500, "rate_limit": None},
        "signals": [{"id": "VRX-SIGNAL-DB-ERROR", "verification_level": "suspected"}],
        "remediation_ids": ["VRX-FIX-PARAM-QUERIES"],
        "status": "draft",
        "owner": "appsec",
        "checksum": "sha256:deadbeef",
    }
    assert rule["weakness_ids"][0] == "CWE-89"
    assert rule["mode"] in ("passive", "active")
    assert rule["safety_class"] in ("passive", "low-impact")
    assert rule["status"] in ("draft", "reviewed", "approved", "retired")
    assert rule["request_budget"]["max_requests"] >= 0


def test_no_arbitrary_code_in_rule():
    rule_str = """rule_id: VRX-SQLI-001\nversion: 1.0.0\n"""
    assert "import" not in rule_str
    assert "exec(" not in rule_str
    assert "subprocess" not in rule_str
