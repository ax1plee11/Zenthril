# PoC 10: CORS Misconfiguration

**Risk:** High.

## Safe PoC

```bash
BASE=http://127.0.0.1:8080

curl -i -X OPTIONS "$BASE/api/v1/auth/login" \
  -H "Origin: https://evil.example" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization, Content-Type"
```

Expected: `403` and no reflected `Access-Control-Allow-Origin`.

Allowed origin check:

```bash
curl -i -X OPTIONS "$BASE/api/v1/auth/login" \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization, Content-Type"
```

Expected: `204` and exact `Access-Control-Allow-Origin: http://localhost:5173`.

## Recommendation

- Never allow `*` with credentials.
- Keep exact origin validation in config and middleware.
