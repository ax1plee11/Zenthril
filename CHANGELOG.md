# Changelog

All notable changes to this project will be documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

## [Unreleased]

### Fixed
- JWT_ACCESS_TTL is now configurable via environment variable
- PostgreSQL and Redis ports no longer exposed publicly in docker-compose
- Memory event bus banned in production environment

### Added
- E2EE key lifecycle documentation
- TOTP MFA scaffold
- OpenAPI 3.1 skeleton
- WebSocket integration test stubs
- Web/PWA build target
- Prometheus + Grafana monitoring stack
- Docker resource limits
