# PoC 08: Message Injection / XSS

**Risk:** High.

## Safe PoC

In a local test account, send plaintext before encryption containing:

```html
<img src=x onerror=alert('xss')>
<script>alert('xss')</script>
javascript:alert('xss')
```

Expected:

- React renders message text as text, not HTML.
- No `dangerouslySetInnerHTML` is used for untrusted message content.
- Links, GIF previews, and rich previews are sanitized before rendering.

## Repository Check

```bash
rg -n "dangerouslySetInnerHTML|innerHTML|insertAdjacentHTML" client/src
```

## Recommendation

- Keep untrusted content in React text nodes.
- If markdown/rich embeds are added later, use a sanitizer with a strict allowlist.
- Add a frontend test that renders malicious message text and asserts no script execution path exists.
