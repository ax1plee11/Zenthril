# Transport Resilience Notes

Zenthril is alpha software. Transport resilience is an outage-recovery layer for
self-hosted deployments, not a privacy or invisibility guarantee.

## Implemented

- Multi-server retry with administrator-configured backup endpoint metadata.
- DNS-over-HTTPS bootstrap for alternate server-list discovery.
- Optional WebSocket JSON padding frames with configurable padding bounds.
- Optional send jitter controlled by Connectivity Mode.

## Not Implemented

- TLS fingerprint randomization.
- Built-in anonymous routing runtime.
- QUIC transport.
- DNSCrypt client.
- Traffic mimicry that claims to be indistinguishable from another product.

## Operator Guidance

- Use domain names and CDN properties you control.
- Sign server lists before distributing them publicly.
- Avoid public claims that traffic is invisible, indistinguishable, or impossible
  to block.
- Measure latency and battery impact before enabling strict padding by default.
