package guild

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rbacFixture struct {
	svc        *Service
	pool       *pgxpool.Pool
	guildID    uuid.UUID
	ownerID    uuid.UUID
	adminID    uuid.UUID
	modID      uuid.UUID
	memberID   uuid.UUID
	bannedID   uuid.UUID
	ownerRole  uuid.UUID
	adminRole  uuid.UUID
	modRole    uuid.UUID
	memberRole uuid.UUID
}

func newRBACFixture(t *testing.T) *rbacFixture {
	t.Helper()
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		t.Skip("set TEST_DB_URL to run PostgreSQL-backed RBAC tests")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect root db: %v", err)
	}
	defer root.Close()

	schema := "rbac_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := root.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = root.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
	})

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse db url: %v", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET search_path TO `+schema)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect schema db: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `
CREATE TABLE users (id UUID PRIMARY KEY, username TEXT NOT NULL);
CREATE TABLE guilds (id UUID PRIMARY KEY, owner_id UUID NOT NULL REFERENCES users(id), name TEXT NOT NULL, node_id TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE roles (
	id UUID PRIMARY KEY,
	guild_id UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	level INT NOT NULL,
	permissions BIGINT NOT NULL DEFAULT 0,
	description TEXT NOT NULL DEFAULT '',
	color VARCHAR(16) NOT NULL DEFAULT '',
	position INT NOT NULL DEFAULT 0,
	is_system BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_roles_guild_id_id ON roles(guild_id, id);
CREATE TABLE guild_members (
	guild_id UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id UUID REFERENCES roles(id),
	joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	banned BOOLEAN NOT NULL DEFAULT FALSE,
	muted_until TIMESTAMPTZ,
	PRIMARY KEY (guild_id, user_id)
);
CREATE TABLE guild_member_roles (
	guild_id UUID NOT NULL,
	user_id UUID NOT NULL,
	role_id UUID NOT NULL,
	assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
	assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (guild_id, user_id, role_id),
	FOREIGN KEY (guild_id, user_id) REFERENCES guild_members(guild_id, user_id) ON DELETE CASCADE,
	FOREIGN KEY (guild_id, role_id) REFERENCES roles(guild_id, id) ON DELETE CASCADE
);
CREATE TABLE channels (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	guild_id UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	position INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE role_audit_log (
	id BIGSERIAL PRIMARY KEY,
	guild_id UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
	actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
	target_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
	role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
	action TEXT NOT NULL,
	details JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
	if err != nil {
		t.Fatalf("create test schema tables: %v", err)
	}

	f := &rbacFixture{
		svc:        NewService(pool, "test-node"),
		pool:       pool,
		guildID:    uuid.New(),
		ownerID:    uuid.New(),
		adminID:    uuid.New(),
		modID:      uuid.New(),
		memberID:   uuid.New(),
		bannedID:   uuid.New(),
		ownerRole:  uuid.New(),
		adminRole:  uuid.New(),
		modRole:    uuid.New(),
		memberRole: uuid.New(),
	}
	users := []uuid.UUID{f.ownerID, f.adminID, f.modID, f.memberID, f.bannedID}
	for i, id := range users {
		if _, err := pool.Exec(ctx, `INSERT INTO users (id, username) VALUES ($1, $2)`, id, "u"+uuid.NewString()+string(rune('a'+i))); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO guilds (id, owner_id, name, node_id) VALUES ($1, $2, 'g', 'n')`, f.guildID, f.ownerID); err != nil {
		t.Fatalf("insert guild: %v", err)
	}
	insertRole := func(id uuid.UUID, name string, level int, permissions int64, system bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO roles (id, guild_id, name, level, permissions, is_system) VALUES ($1, $2, $3, $4, $5, $6)`, id, f.guildID, name, level, permissions, system); err != nil {
			t.Fatalf("insert role %s: %v", name, err)
		}
	}
	insertRole(f.ownerRole, "Owner", RoleLevelOwner, AllPermissions, true)
	insertRole(f.adminRole, "Admin", RoleLevelAdmin, PermissionManageChannels|PermissionManageRoles|PermissionModerateMembers|PermissionCreateInvite, false)
	insertRole(f.modRole, "Moderator", RoleLevelModerator, PermissionModerateMembers, false)
	insertRole(f.memberRole, "Member", RoleLevelMember, 0, true)

	addMember := func(userID, roleID uuid.UUID, banned bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO guild_members (guild_id, user_id, role_id, banned) VALUES ($1, $2, $3, $4)`, f.guildID, userID, roleID, banned); err != nil {
			t.Fatalf("insert member: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by) VALUES ($1, $2, $3, $4)`, f.guildID, userID, roleID, f.ownerID); err != nil {
			t.Fatalf("insert member role: %v", err)
		}
	}
	addMember(f.ownerID, f.ownerRole, false)
	addMember(f.adminID, f.adminRole, false)
	addMember(f.modID, f.modRole, false)
	addMember(f.memberID, f.memberRole, false)
	addMember(f.bannedID, f.adminRole, true)
	return f
}

func TestRBACV2OwnerHasAllPermissions(t *testing.T) {
	f := newRBACFixture(t)
	perms, err := f.svc.GetEffectivePermissions(context.Background(), f.guildID, f.ownerID)
	if err != nil {
		t.Fatalf("permissions: %v", err)
	}
	if perms != AllPermissions {
		t.Fatalf("owner permissions = %d, want %d", perms, AllPermissions)
	}
}

func TestRBACV2BannedMemberHasZeroPermissions(t *testing.T) {
	f := newRBACFixture(t)
	if _, err := f.svc.GetEffectivePermissions(context.Background(), f.guildID, f.bannedID); err == nil {
		t.Fatal("expected banned member permissions to fail")
	}
}

func TestRBACV2MemberCannotCreateChannel(t *testing.T) {
	f := newRBACFixture(t)
	_, err := f.svc.CreateChannel(context.Background(), f.guildID.String(), f.memberID.String(), "private", "text")
	if err == nil {
		t.Fatal("expected member create channel to fail")
	}
}

func TestRBACV2AdminWithManageChannelsCanCreateChannel(t *testing.T) {
	f := newRBACFixture(t)
	ch, err := f.svc.CreateChannel(context.Background(), f.guildID.String(), f.adminID.String(), "ops", "text")
	if err != nil {
		t.Fatalf("admin create channel: %v", err)
	}
	if ch.Name != "ops" {
		t.Fatalf("channel name = %q", ch.Name)
	}
}

func TestRBACV2ModeratorCanMuteButCannotBan(t *testing.T) {
	f := newRBACFixture(t)
	if err := f.svc.MuteMember(context.Background(), f.guildID.String(), f.modID.String(), f.memberID.String(), 60); err != nil {
		t.Fatalf("moderator mute: %v", err)
	}
	if err := f.svc.BanMember(context.Background(), f.guildID.String(), f.modID.String(), f.memberID.String()); err == nil {
		t.Fatal("expected moderator ban to fail")
	}
}

func TestRBACV2AdminCannotAssignOwnerRole(t *testing.T) {
	f := newRBACFixture(t)
	err := f.svc.AssignRole(context.Background(), f.guildID.String(), f.adminID.String(), f.memberID.String(), f.ownerRole.String())
	if err == nil {
		t.Fatal("expected admin assigning owner role to fail")
	}
}

func TestRBACV2OwnerCannotAssignSystemOwnerRole(t *testing.T) {
	f := newRBACFixture(t)
	err := f.svc.AssignRole(context.Background(), f.guildID.String(), f.ownerID.String(), f.memberID.String(), f.ownerRole.String())
	if err == nil {
		t.Fatal("expected owner assigning system owner role to fail")
	}
}

func TestRBACV2AdminCannotManageEqualOrHigherRole(t *testing.T) {
	f := newRBACFixture(t)
	err := f.svc.CanManageRole(context.Background(), f.guildID, f.adminID, f.adminRole)
	if err == nil {
		t.Fatal("expected admin managing equal role to fail")
	}
}

func TestRBACV2ManageRolesCannotGrantUnknownPermissions(t *testing.T) {
	f := newRBACFixture(t)
	_, err := f.svc.CreateRole(context.Background(), f.guildID.String(), f.adminID.String(), "Ban helper", PermissionBanMembers)
	if err == nil {
		t.Fatal("expected admin granting missing BanMembers permission to fail")
	}
}

func TestRBACV2BackfillMigrationCopiesLegacyRoleID(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "migrations", "010_rbac_v2.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(data)
	if !strings.Contains(sql, "INSERT INTO guild_member_roles") || !strings.Contains(sql, "gm.role_id") {
		t.Fatal("migration must backfill guild_members.role_id into guild_member_roles")
	}
}

func TestRBACV2SuperAdminHasGodRole(t *testing.T) {
	svc := NewService(nil, "test-node")
	adminID := uuid.New()
	svc.SetSuperAdmins([]string{adminID.String()})

	perms, err := svc.GetEffectivePermissions(context.Background(), uuid.New(), adminID)
	if err != nil {
		t.Fatalf("super admin permissions: %v", err)
	}
	if perms != AllPermissions {
		t.Fatalf("super admin permissions = %d, want %d", perms, AllPermissions)
	}
	level, err := svc.GetHighestRoleLevel(context.Background(), uuid.New(), adminID)
	if err != nil {
		t.Fatalf("super admin level: %v", err)
	}
	if level != RoleLevelOwner {
		t.Fatalf("super admin level = %d, want %d", level, RoleLevelOwner)
	}
	if err := svc.requireMember(context.Background(), uuid.New(), adminID); err != nil {
		t.Fatalf("super admin should bypass membership gate: %v", err)
	}
}

func TestRBACV2SuperAdminCanAssignOwnerRoleForRecovery(t *testing.T) {
	f := newRBACFixture(t)
	superID := uuid.New()
	if _, err := f.pool.Exec(context.Background(), `INSERT INTO users (id, username) VALUES ($1, 'super')`, superID); err != nil {
		t.Fatalf("insert super user: %v", err)
	}
	f.svc.SetSuperAdmins([]string{superID.String()})
	if err := f.svc.AssignRole(context.Background(), f.guildID.String(), superID.String(), f.memberID.String(), f.ownerRole.String()); err != nil {
		t.Fatalf("super admin owner role recovery assignment: %v", err)
	}
}
