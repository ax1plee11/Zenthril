# Security Hardening Notes

Zenthril is currently an alpha project. The backend now defaults to a stricter security posture for browser-facing and operational surfaces, but the project should still be treated as unsuitable for highly sensitive production communication until the E2EE roadmap is complete.

## Runtime Controls

- `CORS_ALLOWED_ORIGINS` must contain exact origins only, for example `https://app.example.com`.
- `WS_ALLOWED_ORIGINS` must contain exact origins only and is required for WebSocket upgrades.
- Wildcards such as `*` and origins with paths are rejected by configuration validation.
- WebSocket upgrades fail closed when no allowed origin is configured.
- WebSocket handlers reject missing or cross-site `Origin` headers to prevent Cross-Site WebSocket Hijacking.
- Legacy WebSocket tickets are not consumed until the request origin is accepted.
- WebSocket messages have size limits, per-connection limits, per-user limits, and malformed command limits.

## Operational Endpoints

Operational endpoints are protected in production:

- Legacy API: `/health`, `/metrics`, `/metrics/prometheus`
- Next-generation API: `/healthz`, `/readyz`, `/api/v2/gateway/stats`

Use an authorization header:

```http
Authorization: Bearer <OPERATIONAL_TOKEN or METRICS_TOKEN>
```

`OPERATIONAL_TOKEN` is preferred for health/readiness/stat endpoints. If it is not set in the next-generation API, `METRICS_TOKEN` is used as a fallback.

## Deployment Requirements

Production environments must provide:

- `JWT_SECRET`
- `METRICS_TOKEN`
- `OPERATIONAL_TOKEN`
- `CORS_ALLOWED_ORIGINS`
- `WS_ALLOWED_ORIGINS`
- `DB_URL`
- `REDIS_URL`

Kubernetes probes in `deployments/k8s/zenthril-core.yaml` use `OPERATIONAL_TOKEN` through an exec probe. Traefik health checks in `deployments/traefik/dynamic.yml` also include an authorization header.

Caddy health checks in `deployments/Caddyfile` also send `Authorization: Bearer {$OPERATIONAL_TOKEN}`. Keep proxy and application tokens in sync when rotating secrets.

## JWT And Refresh Tokens

- Access tokens are short-lived.
- Refresh tokens are stored server-side in Redis by token ID and hash.
- Refresh token rotation marks old token IDs as used.
- Refresh token replay triggers revocation of the user's active refresh token set.
- Logged-out access tokens are blacklisted in Redis until natural expiry.
- JWT validation pins `HS256` and rejects `alg:none` / algorithm-confusion attempts.

## Remaining Alpha Risks

Client-side storage still needs more hardening:

- The web fallback stores access tokens in `localStorage`.
- E2EE private-key fallback paths use `localStorage` when Tauri commands are unavailable.
- Tauri currently uses `tauri-plugin-store`; a production-grade desktop build should move private keys to OS keychain or Stronghold.

E2EE is still foundational:

- X25519 and AES-256-GCM primitives exist.
- Device key management and revocation are present.
- Backend Double Ratchet state and HKDF key-evolution foundations are present.
- Full message protocol integration, robust session healing, and mature multi-device recovery are still roadmap items.
