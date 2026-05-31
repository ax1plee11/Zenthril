# Zenthril Roadmap

This roadmap describes the planned path from the current alpha prototype toward a safer, more maintainable, and research-friendly messaging platform.

Zenthril is still an alpha project. Items marked as planned should not be presented as production-ready functionality until they are implemented, tested, and documented.

## Phase 0: Security Foundation

Status: mostly complete for the current alpha scope.

- Strict CORS and WebSocket origin validation.
- WebSocket message size limits and basic flood protection.
- Security headers middleware.
- Protected metrics and operational endpoints.
- JWT access tokens, refresh tokens, and Redis-backed revocation.
- Safer production configuration validation.
- Non-root Docker runtime and deployment health checks.

Remaining work:

- Move private key storage to OS keychain or Stronghold in production desktop builds.
- Expand security tests around browser-origin attacks and token replay.
- Document an explicit threat model for the alpha release.

## Phase 1: E2EE Protocol Maturity

Status: in progress.

- Integrate Double Ratchet state into real message send/receive flows.
- Add X3DH-style pre-key bundles and one-time pre-key consumption.
- Add safety number verification in the client UX.
- Improve multi-device session handling.
- Add session healing after skipped messages and device changes.
- Add property-based tests and test vectors for ratchet behavior.

Exit criteria:

- The server cannot decrypt message content.
- Key rotation is exercised in automated tests.
- Device revocation blocks future sessions with revoked devices.
- Documentation clearly states remaining metadata exposure.

## Phase 2: Architecture And Observability

Status: partially started.

- Continue moving the next-generation API to explicit dependency injection with `uber/fx`.
- Add OpenTelemetry traces and metrics to HTTP, WebSocket, database, and event paths.
- Split gateway, event bus, repository, and service boundaries more clearly.
- Keep the legacy API stable while new internals mature.
- Add structured logs for security-relevant events.

## Phase 3: Realtime Scaling

Status: planned.

- Run multiple WebSocket gateway instances.
- Add Redis Pub/Sub fan-out between gateway nodes.
- Prepare the event layer for Kafka, Redpanda, or NATS JetStream.
- Add connection draining and graceful shutdown tests.
- Define delivery semantics: at-least-once delivery with idempotent client-side deduplication.

## Phase 3.5: Anti-Censorship And Resilience

Status: foundation started.

- Client loads a dynamic server list from `servers.json`.
- Users can switch servers and add custom mirrors.
- The client caches the server list and falls back across configured servers on network errors.
- Mirror origins are expanded into normal fallback targets.
- DNS-over-HTTPS bootstrap and Tor/onion metadata are present.
- Bridge fallback planning and P2P direct-message scaffolding are present.

Remaining work:

- Sign server lists and pin trust roots in the client.
- Add DNS-over-HTTPS bootstrap for server-list discovery.
- Add optional Tor `.onion` entries and external proxy mode.
- Add bridge-node metadata and P2P fallback routing.
- Keep all anti-censorship claims explicit about limitations.

## Phase 4: Federation

Status: planned; current federation endpoints are not production-ready.

- Define a minimal federation protocol before expanding public claims.
- Add node identity, signed server-to-server requests, and replay protection.
- Build compatibility tests between two local Zenthril nodes.
- Document which data is federated and which data remains local.

Current foundation:

- Federation inbox stores encrypted message envelopes.
- Federation remains disabled by default and bearer-token protected.

## Phase 5: Research-Grade Evaluation

Status: active and ongoing.

- Maintain reproducible benchmark scripts.
- Record hardware, dataset, and test parameters for every published result.
- Compare security and scaling tradeoffs with Matrix, Signal-style messaging, and Discord-like realtime systems.
- Produce graphs and analysis suitable for a diploma or master's thesis.
