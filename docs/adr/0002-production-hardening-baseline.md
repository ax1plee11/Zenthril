# ADR 0002: Establish a production hardening baseline

## Status

Accepted

## Context

Zenthril is an alpha messaging project, but even alpha deployments can be exposed to browsers, reverse proxies, and public networks. The highest-risk surfaces are WebSocket upgrades, CORS, token handling, operational endpoints, and container runtime privileges.

The project needs a clear baseline that prevents accidental insecure deployments while still allowing local development.

## Decision

Adopt a production hardening baseline across backend code, configuration, and deployment files:

- Exact allowlists for `CORS_ALLOWED_ORIGINS` and `WS_ALLOWED_ORIGINS`.
- Protected metrics, health, readiness, and gateway stats endpoints in production.
- Dedicated `METRICS_TOKEN` and `OPERATIONAL_TOKEN`.
- Security headers middleware.
- JWT validation pinned to the expected signing method.
- Refresh token revocation with Redis-backed state.
- WebSocket message size and rate limits.
- Non-root runtime containers.
- Health checks in Docker Compose and reverse proxy configs.
- Graceful shutdown for API entrypoints.

## Rationale

These controls reduce the chance of common alpha-stage mistakes:

- Cross-Site WebSocket Hijacking.
- Browser-origin confusion.
- Exposed metrics or debug surfaces.
- Token replay after logout.
- Containers running with unnecessary privileges.
- Hard shutdowns during deployment.

## Consequences

Positive:

- Production-like environments fail earlier when required secrets are missing.
- Reverse proxies can perform authenticated health checks.
- Security behavior is documented and easier to review.

Tradeoffs:

- Local setup requires more environment variables.
- Operators must rotate and keep operational tokens in sync across app and proxy configuration.
- Some deployment templates remain examples and still require environment-specific review.

## Follow-up

- Add a formal threat model document.
- Add integration tests for origin validation and token replay.
- Add automated container image scanning in CI.
