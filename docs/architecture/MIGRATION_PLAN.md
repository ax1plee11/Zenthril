# Zenthril Migration Plan
**Version:** 1.0  
**Status:** DRAFT  
**Generated:** 2026-11-02

---

## 1. Migration Strategy: Strangler Pattern

The project will be modernized gradually using the Strangler Pattern. Existing functionality is preserved while new components are built alongside, tested, and then switched over.

**Rules:**
- Never rewrite everything at once
- Never break existing functionality
- Every new component must have tests before switching
- Every migration has a rollback plan
- System must remain operational throughout

---

## 2. Migration Phases

### Phase 1: Stabilize (Weeks 1-2)

**Goal:** Fix critical defects, make system functional.

| Task | Description | Rollback |
|------|-------------|----------|
| Fix X3DH KDF `info` mismatch | Unify Go and TypeScript KDF input | Revert protocol.go change |
| Fix TOTP secret storage | Store base32 secret instead of SHA-256 hash | Revert service.go change |
| Add rate limiting on TOTP | Apply AuthRateLimit to TOTP routes | Remove middleware |
| Add missing tests | Tests for federation (done), guild, message | N/A (additive) |
| Fix moderation IP logging | Already fixed (slog instead of log.Printf) | N/A |
| Remove backend-2.0 typo dir | Already removed | N/A |

**Exit Criteria:**
- `go test ./...` passes
- `npm test` passes
- TOTP end-to-end flow works
- E2EE cross-language keys match

### Phase 2: Repository Pattern (Weeks 3-4)

**Goal:** Introduce data access abstraction.

| Task | Description | Rollback |
|------|-------------|----------|
| Create `internal/repository/` | Repository interfaces for User, Device, Message, Guild | Keep old direct DB access |
| Refactor `auth` service | Use UserRepository | Revert to direct pgxpool |
| Refactor `device` service | Use DeviceRepository | Revert to direct pgxpool |
| Add repository tests | Test repository implementations | N/A |

**Exit Criteria:**
- Repository interfaces defined
- At least auth and device services use repositories
- All repository tests pass

### Phase 3: Consolidate Entry Points (Weeks 5-6)

**Goal:** Unify backend entry points.

| Task | Description | Rollback |
|------|-------------|----------|
| Wire full business logic into cmd/api | Add message, guild, friends handlers to fx container | Keep legacy main.go running |
| Run both entry points in parallel | Compare behavior | Stop cmd/api |
| Deprecate legacy main.go | Mark as deprecated, remove routes migrated | Re-enable main.go |
| Remove legacy main.go | After full verification | Restore from git |

**Exit Criteria:**
- cmd/api handles all routes that main.go handles
- Behavior identical between entry points
- Legacy main.go can be removed

### Phase 4: backend-2.0 Decision (Weeks 7-8)

**Goal:** Resolve backend-2.0 status.

**Option A: Complete backend-2.0**
- Implement all domain modules
- Wire fx container
- Add tests
- Deploy as replacement

**Option B: Consolidate into backend 1.x**
- Move backend-2.0 concepts into backend 1.x
- Delete backend-2.0 directory
- Single codebase

**Recommendation:** Option B (consolidate) to avoid maintaining two codebases.

---

## 3. Database Migration Plan

### Migration Rules

1. **Expand-migrate-contract:** Add new columns/tables before removing old ones
2. **Backward-compatible:** All migrations must work with current code version
3. **Rollback-capable:** Every migration has a down migration
4. **Tested:** Every migration tested with up/down cycle

### Planned Migrations

| Migration | Description | Status |
|-----------|-------------|--------|
| 016_password_reset_tokens | Password reset token table | Planned |
| 017_mfa_recovery_codes | MFA recovery codes table | Planned |
| 018_security_log_retention | Retention policy for security_log | Planned |
| 019_message_reactions | Message reactions table | Planned |
| 020_message_edits_history | Message edit history | Planned |

---

## 4. API Versioning Plan

### Current API: `/api/v1`

**Versioning Strategy:**
- URL-based versioning: `/api/v1`, `/api/v2`
- Deprecation headers: `Deprecation`, `Sunset`
- Breaking changes require new version
- Old versions supported for minimum 6 months

### Planned API Changes (v2)

| Endpoint | v1 | v2 | Change |
|----------|----|----|--------|
| POST /auth/register | ✅ | ✅ | Add password strength metadata |
| POST /auth/login | ✅ | ✅ | Add device fingerprint |
| GET /auth/me | ✅ | ✅ | Add MFA status |
| POST /messages | ✅ | ✅ | Add reply_to, thread_id |
| GET /messages | ✅ | ✅ | Add cursor-based pagination |

---

## 5. Rollback Procedures

### Code Rollback
```bash
# Revert to previous commit
git revert <commit-hash>

# Or checkout previous version
git checkout <previous-tag>
```

### Database Rollback
```bash
# Run down migration
psql -f migrations/0XX_down.sql
```

### Configuration Rollback
```bash
# Restore previous .env
cp .env.backup .env
```

### Feature Flag Rollback
```bash
# Disable feature via environment variable
FEATURE_NEW_MESSAGING=false go run .
```

---

## 6. Migration Tracking

**File:** `docs/PROJECT_STATUS.md`

Updated after each migration phase with:
- Completed tasks
- In-progress tasks
- Blocked tasks
- Rollback status
- Test results

---

## 7. Definition of Done per Phase

- [ ] All new code has tests
- [ ] All existing tests pass
- [ ] Go build succeeds
- Go vet passes
- TypeScript typecheck passes
- TypeScript lint passes
- Cross-language E2EE test passes (if applicable)
- Documentation updated
- Rollback plan documented

---
