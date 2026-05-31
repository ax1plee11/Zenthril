# PoC 09: Refresh Token Replay

**Risk:** High.

## Safe PoC

After logging in locally, save the first refresh token as `$REFRESH`.

```bash
BASE=http://127.0.0.1:8080

curl -s -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH\"}" \
  "$BASE/api/v1/auth/refresh"

curl -i -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH\"}" \
  "$BASE/api/v1/auth/refresh"
```

Expected:

- First request succeeds and rotates the token.
- Second request returns `401 invalid_refresh_token`.
- Server logs a replay warning and revokes active refresh tokens for that user.

## Recommendation

- Keep Redis `GETDEL` atomic consumption.
- Add a concurrent replay test where two refresh requests race with the same token.
