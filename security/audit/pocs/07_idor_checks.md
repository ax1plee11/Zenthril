# PoC 07: IDOR Checks

**Risk:** High.

## Safe PoC

Create two users in a local database:

- User A owns guild/channel/message.
- User B is not a member.

Then verify User B receives `403` for:

```bash
curl -i -H "Authorization: Bearer $TOKEN_B" \
  http://127.0.0.1:8080/api/v1/guilds/$GUILD_A/members

curl -i -H "Authorization: Bearer $TOKEN_B" \
  http://127.0.0.1:8080/api/v1/channels/$CHANNEL_A/messages

curl -i -X PATCH -H "Authorization: Bearer $TOKEN_B" -H "Content-Type: application/json" \
  -d '{"payload":{"ciphertext":"AA==","iv":"AAAAAAAAAAAAAAAA","tag":"AAAAAAAAAAAAAAAAAAAAAA==","key_id":"k","protocol_version":1}}' \
  http://127.0.0.1:8080/api/v1/messages/$MESSAGE_A
```

## Current Review Notes

Handlers call service-layer permission checks for guild/channel/message flows. Keep these checks in services, not only handlers.

## Recommendation

- Add DB-backed integration tests for cross-user guild, channel, message, and friends access.
- Ensure every direct object route has an ownership or membership check.
