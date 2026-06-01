# Zenthril Target Architecture

Zenthril is an alpha self-hosted messenger. The target architecture below is a
product roadmap for a safer, more maintainable system; it is not a claim that
all pieces are production-ready today.

## Current Weak Spots

- Legacy `backend/main.go` still owns much of the HTTP surface and service
  wiring.
- The next-generation `cmd/api` already has `uber/fx`, a secure gateway, and
  an event bus, but most domain modules are not yet moved behind it.
- Message, guild, auth, device, federation, and moderation flows are still
  mostly CRUD/service oriented instead of command/query oriented.
- Event publishing exists, but there is no durable production event store yet.
- E2EE remains alpha and must not be described as Signal-grade.
- Federation and P2P are experimental and must stay disabled or explicit opt-in
  until trust, replay protection, and abuse controls mature.
- Observability exists in pieces; tracing and security metrics need one
  consistent model.

## Target Backend Structure

```text
backend/
  cmd/
    api/                    # production API / secure gateway entrypoint
    worker/                 # projections, async jobs, fan-out workers
    sfu/                    # voice SFU process
  internal/
    app/                    # fx dependency graph
    config/                 # validated fail-closed config
    domain/                 # aggregate types, value objects, domain errors
    cqrs/                   # command bus, query bus, event store contracts
    application/            # command/query handlers
    repository/             # storage adapters and shard manager
    event/                  # event bus adapters: memory, Kafka, NATS later
    gateway/                # secure multi-instance WebSocket gateway
    security/               # anti-abuse, request guards, token policies
    middleware/             # HTTP middleware
    crypto/                 # backend crypto primitives and protocol metadata
    observability/          # logging, metrics, tracing
  migrations/
```

## Target Client Structure

```text
client/src/
  app/                      # app shell and route/state boundaries
  api/                      # typed API client and auth transport
  crypto/                   # E2EE envelope, AAD, ratchet foundations
  features/
    messaging/
    guilds/
    devices/
    security-status/
    voice/
    p2p/
  platform/                 # Tauri/web abstractions
  store/                    # small persistent state helpers
  transport/                # WebSocket framing and connectivity policy
  components/               # shared UI
```

## CQRS And Event Sourcing

Write operations should enter through explicit commands:

```text
HTTP/WebSocket handler -> CommandBus -> CommandHandler -> Aggregate/Policy -> EventStore -> EventBus
```

Read operations should use queries and projections:

```text
HTTP handler -> QueryBus -> QueryHandler -> read model/repository
```

For messaging:

- `SendMessageCommand` validates channel membership and encrypted envelope.
- The handler appends `message.created.v1` to the message stream.
- The event bus publishes delivery events for gateway fan-out.
- Projection workers update read models for history, moderation, and search.

`backend/internal/cqrs` currently provides the alpha in-memory command bus,
query bus, and event store. Durable adapters are required before production
event-sourced aggregates.

## Secure Gateway Layer

The secure gateway should remain separate from domain logic:

- strict WebSocket Origin validation;
- no query-string bearer tokens;
- message size and malformed-message limits;
- per-connection and per-user rate limiting;
- draining support for deploys;
- event-bus based fan-out for multi-instance deployment.

## Security Boundaries

- `security/` and middleware enforce transport/request-level controls.
- Domain command handlers enforce authorization and invariants.
- Repositories never decide permissions.
- Event handlers treat all event payloads as untrusted input unless produced by
  local trusted command handlers.

## Connectivity Resilience

Zenthril supports self-hosted outage recovery through administrator-configured
backup endpoints and explicit custom server selection. It does not promise
invisible, indistinguishable, or impossible-to-block traffic.

