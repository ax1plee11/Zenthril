# PoC 11: Supply Chain Review

**Risk:** Medium.

## Safe Checks

Backend:

```bash
cd backend
go test ./... -count=1
go vet ./...
govulncheck ./...
```

Client:

```bash
cd client
npm ci
npm audit --audit-level=high
npm run build
npm run lint
```

Tauri/Rust:

```bash
cd client/src-tauri
cargo check
cargo audit
```

## Recommendation

- Add Dependabot or Renovate.
- Add CodeQL.
- Pin GitHub Actions versions.
- Keep lockfiles committed.
