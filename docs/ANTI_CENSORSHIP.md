# Anti-Censorship And Resilience Plan

Zenthril is still an alpha project. This document describes the resilience
architecture being built; it is not a claim that the messenger is unblockable.

## Current Status

Implemented foundation:

- Client-side multi-server list loaded from `client/public/servers.json`.
- Optional build-time `VITE_SERVER_LIST` with JSON array of API origins.
- User-selectable server switcher in the login screen and authenticated UI.
- Custom server entry stored locally by the client.
- Cached server list fallback when `servers.json` cannot be fetched.
- Network-error fallback across configured servers.
- Mirror entries in `servers.json` are promoted into normal fallback servers.
- Basic client health check for each server.
- DNS-over-HTTPS bootstrap can discover alternate server-list URLs.
- `.onion` server entries are represented as Tor transport targets.
- Experimental WebSocket JSON padding camouflage can be enabled with `VITE_WS_CAMOUFLAGE=json-padding-v1`.
- Bridge metadata and self-healing fallback planning are available for primary,
  mirror, bridge, Tor, and P2P fallback attempts.
- Client Stealth Mode controls WebSocket padding and send jitter.

## Server List Format

```json
{
  "version": 1,
  "updated_at": "2026-05-29",
  "servers": [
    {
      "id": "primary",
      "name": "Primary node",
      "api_base": "https://api.example.com",
      "ws_base": "wss://api.example.com",
      "health_path": "/health",
      "transport": "direct",
      "onion": false,
      "bridges": [
        "bridge-a.example"
      ],
      "mirrors": [
        "https://mirror-1.example.net",
        "https://mirror-2.example.org"
      ]
    }
  ]
}
```

`mirrors` are API origins. The client derives `ws://` or `wss://` from each
mirror origin.

## DoH And Tor Status

Browser and Tauri webview clients cannot force `fetch` or `WebSocket` to use a
custom DNS answer. DoH is therefore used only as bootstrap metadata for finding
alternate server-list URLs. Full DNS routing still depends on the operating
system, browser engine, proxy, or Tor client.

`.onion` entries are supported as server metadata and URLs. They require an
external Tor-capable environment or proxy configuration. Zenthril does not yet
bundle Tor.

DoT is documented as a deployment/operator option for resolvers and proxies.
It is not directly implemented inside the browser client because JavaScript
clients do not expose raw TLS DNS sockets.

## WebSocket Camouflage

`VITE_WS_CAMOUFLAGE=json-padding-v1` wraps outbound client events in a padded
JSON frame. This can reduce the stability of simple payload fingerprints, but
it does not hide IP addresses, TLS metadata, timing, packet sizes, or server
names. It is a transport experiment, not a privacy guarantee.

Stealth Mode can also be enabled from the client server settings:

- `off`: normal transport behavior.
- `balanced`: enables JSON padding with moderate random delay.
- `strict`: increases padding and timing jitter at a higher latency/battery cost.

## Domain Fronting And Mimicry

Zenthril does not impersonate unrelated services such as YouTube, Google Meet,
Telegram, or other platforms. Domain fronting, when used, must be limited to
infrastructure controlled by the operator and allowed by the CDN/provider terms.

`VITE_ALLOW_DOMAIN_FRONTING=true` and `VITE_FRONTING_HOST` are reserved for
authorized deployments. They are not a bypass guarantee and do not change TLS
fingerprints from browser or Tauri webview networking stacks.

## DoT, DNSCrypt, QUIC, And I2P

DoT and DNSCrypt are deployment/proxy responsibilities in the current alpha
client. Browser JavaScript cannot open raw DNS-over-TLS or DNSCrypt sockets.

QUIC and I2P are planned transports, not implemented transports. They require
separate client runtime support, abuse controls, and compatibility testing.

## Threat Model

This layer helps when:

- A primary domain is unavailable.
- A primary IP address is blocked.
- A server is temporarily offline.
- A user needs to manually switch to a known reachable mirror.

This layer does **not** yet solve:

- Account and message portability across unrelated servers.
- Server-list authenticity without a signed list.
- Global blocking of all known mirrors.
- Traffic fingerprinting by an advanced network censor.
- Full federation or P2P delivery.
- Built-in Tor transport.
- Automatic bridge message relay.
- TLS fingerprint randomization from browser/Tauri networking APIs.

## Three-Month Resilience Roadmap

### Month 1: Foundation

- Complete security hardening for browser-facing and operational endpoints.
- Add multi-server client architecture.
- Add dynamic `servers.json`, local cache, custom servers, and mirror fallback.
- Add user-facing server switcher.
- Keep all federation and E2EE claims alpha-level and honest.

### Month 2: Anti-Censorship And E2EE Upgrade

- Add DNS-over-HTTPS bootstrap for server-list discovery.
- Add optional `.onion` server entries and external Tor proxy mode.
- Add basic WebSocket camouflage as a transport experiment, not a security guarantee.
- Continue X3DH and Double Ratchet integration with test vectors.
- Add safety-number / QR verification UX.
- Move desktop private key material toward OS keychain or Stronghold.

### Month 3: P2P And Federation Foundation

- WebRTC direct-message fallback prototype.
- Minimal server-to-server federation inbox for encrypted envelopes.
- Bridge nodes and explicit fallback routing metadata.
- Signed server lists and trust roots.
- Self-healing reconnect planning across primary, mirrors, bridge nodes, and P2P.

## Required Future Hardening

- Sign `servers.json` and pin maintainer public keys in the client.
- Add server identity keys and certificate transparency-style history for mirrors.
- Avoid logging full server URLs when they may contain sensitive deployment paths.
- Add metrics for fallback attempts without exposing user identity.
- Document Tor and DoH limitations clearly before release.
