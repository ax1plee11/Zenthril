# Zenthril Security Audit Notes

This directory contains security review notes for the Zenthril alpha project.

Executable proof-of-concept scripts are intentionally not published in this
repository. Security regressions should be covered by normal unit,
integration, and CI checks instead of shipping attack tooling in the public
source tree.

## Safety Rules

- Run security testing only against a Zenthril instance you own or have written
  permission to test.
- Do not use this project for spam, fraud, unauthorized access, or service
  disruption.
- Prefer defensive regression tests in `backend/*_test.go` and
  `client/src/**/*.test.ts(x)`.
- Report security issues through the responsible disclosure process in
  `SECURITY.md`.

## Files

- `REPORT.md` - current audit findings and residual risk.

## Defensive Verification

```bash
cd backend
go test ./... -count=1

cd ../client
npm test
npm run lint
```

Expected result for hardened builds: malformed tokens, invalid origins,
oversized envelopes, replayed refresh tokens, and unauthorized resource access
are rejected by regression tests or safe validation errors.
