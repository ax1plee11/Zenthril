# PoC 14: Admin Privilege Escalation

**Risk:** High.

## Safe PoC

Use a normal non-admin token:

```bash
curl -i -X POST \
  -H "Authorization: Bearer $NORMAL_USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"test"}' \
  http://127.0.0.1:8080/api/v1/admin/users/$TARGET_USER_ID/ban
```

Expected: `403 forbidden`.

## Current Review Notes

Admin routes are protected by auth middleware and an `ADMIN_USER_IDS` allowlist.

## Recommendation

- Add integration tests for admin routes with normal user token, missing token, expired token, and admin token.
- Keep admin identity outside client-controlled claims where possible.
