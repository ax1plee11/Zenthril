# Connectivity Resilience

Zenthril is an alpha self-hosted messenger. This document describes conservative
connectivity and outage-recovery features for administrators and users.

This is not a circumvention feature set. Zenthril does not claim invisible or
indistinguishable traffic.

## Current Status

Implemented foundation:

- Client-side server list loaded from `client/public/servers.json`.
- Optional build-time `VITE_SERVER_LIST` with JSON array of API origins.
- User-selectable server switcher in the login screen and authenticated UI.
- Explicit custom server entry stored locally by the client.
- Cached server list when `servers.json` cannot be fetched.
- Network-error retry across administrator-configured backup endpoints.
- Basic health check for each configured server.
- DNS-over-HTTPS bootstrap can discover alternate server-list URLs when an
  operator configures it.
- Optional WebSocket JSON padding can be enabled for interoperability testing.

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
      "backup_endpoints": [
        "https://backup-1.example.net",
        "https://backup-2.example.org"
      ]
    }
  ]
}
```

`backup_endpoints` contains administrator-configured backup API origins. The
client derives `ws://` or `wss://` from each backup origin.

## Client Behavior

The client supports this sequence:

```text
primary endpoint -> administrator-configured backup endpoint -> manual custom server
```

Direct peer transport is experimental and must remain explicit opt-in. It is not
used as an automatic recovery path.

## DNS Bootstrap Status

Browser and Tauri webview clients cannot force `fetch` or `WebSocket` to use a
custom DNS answer. DoH is therefore used only as optional bootstrap metadata for
finding alternate server-list URLs. Full DNS routing still depends on the
operating system, browser engine, and network configuration.

## WebSocket JSON Padding

`VITE_WS_PADDING=json-padding-v1` wraps outbound client events in a padded JSON
frame. This is an interoperability and measurement experiment. It does not hide
IP addresses, domains, TLS metadata, timing, packet sizes, or server names.

The client exposes a Connectivity Mode setting:

- `off`: normal transport behavior.
- `balanced`: JSON padding with moderate random delay.
- `strict`: more padding and timing jitter at a higher latency/battery cost.

## Threat Model

This layer helps when:

- a primary server is temporarily offline;
- a primary domain is moved by the administrator;
- an operator needs disaster-recovery endpoints;
- a user manually selects a known self-hosted server.

This layer does not provide:

- account and message portability across unrelated servers;
- authenticity for unsigned server lists;
- indistinguishable network traffic;
- automatic direct-peer recovery;
- production federation.

## Required Future Hardening

- Sign `servers.json` and pin maintainer public keys in the client.
- Add server identity keys and server-list history.
- Avoid logging full server URLs when they may contain sensitive deployment paths.
- Add metrics for endpoint retry attempts without exposing user identity.
