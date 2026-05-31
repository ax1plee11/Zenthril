# Traffic Obfuscation Notes

Zenthril is alpha software. Traffic obfuscation is a resilience layer, not a
guarantee of invisibility against a capable network censor.

## Implemented

- Multi-server fallback with mirrors and bridge metadata.
- DNS-over-HTTPS bootstrap for alternate server-list discovery.
- `.onion` server metadata for external Tor-capable environments.
- WebSocket JSON padding frames with configurable padding bounds.
- Optional send jitter controlled by Stealth Mode.
- Self-healing fallback planning across primary, mirror, bridge, Tor, and P2P.

## Not Implemented

- TLS fingerprint randomization.
- Built-in Tor or I2P runtime.
- QUIC transport.
- DNSCrypt client.
- Domain fronting against unrelated third-party services.
- Traffic mimicry that claims to be indistinguishable from another product.

## Operator Guidance

- Use domain names and CDN properties you control.
- Keep server-list mirrors signed before distributing them publicly.
- Prefer external Tor/I2P tooling until bundled transports are reviewed.
- Avoid making public claims that traffic is unblockable or undetectable.
- Measure latency and battery impact before enabling strict padding by default.
