# Security Hardening Notes

Zenthril is currently an alpha project. The backend now defaults to a stricter security posture for browser-facing and operational surfaces, but the project should still be treated as unsuitable for highly sensitive production communication until the E2EE roadmap is complete.

## Runtime Controls

- `CORS_ALLOWED_ORIGINS` must contain exact origins only, for example `https://app.example.com`.
- `WS_ALLOWED_ORIGINS` must contain exact origins only and is required for WebSocket upgrades.
- Wildcards such as `*` and origins with paths are rejected by configuration validation.
- WebSocket upgrades fail closed when no allowed origin is configured.
- WebSocket handlers reject missing or cross-site `Origin` headers to prevent Cross-Site WebSocket Hijacking.
- Legacy WebSocket tickets are not consumed until the request origin is accepted.
- The next-generation gateway rejects untrusted origins before authentication and does not accept long-lived credentials from query strings.
- WebSocket messages have size limits, per-connection limits, per-user limits, and malformed command limits.
- CORS preflight requests must ask only for allowed methods and headers.

## Operational Endpoints

Operational endpoints are protected in production:

- Legacy API: `/health`, `/healthz`, `/ready`, `/readyz`, `/metrics`, `/metrics/prometheus`
- Next-generation API: `/healthz`, `/readyz`, `/api/v2/gateway/stats`
- Debug and pprof-style `/debug/*` endpoints are explicitly blocked in production.

Use an authorization header:

```http
Authorization: Bearer <OPERATIONAL_TOKEN or METRICS_TOKEN>
```

`OPERATIONAL_TOKEN` is used for health/readiness/stat endpoints. `METRICS_TOKEN` is used for metrics scraping. Metrics can also be accessed by configured admins with a valid access token.

`/health` and `/healthz` are liveness probes: they only confirm the process can
serve HTTP. `/ready` and `/readyz` are readiness probes: in the legacy backend
they ping PostgreSQL and Redis with a short timeout before reporting ready. In
production, failed readiness responses intentionally avoid exposing dependency
names or connection details. The next-generation `cmd/api` entrypoint currently
reports only the dependencies actually wired into its container (gateway,
event bus configuration, and shard configuration); DB/Redis readiness there
will become strict after those clients are moved into the DI container.

Prometheus metrics include WebSocket security counters:

- `zenthril_ws_rejected_total`
- `zenthril_ws_rate_limit_hits_total`
- `zenthril_ws_malformed_total`
- `zenthril_readiness_failures_total`

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
- Refresh token rotation consumes the Redis record atomically with `GETDEL`.
- Refresh token replay triggers revocation of the user's active refresh token set.
- Logged-out access tokens are blacklisted in Redis until natural expiry.
- JWT validation pins `HS256`, requires `exp`, and rejects `alg:none` / algorithm-confusion attempts.
- Production refresh/logout requests using auth cookies require an `Origin` header that has already passed the strict global CORS allowlist.

## Federation

- Federation endpoints are disabled by default.
- `FEDERATION_ENABLED=true` is required before `/federation/v1/*` will process requests.
- Enabled federation endpoints require `Authorization: Bearer <FEDERATION_TOKEN>`.
- Federation request bodies are size-limited before JSON decoding.

## Remaining Alpha Risks

Client-side storage still needs more hardening:

- The web fallback stores access tokens in `localStorage`.
- E2EE private-key fallback paths use `localStorage` only in development unless `VITE_ALLOW_INSECURE_KEY_STORAGE=true` is set explicitly.
- Tauri currently uses `tauri-plugin-store`; a production-grade desktop build should move private keys to OS keychain or Stronghold.

E2EE is still foundational:

- X25519 and AES-256-GCM primitives exist.
- Device key management and revocation are present.
- Message payloads now use a protocol-v1 envelope with persisted AES-GCM tag, HKDF-derived keys, and AAD binding.
- Backend Double Ratchet state and HKDF key-evolution foundations are present.
- Full X3DH/Double Ratchet integration, robust session healing, and mature multi-device recovery are still roadmap items.
