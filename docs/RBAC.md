# RBAC v2

Zenthril uses an alpha RBAC v2 model for guild-level authorization.

## Goals

- A guild member can have multiple roles.
- Effective permissions are calculated as a bitwise OR of all assigned roles.
- Highest role level is used only for role hierarchy checks.
- Sensitive actions are checked through explicit permissions, not only through role level.
- Legacy `guild_members.role_id` is kept temporarily for backward compatibility.

## Tables

- `roles`: role metadata, hierarchy level, permission bitmask, and `is_system`.
- `guild_members`: legacy membership row and temporary `role_id` compatibility field.
- `guild_member_roles`: many-to-many role assignments.
- `role_audit_log`: audit trail for role assignment/update/delete operations.

## Permissions

Permission constants live in `backend/guild/permissions.go`.

| Permission | Purpose |
| --- | --- |
| `PermissionCreateInvite` | Create guild invites |
| `PermissionManageChannels` | Create/manage channels |
| `PermissionManageRoles` | Create/update/delete roles and assign roles |
| `PermissionKickMembers` | Remove members |
| `PermissionBanMembers` | Ban members |
| `PermissionModerateMembers` | Mute members |
| `PermissionManageGuild` | Guild-level administration |
| `PermissionViewAuditLog` | Read audit log data |

Guild owners receive `AllPermissions` through owner override.

## Hierarchy Rules

- Non-owners cannot manage a role with level greater than or equal to their highest role level.
- Non-owners cannot assign a role with level greater than or equal to their highest role level.
- System roles are protected from deletion and dangerous editing.
- System and owner-level roles cannot be assigned through normal guild role APIs.
- A requester cannot grant permissions they do not already have unless they are the guild owner.
- Banned members have no effective permissions.
- Guild owners cannot be kicked, banned, or muted by non-owner flows.

## Super Admin / God Role

The developer "god role" is **not** represented by a guild role row and cannot be
granted through the roles API. It is derived only from server-side
`ADMIN_USER_IDS` configuration.

This means:

- users cannot obtain super-admin by creating a high-permission guild role;
- users cannot obtain super-admin by assigning the system `Owner` role;
- `ADMIN_USER_IDS` changes require server configuration access and backend restart;
- super-admin should be used only for development, recovery, and emergency moderation.

## API

Current guild role endpoints:

```text
GET    /api/v1/guilds/{guildId}/roles
POST   /api/v1/guilds/{guildId}/roles
PATCH  /api/v1/guilds/{guildId}/roles/{roleId}
DELETE /api/v1/guilds/{guildId}/roles/{roleId}

PUT    /api/v1/guilds/{guildId}/members/{userId}/roles/{roleId}
DELETE /api/v1/guilds/{guildId}/members/{userId}/roles/{roleId}
```

The old `PATCH /api/v1/guilds/{guildId}/members/{userId}/role` route is kept as a compatibility path, but it now assigns into `guild_member_roles`.

## Migration Notes

Migration `backend/migrations/010_rbac_v2.sql` backfills existing `guild_members.role_id` values into `guild_member_roles`. Do not drop `guild_members.role_id` until at least one later release after all code paths and tests use `guild_member_roles`.
