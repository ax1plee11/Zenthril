# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x | ✅ |

## Reporting a Vulnerability

If you discover a security vulnerability, please **do not** open a public GitHub issue.

Instead, email: **ax1plee@gmail.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

You will receive a response within 48 hours.

## Security Features

- End-to-end encryption (X25519 + AES-256-GCM)
- Password hashing with Argon2id
- JWT authentication with token blacklisting
- DDoS protection (rate limiting per IP)
- Brute-force protection on login
- TLS support for all connections
- Input validation and parameterized queries (no SQL injection)
