# Changelog

## Security remediation — 2026-11-12

- Enforce TOTP before issuing login tokens, reject OTP replay, and encrypt stored MFA secrets.
- Revoke account-wide access/refresh sessions and pending WebSocket tickets; close local realtime sessions on revocation.
- Consume password-reset tokens atomically and restrict cross-user device-key claims.
- Preserve the transport peer address for trusted-proxy-aware rate limiting and fail closed on authorization errors.
- Update Go, JavaScript, Python and Rust dependencies and repair Docker build/persistence configuration.
- Include backend moderation/audit source and migrations so a clean checkout contains the application it builds.
- Publish architecture, audit and knowledge-base material; link ZenthrilDB as a pinned submodule.
- See [the remediation report](docs/security/SECURITY_REMEDIATION_2026-11-12.md) for validation, deployment steps and remaining limitations. Shannon exploitation testing was not completed.

All notable changes to this project will be documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

## [Unreleased]

### Fixed
- JWT_ACCESS_TTL is now configurable via environment variable
- PostgreSQL and Redis ports no longer exposed publicly in docker-compose
- Memory event bus banned in production environment
- Critical Vitest dev-dependency advisory resolved by upgrading Vitest packages
- Go JWT and Redis dependencies updated for govulncheck-reported advisories

### Added
- E2EE key lifecycle documentation
- TOTP MFA scaffold
- OpenAPI 3.1 skeleton
- WebSocket integration test stubs
- Web/PWA build target
- Prometheus + Grafana monitoring stack
- Docker resource limits
- CI go vet, critical npm audit gate, govulncheck workflow, CodeQL workflow, and Dependabot config
- CI Go toolchain pinned to a patched version for standard-library vulnerability scanning

### Changed
- Repository positioning clarified as public-visible/source-available for transparency, portfolio, and authorship provenance, but not open-source licensed.
- Legal terms updated to proprietary / all-rights-reserved usage with explicit permission required for copying, redistribution, public forks, commercial use, and derivative publication.

### Known Issues
- The Linux GTK dependency chain still retains the GLib advisory documented in the 2026-11-12 remediation report; independent cryptographic auditing remains outstanding.
