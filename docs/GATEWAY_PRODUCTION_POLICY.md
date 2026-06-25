# Gateway production policy

Zenthril currently has two backend entrypoints:

- `backend/main.go` is the full API used by production compose.
- `backend/cmd/api` is the next-generation gateway/CQRS entrypoint and remains experimental.

## Gateway roles

`GATEWAY_ROLE` defines how strict the next-generation gateway must be about account lifecycle checks.

| Role | Use case | Global ban lookup required in production |
|---|---|---|
| `primary` | Main API/realtime gateway | Yes |
| `edge` | Edge-only gateway behind a primary API | No |

The default is `primary`.

## Security rule

A primary production gateway must not start unless global ban lookup is wired. This prevents a banned account from keeping realtime access through a gateway that only checks JWT validity and token revocation.

Edge gateways may omit the ban lookup only when bans are enforced by a primary API layer before traffic reaches the edge.

## Release note

`cmd/api` must not replace the full production API until durable event storage and global ban lookup are wired into its dependency graph.
