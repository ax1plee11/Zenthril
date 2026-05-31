# PoC 06: Rate-Limit Bypass Review

**Risk:** High.

## Current Controls

- Login route uses `BruteForceProtect`.
- HTTP requests pass through IP rate limiting.
- Message sends pass through spam guard.
- WebSocket has per-connection and per-user limits.

## Safe PoC

Use only a small local burst. Do not run large traffic against shared systems.

```bash
BASE=http://127.0.0.1:8080
for i in 1 2 3 4 5; do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -H "Content-Type: application/json" \
    -d '{"username":"does-not-exist","password":"wrong"}' \
    "$BASE/api/v1/auth/login"
done
```

Expected: after configured thresholds, login attempts should be slowed or rejected.

## Recommendation

- Add integration tests for IP + username + account rate limits.
- Add distributed rate-limit counters for multi-node deployments.
- Emit metrics for rate-limit hits.
