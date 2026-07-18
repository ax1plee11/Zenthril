# Changelog

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
- `npm audit` still reports the Vite/esbuild development-server advisory at moderate severity; fixing it requires a Vite major-version migration and is tracked as a follow-up batch.
