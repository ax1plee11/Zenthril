# Zenthril 1M Architecture Plan

This document describes the target production architecture for 1M+ registered
users and 300k+ concurrent websocket users.

## Target go.mod

```go
module zenthril-backend

go 1.23

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.1
	github.com/redis/go-redis/v9 v9.6.1
	github.com/segmentio/kafka-go v0.4.47
	github.com/nats-io/nats.go v1.37.0
	github.com/gocql/gocql v1.7.0
	github.com/minio/minio-go/v7 v7.0.80
	github.com/prometheus/client_golang v1.20.5
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.31.0
	go.opentelemetry.io/otel/sdk v1.31.0
	go.uber.org/fx v1.23.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.27.0
)
```

The current implementation keeps the first next-gen packages on existing
dependencies to avoid a risky dependency flood. Kafka/NATS/Scylla/S3/OTel
adapters should be added one adapter at a time with integration tests.

## Folder Structure

```text
backend/
  cmd/
    api/
    worker/
    sfu/
  internal/
    app/
    config/
    crypto/
    domain/
    event/
    gateway/
    handler/
    middleware/
    pubsub/
    repository/
    service/
  pkg/
deployments/
  docker-compose.prod.yml
  Caddyfile
  k8s/
    zenthril-core.yaml
```

## WebSocket Scaling

WebSocket gateway nodes are stateless from the business perspective. A gateway
keeps only local sockets, local subscriptions, and local backpressure queues.
Auth, permissions, messages, and presence are external services.

Routing strategy:

- Edge layer uses sticky affinity based on a short-lived websocket ticket or
  `user_id` hash.
- Sticky sessions are an optimization, not a correctness requirement.
- Every domain write emits an event into Kafka/Redpanda or NATS JetStream.
- Gateway nodes consume delivery events and deliver only to local sockets.
- Cross-node fan-out uses event partition keys such as `channel_id`, `guild_id`,
  and `user_id`.
- Delivery is at-least-once. Clients and servers deduplicate by `event_id` and
  monotonic per-channel sequence numbers.

Per-node target:

- 30k-75k sockets per gateway pod depending on CPU, memory, kernel limits, and
  message fan-out shape.
- 300k concurrent users means roughly 6-12 gateway pods plus hot spare capacity.
- Connection draining starts on SIGTERM: stop accepting new sockets, keep old
  sockets until drain timeout, then close cleanly.

## Storage And Sharding

PostgreSQL:

- users/auth/devices: shard by `user_id`
- guilds/channels/roles: shard by `guild_id`
- membership index: duplicate minimal lookup rows by `user_id` for fast guild
  list reads

Scylla/Cassandra:

- messages partition key: `(channel_id, time_bucket)`
- clustering: `(created_at DESC, message_id)`
- write path is append-only; edits/deletes are tombstone events or compacted
  materialized state

Redis Cluster:

- presence
- sessions
- websocket ticket state
- distributed rate limiting
- short-lived dedupe windows

Kafka/Redpanda or NATS JetStream:

- message.created.v1
- gateway.deliver.v1
- presence.changed.v1
- audit.log.v1

S3-compatible storage:

- media and attachments
- AV scan state in Postgres
- object keys are content-addressed plus owner namespace

## E2EE Plan

Phase 1:

- identity key per device
- signed prekey per device
- one-time prekey pool
- device list with safety number / fingerprint display

Phase 2:

- X3DH session bootstrap
- Double Ratchet for direct messages and small private groups
- skipped message key cache with strict bounds
- server-side encrypted blobs only

Phase 3:

- group sender keys for guild channels
- epoch rotation on member add/remove/device revoke
- encrypted key backup protected by client-side recovery secret

Phase 4:

- MLS-like group state once the product needs large encrypted rooms
- independent cryptographic review before claiming production-grade E2EE

## Implementation Phases

1. Next-gen skeleton: config, DI, gateway registry, event bus interfaces,
   shard manager, deployment skeleton.
2. Replace in-memory bus with Redpanda adapter and delivery worker.
3. Move message history writes to Scylla while keeping Postgres metadata.
4. Add Redis Cluster presence/session/rate-limit adapters.
5. Introduce device key APIs and client-side E2EE flows.
6. Split API/gateway/worker into independent deployable services.
7. Multi-region: regional gateways, local writes, global async federation.
