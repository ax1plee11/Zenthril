# Full Project Security Analysis - 2026-06-25

Project: Zenthril (`C:\Users\User\Desktop\дс аналонг`)

## Skills used

- `performing-web-application-vulnerability-triage`: OWASP-style severity and exploitability triage.
- `testing-api-security-with-owasp-top-10`: API auth, access control, CORS, WebSocket, input/data exposure review.
- `testing-for-json-web-token-vulnerabilities`: JWT signing, expiry, and weak secret checks.
- `testing-for-broken-access-control`: role/permission and cross-user/guild boundary review.
- `testing-websocket-api-security`: origin checks, WS auth, rate limiting, and event authorization review.
- `testing-for-sensitive-data-exposure`: secret scanning and client/server data exposure review.
- `securing-github-actions-workflows`: CI permissions, scanners, and action-pinning review.
- `scanning-iac-and-images-with-trivy`: container/IaC workflow review; local Trivy binary was unavailable.
- `analyzing-sbom-for-supply-chain-vulnerabilities`: dependency and SBOM-oriented review using available local package managers.
- `performing-threat-modeling-with-owasp-threat-dragon`: threat model pass over assets, trust boundaries, and high-risk flows.

## Executive summary

The project is in a much better security state after the dependency cleanup: backend Go tests pass, frontend tests pass, production npm audit is clean, and `govulncheck` reports no reachable vulnerabilities.

The main residual risk is not a dependency CVE. It is an integration gap in the WebSocket invite path: the hub supports an `InviteAuthorizer`, but the production wiring still constructs it with the legacy nil authorizer path, so `invite.send` relay authorization is not actually enabled.

## Verification run

- `backend`: `go test ./...` passed.
- `backend`: `govulncheck ./...` passed with 0 reachable vulnerabilities.
- `client`: `npm audit --json` returned 0 vulnerabilities.
- `client`: `npm test` passed, 22 files / 133 tests.
- `client`: `npm run build` passed.
- `deployments/docker-compose.prod.yml`: `docker compose config --quiet` passed when required env vars were provided as temporary dummy values.

Local unavailable scanners:

- `trivy`: not installed locally.
- `gitleaks`: not installed locally.
- `semgrep`: not installed locally.
- `syft`: not installed locally.

CI already contains a Trivy filesystem scan job, but it is marked `continue-on-error: true`.

## Findings

### P1 - WebSocket invite authorization is implemented but not wired

Evidence:

- `backend/main.go:309` creates the hub with `hub.NewHubWithUserMessageLimiter(...)`.
- `backend/hub/hub.go:113` forwards that constructor to `NewHubFull(..., nil)`.
- `backend/hub/hub.go:85` documents nil `InviteAuthorizer` as legacy "allow all".
- `backend/hub/hub.go:557` only checks `CanSendInvite` when `inviteAuthorizer != nil`.

Impact:

Authenticated users can still relay arbitrary invite codes to arbitrary target user IDs through the WebSocket `invite.send` event, because the production hub path does not pass a real authorizer.

Recommendation:

Wire a real `InviteAuthorizer` into `main.go`, preferably the existing guild service if it implements/receives a `CanSendInvite(ctx, senderID, targetUserID, inviteCode)` method. Add an integration test that boots the same constructor path used by `main.go` and verifies unauthorized `invite.send` is rejected.

### P2 - Media click opens user-controlled URL without noopener

Evidence:

- `client/src/components/MessageItem.tsx:130` uses `window.open(url, "_blank")`.

Impact:

A malicious media URL opened from chat can retain access to `window.opener` in some browser contexts and attempt reverse-tabnabbing or opener manipulation.

Recommendation:

Use `window.open(url, "_blank", "noopener,noreferrer")` and explicitly clear `opened.opener = null` as a compatibility fallback.

### P2 - Localhost/private URLs are intentionally treated as direct media sources

Evidence:

- `client/src/components/MessageItem.tsx:17` bypasses image proxying for URLs containing `localhost` or `127.0.0.1`.

Impact:

This is client-side, not server-side SSRF, but it can make chat messages a delivery path for social engineering against local admin panels or localhost-only services. It also means the browser may directly request local resources when users view/click media.

Recommendation:

Restrict direct media rendering to trusted public hosts, or at least do not auto-render loopback/private-address URLs as media. Render them as normal links with a warning or require explicit click.

### P3 - GitHub Actions are tag-pinned, not SHA-pinned

Evidence:

- `.github/workflows/*.yml` uses actions such as `actions/checkout@v4`, `actions/setup-node@v4`, `github/codeql-action/*@v4`, and `aquasecurity/trivy-action@v0.36.0`.

Impact:

Tag pinning is standard but weaker than commit SHA pinning for supply-chain hardening. A compromised action tag or upstream release path could affect CI.

Recommendation:

Pin high-trust workflows to full commit SHAs and use Dependabot/Renovate to update those pins. Keep `permissions:` minimal per job instead of broad workflow-level permissions where possible.

### P3 - Trivy CI scan is non-blocking

Evidence:

- `.github/workflows/ci.yml` security scan uses `continue-on-error: true` for Trivy.

Impact:

Critical/high findings are uploaded to GitHub Security, but CI can still pass. This is acceptable during rollout, but it weakens enforcement.

Recommendation:

Once baseline findings are clean, remove `continue-on-error: true` or split advisory scan and blocking scan by severity.

### Informational - Production Kafka listener is plaintext inside the compose network

Evidence:

- `deployments/docker-compose.prod.yml:125` configures Redpanda as `PLAINTEXT://0.0.0.0:9092`.
- The service does not publish `9092` under `ports`, so exposure is internal to the Compose network unless the deployment platform changes networking.

Impact:

Internal plaintext may be acceptable for a single-host compose deployment, but it becomes a risk if the network is shared, bridged, or extended across hosts.

Recommendation:

Document the intended network boundary. For multi-host or shared infrastructure, enable authenticated/TLS Kafka listeners.

### Informational - Local generated backend static output is untracked

Evidence:

- `git status --short` shows `?? backend/static/`.

Impact:

This is likely generated frontend build output and was intentionally not pushed. It is a repo hygiene item, not an immediate vulnerability.

Recommendation:

Keep it ignored if generated, or clean it before release packaging to avoid accidentally shipping stale assets.

## Positive controls observed

- JWT configuration rejects weak/placeholder production secrets and enforces expiry checks.
- CORS and WebSocket allowed origins are fail-closed in production.
- WebSocket origin checks and per-user message rate limiting are present.
- Guild invite creation and joining use server-side permission checks and transactional invite usage.
- Sensitive `.env` data is ignored by git; `.env.example` contains placeholders, not real secrets.
- Frontend GIF provider keys come from `VITE_*` env variables and are not hardcoded in source.
- Production compose requires secrets via `${VAR:?set VAR}` instead of defaults.
- Production containers drop all Linux capabilities and set `no-new-privileges` for app services.

## Recommended next actions

1. Wire the WebSocket `InviteAuthorizer` in the production hub constructor path.
2. Harden `MessageItem` external media opening with `noopener,noreferrer`.
3. Decide whether localhost/private URLs should be auto-rendered as media.
4. Make Trivy blocking in CI after the baseline is clean.
5. Pin GitHub Actions to SHAs for stronger supply-chain posture.
6. Install local `trivy`, `gitleaks`, `semgrep`, and `syft` if recurring local audits are desired.
