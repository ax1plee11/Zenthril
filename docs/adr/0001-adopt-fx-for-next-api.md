# ADR 0001: Adopt uber/fx for the next-generation API

## Status

Accepted

## Context

Zenthril currently has two backend entrypoints:

- `backend/main.go`, the legacy API used by the existing product surface.
- `backend/cmd/api`, the next-generation API foundation for gateway, event bus, sharding, and future service boundaries.

The project needs clearer dependency construction before adding more production-grade pieces such as observability, distributed gateway adapters, and storage abstractions.

## Decision

Use `uber/fx` for the next-generation API dependency graph under `backend/internal/app`.

The initial module wires:

- configuration
- structured logger
- shard manager
- event bus
- gateway registry
- gateway authenticator
- gateway handler
- API container

The legacy `backend/main.go` remains unchanged for now.

## Rationale

- `fx` makes dependencies explicit without introducing code generation.
- Lifecycle hooks provide a natural place for startup/shutdown behavior.
- The graph can grow incrementally while the legacy entrypoint remains stable.
- This is useful for research and documentation because architecture boundaries are visible in code.

## Consequences

Positive:

- New services should be added through `internal/app.Module`.
- `cmd/api` can stay focused on HTTP routing and server lifecycle.
- Future OpenTelemetry and distributed adapters can be introduced as providers.

Tradeoffs:

- The project temporarily has both manual wiring and `fx` wiring.
- Contributors need to understand `fx` basics before modifying the next-generation API.

## Follow-up

- Move operational middleware into `internal/middleware`.
- Add OpenTelemetry providers to `internal/app.Module`.
- Gradually migrate legacy API services behind explicit constructors before replacing `backend/main.go`.
