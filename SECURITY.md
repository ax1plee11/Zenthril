# Security Policy

## Supported versions

Security fixes are applied to the default branch (`main`). For production deployments, track releases/tags when they exist.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security problems.

Preferred options:

1. **GitHub Security Advisories** (if enabled for the repository): *Security → Advisories → Report a vulnerability*.
2. Or email the maintainer privately (add your contact in this file when you publish the service).

Include:

- Description and impact
- Steps to reproduce (minimal)
- Affected component (backend / client / infra)
- Optional: suggested fix or patch

We try to acknowledge within **7 days** and provide a timeline for a fix. Critical issues (RCE, auth bypass, mass data leak) are prioritized.

## Scope

In scope:

- Authentication/authorization bugs
- Data exposure (messages, tokens, keys)
- Server-side injection, IDOR, broken access control
- WebSocket / API abuse that breaks security guarantees

Out of scope (unless they directly lead to a security issue):

- Generic DoS by volume without a specific bug
- Social engineering
- Issues in third-party services (Tenor/Giphy) — report to them

## Secure deployment reminders

- Never commit `.env` or real secrets.
- Use strong `JWT_SECRET`, TLS in production, restrict DB/Redis to private networks.
- Set `CORS_ALLOWED_ORIGINS` / `WS_ALLOWED_ORIGINS` explicitly in production.
- Restrict `ADMIN_USER_IDS` to trusted operator accounts.

---

*This policy is a template — adjust contacts and SLAs for your public service.*
