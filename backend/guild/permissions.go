package guild

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	PermissionViewChannels int64 = 1 << iota
	PermissionSendMessages
	PermissionManageMessages
	PermissionCreateInvite
	PermissionManageChannels
	PermissionManageRoles
	PermissionKickMembers
	PermissionBanMembers
	PermissionModerateMembers
	PermissionManageGuild
	PermissionViewAuditLog
)

const AllPermissions int64 = PermissionViewChannels |
	PermissionSendMessages |
	PermissionManageMessages |
	PermissionCreateInvite |
	PermissionManageChannels |
	PermissionManageRoles |
	PermissionKickMembers |
	PermissionBanMembers |
	PermissionModerateMembers |
	PermissionManageGuild |
	PermissionViewAuditLog

const DangerousPermissions int64 = PermissionManageRoles |
	PermissionKickMembers |
	PermissionBanMembers |
	PermissionManageGuild

func (s *Service) IsGuildOwner(ctx context.Context, guildID, userID uuid.UUID) (bool, error) {
	if s.IsSuperAdmin(userID) {
		return true, nil
	}
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM guilds WHERE id = $1 AND owner_id = $2)`,
		guildID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check guild owner: %w", err)
	}
	return ok, nil
}

func (s *Service) GetEffectivePermissions(ctx context.Context, guildID, userID uuid.UUID) (int64, error) {
	if s.IsSuperAdmin(userID) {
		return AllPermissions, nil
	}
	isOwner, err := s.IsGuildOwner(ctx, guildID, userID)
	if err != nil {
		return 0, err
	}
	if isOwner {
		return AllPermissions, nil
	}

	var banned bool
	err = s.db.QueryRow(ctx,
		`SELECT banned FROM guild_members WHERE guild_id = $1 AND user_id = $2`,
		guildID, userID,
	).Scan(&banned)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrForbidden
		}
		return 0, fmt.Errorf("query membership: %w", err)
	}
	if banned {
		return 0, ErrForbidden
	}

	var permissions int64
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(bit_or(r.permissions), 0)
		 FROM guild_member_roles gmr
		 JOIN roles r ON r.id = gmr.role_id AND r.guild_id = gmr.guild_id
		 WHERE gmr.guild_id = $1 AND gmr.user_id = $2`,
		guildID, userID,
	).Scan(&permissions)
	if err != nil {
		return 0, fmt.Errorf("query effective permissions: %w", err)
	}
	return permissions, nil
}

func (s *Service) HasPermission(ctx context.Context, guildID, userID uuid.UUID, permission int64) (bool, error) {
	permissions, err := s.GetEffectivePermissions(ctx, guildID, userID)
	if err != nil {
		return false, err
	}
	return permissions&permission == permission, nil
}

func (s *Service) RequirePermission(ctx context.Context, guildID, userID uuid.UUID, permission int64) error {
	ok, err := s.HasPermission(ctx, guildID, userID, permission)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) GetHighestRoleLevel(ctx context.Context, guildID, userID uuid.UUID) (int, error) {
	if s.IsSuperAdmin(userID) {
		return RoleLevelOwner, nil
	}
	isOwner, err := s.IsGuildOwner(ctx, guildID, userID)
	if err != nil {
		return 0, err
	}
	if isOwner {
		return RoleLevelOwner, nil
	}

	var level int
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(r.level), 0)
		 FROM guild_member_roles gmr
		 JOIN roles r ON r.id = gmr.role_id AND r.guild_id = gmr.guild_id
		 JOIN guild_members gm ON gm.guild_id = gmr.guild_id AND gm.user_id = gmr.user_id
		 WHERE gmr.guild_id = $1 AND gmr.user_id = $2 AND gm.banned = FALSE`,
		guildID, userID,
	).Scan(&level)
	if err != nil {
		return 0, fmt.Errorf("query highest role level: %w", err)
	}
	return level, nil
}

func (s *Service) CanManageRole(ctx context.Context, guildID, requesterID, roleID uuid.UUID) error {
	if err := s.RequirePermission(ctx, guildID, requesterID, PermissionManageRoles); err != nil {
		return err
	}

	var roleLevel int
	var isSystem bool
	err := s.db.QueryRow(ctx,
		`SELECT level, is_system FROM roles WHERE guild_id = $1 AND id = $2`,
		guildID, roleID,
	).Scan(&roleLevel, &isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("query target role: %w", err)
	}

	if s.IsSuperAdmin(requesterID) {
		return nil
	}
	isOwner, err := s.IsGuildOwner(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}
	if isSystem {
		return ErrForbidden
	}

	requesterLevel, err := s.GetHighestRoleLevel(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if roleLevel >= requesterLevel {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CanGrantPermissions(ctx context.Context, guildID, requesterID uuid.UUID, requestedPermissions int64) error {
	isOwner, err := s.IsGuildOwner(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}
	requesterPermissions, err := s.GetEffectivePermissions(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if requestedPermissions&^requesterPermissions != 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CanAssignRole(ctx context.Context, guildID, requesterID, targetUserID, roleID uuid.UUID) error {
	if err := s.CanManageRole(ctx, guildID, requesterID, roleID); err != nil {
		return err
	}
	if err := s.requireMember(ctx, guildID, targetUserID); err != nil {
		return ErrNotFound
	}

	var roleLevel int
	var isSystem bool
	err := s.db.QueryRow(ctx,
		`SELECT level, is_system FROM roles WHERE guild_id = $1 AND id = $2`,
		guildID, roleID,
	).Scan(&roleLevel, &isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("query assign role: %w", err)
	}
	// SECURITY: the "god role" is not a guild role. It exists only through
	// ADMIN_USER_IDS/SuperAdmin config, so system and owner-level roles cannot
	// be handed out through normal guild role assignment.
	if !s.IsSuperAdmin(requesterID) && (isSystem || roleLevel >= RoleLevelOwner) {
		return ErrForbidden
	}

	isOwner, err := s.IsGuildOwner(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}

	requesterLevel, err := s.GetHighestRoleLevel(ctx, guildID, requesterID)
	if err != nil {
		return err
	}
	if roleLevel >= requesterLevel {
		return ErrForbidden
	}
	return nil
}
