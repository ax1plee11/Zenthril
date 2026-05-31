-- Migration 010: RBAC v2 - multiple roles per guild member + permission-based checks.
-- Backward compatible: guild_members.role_id is kept for older code and data.

BEGIN;

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS position INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'roles_level_range'
    ) THEN
        ALTER TABLE roles
            ADD CONSTRAINT roles_level_range CHECK (level >= 0 AND level <= 1000);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_guild_id_id
    ON roles(guild_id, id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_guild_lower_name
    ON roles(guild_id, LOWER(name));

CREATE INDEX IF NOT EXISTS idx_roles_guild_level
    ON roles(guild_id, level DESC);

CREATE TABLE IF NOT EXISTS guild_member_roles (
    guild_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (guild_id, user_id, role_id),

    CONSTRAINT fk_guild_member_roles_member
        FOREIGN KEY (guild_id, user_id)
        REFERENCES guild_members(guild_id, user_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_guild_member_roles_role
        FOREIGN KEY (guild_id, role_id)
        REFERENCES roles(guild_id, id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_guild_member_roles_user
    ON guild_member_roles(user_id);

CREATE INDEX IF NOT EXISTS idx_guild_member_roles_role
    ON guild_member_roles(role_id);

CREATE INDEX IF NOT EXISTS idx_guild_member_roles_guild_user
    ON guild_member_roles(guild_id, user_id);

INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
SELECT gm.guild_id, gm.user_id, gm.role_id, g.owner_id
FROM guild_members gm
JOIN guilds g ON g.id = gm.guild_id
WHERE gm.role_id IS NOT NULL
ON CONFLICT DO NOTHING;

UPDATE roles
SET is_system = TRUE
WHERE LOWER(name) IN ('owner', 'member')
   OR level >= 100;

INSERT INTO roles (guild_id, name, level, permissions, is_system, position)
SELECT g.id, 'Member', 0, 0, TRUE, 0
FROM guilds g
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.guild_id = g.id AND LOWER(r.name) = 'member'
)
ON CONFLICT DO NOTHING;

INSERT INTO roles (guild_id, name, level, permissions, is_system, position)
SELECT g.id, 'Owner', 100, 9223372036854775807, TRUE, 1000
FROM guilds g
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.guild_id = g.id AND LOWER(r.name) = 'owner'
)
ON CONFLICT DO NOTHING;

INSERT INTO guild_members (guild_id, user_id, role_id, banned)
SELECT g.id, g.owner_id, r.id, FALSE
FROM guilds g
JOIN roles r ON r.guild_id = g.id AND LOWER(r.name) = 'owner'
ON CONFLICT (guild_id, user_id) DO UPDATE
SET banned = FALSE,
    role_id = COALESCE(guild_members.role_id, EXCLUDED.role_id);

INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
SELECT g.id, g.owner_id, r.id, g.owner_id
FROM guilds g
JOIN roles r ON r.guild_id = g.id AND LOWER(r.name) = 'owner'
ON CONFLICT DO NOTHING;

INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
SELECT gm.guild_id, gm.user_id, r.id, g.owner_id
FROM guild_members gm
JOIN guilds g ON g.id = gm.guild_id
JOIN roles r ON r.guild_id = gm.guild_id AND LOWER(r.name) = 'member'
WHERE gm.banned = FALSE
  AND NOT EXISTS (
      SELECT 1
      FROM guild_member_roles gmr
      WHERE gmr.guild_id = gm.guild_id
        AND gmr.user_id = gm.user_id
  )
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS role_audit_log (
    id BIGSERIAL PRIMARY KEY,
    guild_id UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_role_audit_log_guild_created
    ON role_audit_log(guild_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_role_audit_log_actor_created
    ON role_audit_log(actor_id, created_at DESC);

COMMIT;
