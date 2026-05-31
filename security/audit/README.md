# Zenthril Red Team Audit Kit

This directory contains controlled red-team checks for the Zenthril alpha project.

The PoCs are intentionally scoped for local, authorized testing only. They are designed to verify that security controls reject attacks, not to provide tooling for attacking third-party systems.

## Safety Rules

- Run only against a Zenthril instance you own or have written permission to test.
- Default targets are `localhost` / `127.0.0.1`.
- The WebSocket flood check is bounded and refuses non-local targets.
- Credential stuffing and destructive denial-of-service are documented as risks, but this kit does not include mass-attack tooling.

## Files

- `REPORT.md` - current audit findings and residual risk.
- `pocs/01_csws_hijacking.html` - browser CSWSH origin check.
- `pocs/02_jwt_alg_none.py` - JWT `alg:none` rejection check.
- `pocs/03_ws_flood_bounded.py` - bounded local WebSocket rate-limit check.
- `pocs/04_missing_origin.py` - WebSocket missing-Origin rejection check.
- `pocs/05_e2ee_envelope_tamper.md` - E2EE envelope tamper cases.
- `pocs/06_rate_limit_bypass.md` - rate-limit bypass checklist.
- `pocs/07_idor_checks.md` - guild/message/friend IDOR checks.
- `pocs/08_message_injection_xss.md` - message injection and rendering checks.
- `pocs/09_refresh_token_replay.md` - refresh-token replay regression check.
- `pocs/10_cors_misconfiguration.md` - CORS preflight checks.
- `pocs/11_supply_chain.md` - dependency and CI checks.
- `pocs/12_metadata_leakage.md` - metadata leakage review.
- `pocs/13_webrtc_ice_leak.md` - WebRTC ICE leak checks.
- `pocs/14_admin_privilege_escalation.md` - admin boundary checks.
- `pocs/15_bruteforce_defense.md` - brute-force defense checks.

## Quick Start

```bash
cd security/audit/pocs
python 02_jwt_alg_none.py --base-url http://127.0.0.1:8080
python 04_missing_origin.py --ws-url ws://127.0.0.1:8080/ws
python 03_ws_flood_bounded.py --ws-url ws://127.0.0.1:8080/ws --ticket YOUR_WS_TICKET
```

Expected result for hardened builds: attack attempts should be rejected with `401`, `403`, close frames, or safe validation errors.
