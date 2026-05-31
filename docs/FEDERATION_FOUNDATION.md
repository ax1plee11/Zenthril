# Federation Foundation

Zenthril federation is alpha-stage and disabled by default. It should not be
described as production-ready.

## Current Scope

- `/federation/v1/announce` registers active peer nodes.
- `/federation/v1/peers` lists active peer nodes.
- `/federation/v1/inbox` accepts encrypted server-to-server message envelopes.
- All federation endpoints require `FEDERATION_ENABLED=true`.
- All federation endpoints require `Authorization: Bearer <FEDERATION_TOKEN>`.

## Message Envelope

```json
{
  "message_id": "global-message-id",
  "source_domain": "node-a.example",
  "target_domain": "node-b.example",
  "sender_user_id": "user-a",
  "target_user_id": "user-b",
  "payload": {
    "ciphertext": "...",
    "iv": "...",
    "tag": "...",
    "protocol_version": 1
  }
}
```

The federation service stores encrypted payloads only. Plaintext message content
must never be sent through federation.

## Remaining Work

- Add per-node signing keys and request signatures.
- Add replay windows and timestamp validation.
- Add delivery workers for outbound federation queues.
- Add compatibility tests with two local nodes.
- Add bridge relay policy and abuse controls.
