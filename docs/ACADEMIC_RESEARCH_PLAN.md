# Academic Research Plan

Zenthril is both a practical messaging application and an academic engineering project. This document defines the research angle so implementation work, benchmarks, and documentation can support a diploma or master's thesis.

## Working Research Topic

Design and evaluation of a self-hosted secure realtime messaging system with end-to-end encryption foundations, hybrid voice communication, and measurable scalability characteristics.

## Core Research Questions

1. What security controls are required before an alpha E2EE messenger can be responsibly tested by small communities?
2. What are the latency and throughput limits of a Go-based WebSocket messaging backend on a single node?
3. How much overhead do encryption primitives and ratchet-style key evolution add to message processing?
4. Which architecture changes are required to move from a single-node MVP to a multi-instance realtime gateway?
5. What tradeoffs appear when comparing self-hosted messaging systems with Matrix-like, Signal-like, and Discord-like architectures?

## Evaluation Areas

### Security

- CORS and WebSocket Origin validation.
- Token lifetime, refresh flow, and revocation behavior.
- Device key lifecycle and revocation.
- E2EE limitations and future Double Ratchet integration.
- Operational endpoint exposure.

### Performance

- WebSocket throughput.
- P50, P95, and P99 message latency.
- Memory per connection.
- Database insert/select latency.
- Encryption and ratchet operation cost.

### Reliability

- Graceful shutdown behavior.
- Health and readiness checks.
- Redis/PostgreSQL dependency failures.
- Multi-instance gateway behavior after Redis Pub/Sub is introduced.

### Usability

- Device verification UX.
- Recovery after device revocation.
- Clarity of security warnings for alpha users.

## Methodology

Each experiment should include:

- Git commit hash.
- Hardware and operating system.
- Go, Node, Docker, PostgreSQL, Redis, and k6 versions.
- Exact environment variables that affect behavior, with secrets redacted.
- Dataset size and user/message counts.
- Raw output stored under `research/results/`.
- Short interpretation explaining what the numbers mean and what they do not prove.

## Suggested Thesis Structure

1. Introduction and problem statement.
2. Related work: Matrix, Signal Protocol, Discord-like realtime systems.
3. Requirements and threat model.
4. Zenthril architecture and implementation.
5. Security hardening and E2EE design.
6. Performance evaluation.
7. Limitations.
8. Future work.
9. Conclusion.

## Research Ethics And Honesty Rules

- Do not claim Signal-grade security until the complete protocol is implemented and reviewed.
- Do not claim federation until two independent nodes can exchange signed events.
- Publish benchmark conditions together with numbers.
- Keep raw results available and avoid cherry-picking only favorable runs.
- Clearly separate implemented features from roadmap items.
