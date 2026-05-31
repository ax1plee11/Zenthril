# PoC 15: Brute Force / Credential Stuffing Defense

**Risk:** High.

This repository does not include credential-stuffing tooling. The safe check below verifies that rate-limit defenses activate under a very small local burst.

## Safe PoC

```bash
BASE=http://127.0.0.1:8080
for i in 1 2 3 4 5; do
  curl -s -o /dev/null -w "attempt=$i status=%{http_code}\n" \
    -H "Content-Type: application/json" \
    -d '{"username":"victim","password":"wrong-password"}' \
    "$BASE/api/v1/auth/login"
done
```

Expected: repeated failures should be counted by brute-force protection.

## Recommendation

- Add TOTP/WebAuthn.
- Add account-level and IP-level throttling.
- Emit auth failure metrics.
- Add alerting for distributed attempts.
