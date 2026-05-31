# PoC 13: WebRTC ICE Leak

**Risk:** Medium.

## Safe PoC

In local voice/P2P testing, inspect ICE candidates in browser devtools or app logs.

Risk indicators:

- `host` candidates exposing local LAN IPs.
- `srflx` candidates exposing public IP through STUN.
- Direct P2P mode enabled when the user expects relay-only privacy.

## Recommendation

- Add a relay-only mode for high-privacy calls.
- Prefer TURN over TLS for privacy-sensitive profiles.
- Warn users that P2P/WebRTC can expose IP metadata.
- Avoid logging full ICE candidate strings in production logs.
