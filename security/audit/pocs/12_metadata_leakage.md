# PoC 12: Metadata Leakage

**Risk:** Medium.

## What A Server Can Still See

Even with encrypted message payloads, the server can observe:

- Account IDs.
- Device IDs.
- Guild/channel membership.
- Message timing.
- Ciphertext size.
- IP address and approximate location.
- WebSocket connection timing.
- WebRTC signaling metadata.

## Safe Review

Capture local traffic metadata only:

```bash
# Observe request paths and sizes through local reverse proxy logs.
docker compose up
```

Expected: plaintext message content should not appear in backend logs or DB rows, but metadata will exist.

## Recommendation

- Document metadata limitations clearly.
- Consider padding and batching.
- Avoid logging payload fields.
- Add privacy-preserving metrics labels.
