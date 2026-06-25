package guild

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zenthril-backend/models"
)

var (
	ErrNotFound      = errors.New("not_found")
	ErrForbidden     = errors.New("forbidden")
	ErrInviteExpired = errors.New("invite_expired_or_invalid")
	ErrAlreadyMember = errors.New("already_member")
	ErrGuildBanned   = errors.New("guild_banned")
)

const (
	RoleLevelMember    = 0
	RoleLevelModerator = 10
	RoleLevelAdmin     = 50
	RoleLevelOwner     = 100
)

type Service struct {
	db           *pgxpool.Pool
	nodeID       string
	stateChecker GuildStateChecker
	superAdmins  map[uuid.UUID]struct{}
}

func NewService(db *pgxpool.Pool, nodeID string) *Service {
	return &Service{db: db, nodeID: nodeID, superAdmins: make(map[uuid.UUID]struct{})}
}

func (s *Service) SetSuperAdmins(userIDs []string) {
	s.superAdmins = make(map[uuid.UUID]struct{}, len(userIDs))
	for _, raw := range userIDs {
		id, err := uuid.Parse(raw)
		if err == nil {
			s.superAdmins[id] = struct{}{}
		}
	}
}

func (s *Service) IsSuperAdmin(userID uuid.UUID) bool {
	_, ok := s.superAdmins[userID]
	return ok
}

func (s *Service) CreateGuild(ctx context.Context, ownerID, name string) (*models.Guild, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var guild models.Guild
	err = tx.QueryRow(ctx,
		`INSERT INTO guilds (name, owner_id, node_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, owner_id, node_id, created_at`,
		name, ownerUUID, s.nodeID,
	).Scan(&guild.ID, &guild.Name, &guild.OwnerID, &guild.NodeID, &guild.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert guild: %w", err)
	}

	var roleID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO roles (guild_id, name, level, permissions, is_system, position)
		 VALUES ($1, 'Owner', $2, $3, TRUE, 1000)
		 RETURNING id`,
		guild.ID, RoleLevelOwner, AllPermissions,
	).Scan(&roleID)
	if err != nil {
		return nil, fmt.Errorf("insert owner role: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO roles (guild_id, name, level, permissions, is_system, position)
		 VALUES ($1, 'Member', $2, 0, TRUE, 0)`,
		guild.ID, RoleLevelMember,
	)
	if err != nil {
		return nil, fmt.Errorf("insert member role: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO guild_members (guild_id, user_id, role_id)
		 VALUES ($1, $2, $3)`,
		guild.ID, ownerUUID, roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert guild member: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
		 VALUES ($1, $2, $3, $2)
		 ON CONFLICT DO NOTHING`,
		guild.ID, ownerUUID, roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert owner role assignment: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO channels (guild_id, name, type, position)
		 VALUES
			($1, 'general', 'text', 0),
			($1, 'voice', 'voice', 1)`,
		guild.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert default channels: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &guild, nil
}

func (s *Service) GetUserGuilds(ctx context.Context, userID string) ([]models.Guild, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT g.id, g.name, g.owner_id, g.node_id, g.created_at
		 FROM guilds g
		 JOIN guild_members gm ON gm.guild_id = g.id
		 WHERE gm.user_id = $1 AND gm.banned = FALSE
		 ORDER BY g.created_at ASC`,
		userUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query guilds: %w", err)
	}
	defer rows.Close()

	var guilds []models.Guild
	for rows.Next() {
		var g models.Guild
		if err := rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.NodeID, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan guild: %w", err)
		}
		guilds = append(guilds, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if guilds == nil {
		guilds = []models.Guild{}
	}
	return guilds, nil
}

func (s *Service) CreateInvite(ctx context.Context, guildID, createdBy string, expiresIn *int, maxUses *int) (*models.Invite, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	creatorUUID, err := uuid.Parse(createdBy)
	if err != nil {
		return nil, fmt.Errorf("invalid creator id: %w", err)
	}

	if err := s.RequirePermission(ctx, guildUUID, creatorUUID, PermissionCreateInvite); err != nil {
		return nil, ErrForbidden
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	var expiresAt *time.Time
	if expiresIn != nil && *expiresIn > 0 {
		t := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		expiresAt = &t
	}

	invite := &models.Invite{
		Code:      code,
		GuildID:   guildUUID,
		CreatedBy: creatorUUID,
		ExpiresAt: expiresAt,
		MaxUses:   maxUses,
		Uses:      0,
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO invites (code, guild_id, created_by, expires_at, max_uses, uses)
		 VALUES ($1, $2, $3, $4, $5, 0)`,
		invite.Code, invite.GuildID, invite.CreatedBy, invite.ExpiresAt, invite.MaxUses,
	)
	if err != nil {
		return nil, fmt.Errorf("insert invite: %w", err)
	}

	return invite, nil
}

func (s *Service) JoinByInvite(ctx context.Context, userID, code string) (*models.Guild, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var invite models.Invite
	err = tx.QueryRow(ctx,
		`SELECT code, guild_id, created_by, expires_at, max_uses, uses
		 FROM invites WHERE code = $1 FOR UPDATE`,
		code,
	).Scan(&invite.Code, &invite.GuildID, &invite.CreatedBy, &invite.ExpiresAt, &invite.MaxUses, &invite.Uses)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInviteExpired
		}
		return nil, fmt.Errorf("query invite: %w", err)
	}

	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	if invite.MaxUses != nil && invite.Uses >= *invite.MaxUses {
		return nil, ErrInviteExpired
	}

	var memberRoleID *uuid.UUID
	var roleID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM roles WHERE guild_id = $1 AND level = $2 LIMIT 1`,
		invite.GuildID, RoleLevelMember,
	).Scan(&roleID)
	if err == nil {
		memberRoleID = &roleID
	}

	var joined int
	err = tx.QueryRow(ctx,
		`INSERT INTO guild_members (guild_id, user_id, role_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (guild_id, user_id) DO UPDATE
		 SET role_id = COALESCE(guild_members.role_id, EXCLUDED.role_id)
		 WHERE guild_members.banned = FALSE
		 RETURNING 1`,
		invite.GuildID, userUUID, memberRoleID,
	).Scan(&joined)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// SECURITY: invites must never clear an existing guild ban.
			return nil, ErrGuildBanned
		}
		return nil, fmt.Errorf("insert member: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE invites SET uses = uses + 1 WHERE code = $1`,
		code,
	)
	if err != nil {
		return nil, fmt.Errorf("update invite uses: %w", err)
	}

	var guild models.Guild
	err = tx.QueryRow(ctx,
		`SELECT id, name, owner_id, node_id, created_at FROM guilds WHERE id = $1`,
		invite.GuildID,
	).Scan(&guild.ID, &guild.Name, &guild.OwnerID, &guild.NodeID, &guild.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query guild: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if memberRoleID != nil {
		_, _ = s.db.Exec(ctx,
			`INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT DO NOTHING`,
			invite.GuildID, userUUID, *memberRoleID, invite.CreatedBy,
		)
	}

	return &guild, nil
}

func (s *Service) RemoveMember(ctx context.Context, guildID, requesterID, targetUserID string) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}

	if s.stateChecker != nil {
		canLeave, reason := s.stateChecker.CanUserLeaveGuild(guildID, targetUserID)
		if !canLeave {
			return fmt.Errorf("%w: %s", ErrGuildLocked, reason)
		}
	}

	if err := s.RequirePermission(ctx, guildUUID, requesterUUID, PermissionKickMembers); err != nil {
		return ErrForbidden
	}

	targetOwner, err := s.IsGuildOwner(ctx, guildUUID, targetUUID)
	if err != nil {
		return err
	}
	if targetOwner {
		return ErrForbidden
	}

	result, err := s.db.Exec(ctx,
		`DELETE FROM guild_members WHERE guild_id = $1 AND user_id = $2`,
		guildUUID, targetUUID,
	)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Service) CreateChannel(ctx context.Context, guildID, requesterID, name, channelType string) (*models.Channel, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}

	if err := s.RequirePermission(ctx, guildUUID, requesterUUID, PermissionManageChannels); err != nil {
		return nil, ErrForbidden
	}

	if channelType != "text" && channelType != "voice" {
		return nil, fmt.Errorf("invalid channel type: must be 'text' or 'voice'")
	}

	var ch models.Channel
	err = s.db.QueryRow(ctx,
		`INSERT INTO channels (guild_id, name, type)
		 VALUES ($1, $2, $3)
		 RETURNING id, guild_id, name, type, position, created_at`,
		guildUUID, name, channelType,
	).Scan(&ch.ID, &ch.GuildID, &ch.Name, &ch.Type, &ch.Position, &ch.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert channel: %w", err)
	}

	return &ch, nil
}

func (s *Service) GetGuildChannels(ctx context.Context, guildID, userID string) ([]models.Channel, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	if err := s.requireMember(ctx, guildUUID, userUUID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, guild_id, name, type, position, created_at
		 FROM channels WHERE guild_id = $1
		 ORDER BY position ASC, created_at ASC`,
		guildUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query channels: %w", err)
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var ch models.Channel
		if err := rows.Scan(&ch.ID, &ch.GuildID, &ch.Name, &ch.Type, &ch.Position, &ch.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if channels == nil {
		channels = []models.Channel{}
	}
	return channels, nil
}

// GuildMember is the public projection of a guild member returned by the API.
type GuildMember struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (s *Service) GetGuildMembers(ctx context.Context, guildID, requesterID string) ([]GuildMember, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}

	if err := s.requireMember(ctx, guildUUID, requesterUUID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT u.id, u.username
		 FROM guild_members gm
		 JOIN users u ON u.id = gm.user_id
		 WHERE gm.guild_id = $1 AND gm.banned = FALSE
		 ORDER BY u.username ASC`,
		guildUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	members := []GuildMember{}
	for rows.Next() {
		var m GuildMember
		if err := rows.Scan(&m.ID, &m.Username); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("members rows: %w", err)
	}
	return members, nil
}

func (s *Service) GetUsername(ctx context.Context, userID string) (string, error) {
	uUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	var username string
	err = s.db.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, uUUID).Scan(&username)
	return username, err
}

func (s *Service) UserHasChannelAccess(ctx context.Context, userID, channelID string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user id: %w", err)
	}
	chUUID, err := uuid.Parse(channelID)
	if err != nil {
		return false, fmt.Errorf("invalid channel id: %w", err)
	}
	if s.IsSuperAdmin(userUUID) {
		return true, nil
	}

	var ok bool
	err = s.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM channels c
			INNER JOIN guild_members gm ON gm.guild_id = c.guild_id
				AND gm.user_id = $2 AND gm.banned = FALSE
			WHERE c.id = $1
		)`,
		chUUID, userUUID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("channel access: %w", err)
	}
	return ok, nil
}

func (s *Service) requireMember(ctx context.Context, guildID, userID uuid.UUID) error {
	if s.IsSuperAdmin(userID) {
		return nil
	}
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM guild_members
			WHERE guild_id = $1 AND user_id = $2 AND banned = FALSE
		)`,
		guildID, userID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check member: %w", err)
	}
	if !exists {
		return ErrForbidden
	}
	return nil
}

func (s *Service) getMemberLevel(ctx context.Context, guildID, userID uuid.UUID) (int, error) {
	return s.GetHighestRoleLevel(ctx, guildID, userID)
}

func (s *Service) CreateRole(ctx context.Context, guildID, requesterID, name string, permissions int64) (*models.Role, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}

	if err := s.RequirePermission(ctx, guildUUID, requesterUUID, PermissionManageRoles); err != nil {
		return nil, ErrForbidden
	}
	if err := s.CanGrantPermissions(ctx, guildUUID, requesterUUID, permissions); err != nil {
		return nil, ErrForbidden
	}
	level, err := s.GetHighestRoleLevel(ctx, guildUUID, requesterUUID)
	if err != nil {
		return nil, ErrForbidden
	}
	newRoleLevel := level - 1
	if newRoleLevel < RoleLevelMember {
		newRoleLevel = RoleLevelMember
	}

	var role models.Role
	err = s.db.QueryRow(ctx,
		`INSERT INTO roles (guild_id, name, level, permissions, is_system, position)
		 VALUES ($1, $2, $3, $4, FALSE, $3)
		 RETURNING id, guild_id, name, level, permissions, description, color, position, is_system, created_at, updated_at`,
		guildUUID, name, newRoleLevel, permissions,
	).Scan(&role.ID, &role.GuildID, &role.Name, &role.Level, &role.Permissions, &role.Description, &role.Color, &role.Position, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert role: %w", err)
	}

	return &role, nil
}

func (s *Service) ListRoles(ctx context.Context, guildID, requesterID string) ([]models.Role, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}
	if err := s.requireMember(ctx, guildUUID, requesterUUID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, guild_id, name, level, permissions, description, color, position, is_system, created_at, updated_at
		 FROM roles
		 WHERE guild_id = $1
		 ORDER BY position DESC, level DESC, name ASC`,
		guildUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	roles := []models.Role{}
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.GuildID, &role.Name, &role.Level, &role.Permissions, &role.Description, &role.Color, &role.Position, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("roles rows: %w", err)
	}
	return roles, nil
}

func (s *Service) UpdateRole(ctx context.Context, guildID, requesterID, roleID string, name *string, level *int, permissions *int64, description *string, color *string, position *int) (*models.Role, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role id: %w", err)
	}
	if err := s.CanManageRole(ctx, guildUUID, requesterUUID, roleUUID); err != nil {
		return nil, err
	}

	var current models.Role
	err = s.db.QueryRow(ctx,
		`SELECT id, guild_id, name, level, permissions, description, color, position, is_system, created_at, updated_at
		 FROM roles WHERE guild_id = $1 AND id = $2`,
		guildUUID, roleUUID,
	).Scan(&current.ID, &current.GuildID, &current.Name, &current.Level, &current.Permissions, &current.Description, &current.Color, &current.Position, &current.IsSystem, &current.CreatedAt, &current.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load role: %w", err)
	}
	if current.IsSystem {
		return nil, ErrForbidden
	}

	nextName := current.Name
	nextLevel := current.Level
	nextPermissions := current.Permissions
	nextDescription := current.Description
	nextColor := current.Color
	nextPosition := current.Position
	if name != nil {
		nextName = *name
	}
	if level != nil {
		nextLevel = *level
	}
	if permissions != nil {
		nextPermissions = *permissions
	}
	if description != nil {
		nextDescription = *description
	}
	if color != nil {
		nextColor = *color
	}
	if position != nil {
		nextPosition = *position
	}
	if nextLevel < RoleLevelMember || nextLevel >= RoleLevelOwner {
		return nil, ErrForbidden
	}
	requesterLevel, err := s.GetHighestRoleLevel(ctx, guildUUID, requesterUUID)
	if err != nil {
		return nil, ErrForbidden
	}
	isOwner, err := s.IsGuildOwner(ctx, guildUUID, requesterUUID)
	if err != nil {
		return nil, err
	}
	if !isOwner && nextLevel >= requesterLevel {
		return nil, ErrForbidden
	}
	if err := s.CanGrantPermissions(ctx, guildUUID, requesterUUID, nextPermissions); err != nil {
		return nil, ErrForbidden
	}

	var role models.Role
	err = s.db.QueryRow(ctx,
		`UPDATE roles
		 SET name = $1, level = $2, permissions = $3, description = $4, color = $5, position = $6, updated_at = NOW()
		 WHERE guild_id = $7 AND id = $8
		 RETURNING id, guild_id, name, level, permissions, description, color, position, is_system, created_at, updated_at`,
		nextName, nextLevel, nextPermissions, nextDescription, nextColor, nextPosition, guildUUID, roleUUID,
	).Scan(&role.ID, &role.GuildID, &role.Name, &role.Level, &role.Permissions, &role.Description, &role.Color, &role.Position, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}
	_, _ = s.db.Exec(ctx,
		`INSERT INTO role_audit_log (guild_id, actor_id, role_id, action)
		 VALUES ($1, $2, $3, 'role_updated')`,
		guildUUID, requesterUUID, roleUUID,
	)
	return &role, nil
}

func (s *Service) DeleteRole(ctx context.Context, guildID, requesterID, roleID string) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role id: %w", err)
	}
	if err := s.CanManageRole(ctx, guildUUID, requesterUUID, roleUUID); err != nil {
		return err
	}
	var isSystem bool
	err = s.db.QueryRow(ctx, `SELECT is_system FROM roles WHERE guild_id = $1 AND id = $2`, guildUUID, roleUUID).Scan(&isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load role: %w", err)
	}
	if isSystem {
		return ErrForbidden
	}
	result, err := s.db.Exec(ctx, `DELETE FROM roles WHERE guild_id = $1 AND id = $2`, guildUUID, roleUUID)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = s.db.Exec(ctx,
		`INSERT INTO role_audit_log (guild_id, actor_id, role_id, action)
		 VALUES ($1, $2, $3, 'role_deleted')`,
		guildUUID, requesterUUID, roleUUID,
	)
	return nil
}

func (s *Service) AssignRole(ctx context.Context, guildID, requesterID, targetUserID, roleID string) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role id: %w", err)
	}

	if err := s.CanAssignRole(ctx, guildUUID, requesterUUID, targetUUID, roleUUID); err != nil {
		return err
	}

	result, err := s.db.Exec(ctx,
		`INSERT INTO guild_member_roles (guild_id, user_id, role_id, assigned_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		guildUUID, targetUUID, roleUUID, requesterUUID,
	)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil
	}

	_, _ = s.db.Exec(ctx,
		`INSERT INTO role_audit_log (guild_id, actor_id, target_user_id, role_id, action)
		 VALUES ($1, $2, $3, $4, 'role_assigned')`,
		guildUUID, requesterUUID, targetUUID, roleUUID,
	)
	_, _ = s.db.Exec(ctx,
		`UPDATE guild_members SET role_id = COALESCE(role_id, $1)
		 WHERE guild_id = $2 AND user_id = $3`,
		roleUUID, guildUUID, targetUUID,
	)

	return nil
}

func (s *Service) RemoveRole(ctx context.Context, guildID, requesterID, targetUserID, roleID string) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role id: %w", err)
	}
	if err := s.CanManageRole(ctx, guildUUID, requesterUUID, roleUUID); err != nil {
		return err
	}

	var roleName string
	var isSystem bool
	err = s.db.QueryRow(ctx,
		`SELECT name, is_system FROM roles WHERE guild_id = $1 AND id = $2`,
		guildUUID, roleUUID,
	).Scan(&roleName, &isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load role: %w", err)
	}
	if isSystem {
		return ErrForbidden
	}

	result, err := s.db.Exec(ctx,
		`DELETE FROM guild_member_roles WHERE guild_id = $1 AND user_id = $2 AND role_id = $3`,
		guildUUID, targetUUID, roleUUID,
	)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = s.db.Exec(ctx,
		`UPDATE guild_members SET role_id = NULL
		 WHERE guild_id = $1 AND user_id = $2 AND role_id = $3`,
		guildUUID, targetUUID, roleUUID,
	)
	_, _ = s.db.Exec(ctx,
		`INSERT INTO role_audit_log (guild_id, actor_id, target_user_id, role_id, action, details)
		 VALUES ($1, $2, $3, $4, 'role_removed', jsonb_build_object('role_name', $5))`,
		guildUUID, requesterUUID, targetUUID, roleUUID, roleName,
	)
	return nil
}

func (s *Service) MuteMember(ctx context.Context, guildID, requesterID, targetUserID string, durationSeconds int) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}

	if err := s.RequirePermission(ctx, guildUUID, requesterUUID, PermissionModerateMembers); err != nil {
		return ErrForbidden
	}

	targetOwner, err := s.IsGuildOwner(ctx, guildUUID, targetUUID)
	if err != nil {
		return err
	}
	if targetOwner {
		return ErrForbidden
	}

	muteUntil := time.Now().Add(time.Duration(durationSeconds) * time.Second)
	result, err := s.db.Exec(ctx,
		`UPDATE guild_members SET muted_until = $1
		 WHERE guild_id = $2 AND user_id = $3 AND banned = FALSE`,
		muteUntil, guildUUID, targetUUID,
	)
	if err != nil {
		return fmt.Errorf("mute member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Service) BanMember(ctx context.Context, guildID, requesterID, targetUserID string) error {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}

	if err := s.RequirePermission(ctx, guildUUID, requesterUUID, PermissionBanMembers); err != nil {
		return ErrForbidden
	}

	targetOwner, err := s.IsGuildOwner(ctx, guildUUID, targetUUID)
	if err != nil {
		return err
	}
	if targetOwner {
		return ErrForbidden
	}

	result, err := s.db.Exec(ctx,
		`UPDATE guild_members SET banned = TRUE
		 WHERE guild_id = $1 AND user_id = $2`,
		guildUUID, targetUUID,
	)
	if err != nil {
		return fmt.Errorf("ban member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
