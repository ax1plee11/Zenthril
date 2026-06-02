# Zenthril Documentation

This directory is the source of truth for project documentation beyond the root
`README.md`, `SECURITY.md`, and `THREAT_MODEL.md`.

The project intentionally avoids relying on a separate GitHub Wiki for current
technical documentation. A Wiki can drift from the codebase, while repository
documentation is versioned, reviewed, and updated together with code changes.

## Core Documents

- [`../README.md`](../README.md) - project overview, alpha status, features, and quick start.
- [`../SECURITY.md`](../SECURITY.md) - security status, disclosure process, and known limitations.
- [`../THREAT_MODEL.md`](../THREAT_MODEL.md) - trust boundaries, attacker model, and residual risk.
- [`ARCHITECTURE_TARGET.md`](ARCHITECTURE_TARGET.md) - target architecture direction.
- [`SECURITY_HARDENING.md`](SECURITY_HARDENING.md) - security hardening notes.
- [`CONNECTIVITY_RESILIENCE.md`](CONNECTIVITY_RESILIENCE.md) - multi-server and resilience design.
- [`ROADMAP.md`](ROADMAP.md) - development roadmap.
- [`DEPLOYMENT.md`](DEPLOYMENT.md) - deployment guidance.

## Update Policy

- Update documentation in the same PR/commit as the related code change.
- Keep alpha limitations explicit; do not describe unfinished E2EE,
  federation, voice, or scalability work as production-ready.
- Prefer repository Markdown files over external notes so documentation remains
  reviewable and versioned.
