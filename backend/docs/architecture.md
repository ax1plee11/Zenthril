# Zenthril Backend Architecture Status

## Server Inventory

| Binary | Entrypoint | Status | Production use |
|---|---|---|---|
| `main.go` | Legacy modular monolith | Stable alpha | Yes |
| `cmd/api` | Next-generation gateway / CQRS skeleton | Experimental | Not alone |
| `cmd/sfu` | SFU control-plane skeleton | Stub | No |
| `cmd/worker` | Background worker skeleton | Stub | No |

## Production entrypoint

Production compose uses the full `main.go` backend because it currently owns the complete API surface:

- Authentication and refresh-token lifecycle
- Device keys
- Guilds and channels
- Messages
- Friends
- Legacy WebSocket hub
- Metrics and operational endpoints

## Experimental `cmd/api`

`cmd/api` is the future gateway/CQRS entrypoint. It currently wires:

- WebSocket gateway
- Redis-backed token revocation checks
- Event bus abstraction
- Shard manager
- Operational endpoints

It does not yet wire the full business API. Do not deploy it as the only production backend until the migration is complete.

## Production guards

The experimental `cmd/api` refuses unsafe production modes:

- `NoopSessionValidator` is rejected.
- `InMemoryEventStore` is rejected.
- `GATEWAY_ROLE=primary` requires a global-ban lookup.
- `EVENT_BUS_DRIVER=memory` is rejected by config validation.

## Gateway roles

`GATEWAY_ROLE=primary` is the default. It is intended for a main gateway/API node and must enforce account lifecycle checks.

`GATEWAY_ROLE=edge` is for edge-only gateway nodes behind a primary API layer. Edge nodes may omit direct global-ban lookup only when the primary layer already enforces bans before traffic reaches the edge.
