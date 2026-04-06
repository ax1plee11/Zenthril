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

	"veltrix-backend/models"
)

var (
	ErrNotFound       = errors.New("not_found")
	ErrForbidden      = errors.New("forbidden")
	ErrInviteExpired  = errors.New("invite_expired_or_invalid")
	ErrAlreadyMember  = errors.New("already_member")
)

// RoleLevel определяет уровни ролей.
const (
	RoleLevelMember    = 0
	RoleLevelModerator = 10
	RoleLevelAdmin     = 50
	RoleLevelOwner     = 100
)

// Service реализует бизнес-логику управления серверами.
type Service struct {
	db     *pgxpool.Pool
	nodeID string
}

// NewService создаёт новый GuildService.
func NewService(db *pgxpool.Pool, nodeID string) *Service {
	return &Service{db: db, nodeID: nodeID}
}

// CreateGuild создаёт новый сервер и назначает создателя владельцем.
func (s *Service) CreateGuild(ctx context.Context, ownerID, name string) (*models.Guild, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

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

	// Создаём роль владельца
	var roleID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO roles (guild_id, name, level, permissions)
		 VALUES ($1, 'Owner', $2, $3)
		 RETURNING id`,
		guild.ID, RoleLevelOwner, int64(^uint64(0)>>1), // все права
	).Scan(&roleID)
	if err != nil {
		return nil, fmt.Errorf("insert owner role: %w", err)
	}

	// Добавляем создателя как участника с ролью владельца
	_, err = tx.Exec(ctx,
		`INSERT INTO guild_members (guild_id, user_id, role_id)
		 VALUES ($1, $2, $3)`,
		guild.ID, ownerUUID, roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert guild member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &guild, nil
}

// GetUserGuilds возвращает список серверов, в которых состоит пользователь.
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

// CreateInvite создаёт пригласительную ссылку для сервера.
func (s *Service) CreateInvite(ctx context.Context, guildID, createdBy string, expiresIn *int, maxUses *int) (*models.Invite, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	creatorUUID, err := uuid.Parse(createdBy)
	if err != nil {
		return nil, fmt.Errorf("invalid creator id: %w", err)
	}

	// Проверяем, что пользователь является участником сервера
	if err := s.requireMember(ctx, guildUUID, creatorUUID); err != nil {
		return nil, err
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

// JoinByInvite добавляет пользователя на сервер по пригласительной ссылке.
// Возвращает ErrInviteExpired если инвайт истёк или недействителен.
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

	// Получаем инвайт с блокировкой строки
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

	// Проверяем срок действия
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	// Проверяем лимит использований
	if invite.MaxUses != nil && invite.Uses >= *invite.MaxUses {
		return nil, ErrInviteExpired
	}

	// Получаем роль участника (level=0) для этого сервера
	var memberRoleID *uuid.UUID
	var roleID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM roles WHERE guild_id = $1 AND level = $2 LIMIT 1`,
		invite.GuildID, RoleLevelMember,
	).Scan(&roleID)
	if err == nil {
		memberRoleID = &roleID
	}

	// Добавляем участника (или обновляем если уже был)
	_, err = tx.Exec(ctx,
		`INSERT INTO guild_members (guild_id, user_id, role_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (guild_id, user_id) DO UPDATE SET banned = FALSE`,
		invite.GuildID, userUUID, memberRoleID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert member: %w", err)
	}

	// Увеличиваем счётчик использований
	_, err = tx.Exec(ctx,
		`UPDATE invites SET uses = uses + 1 WHERE code = $1`,
		code,
	)
	if err != nil {
		return nil, fmt.Errorf("update invite uses: %w", err)
	}

	// Получаем данные сервера
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

	return &guild, nil
}

// RemoveMember удаляет участника с сервера. Требует прав администратора.
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

	// Проверяем права запрашивающего (должен быть администратором или владельцем)
	requesterLevel, err := s.getMemberLevel(ctx, guildUUID, requesterUUID)
	if err != nil {
		return ErrForbidden
	}
	if requesterLevel < RoleLevelAdmin {
		return ErrForbidden
	}

	// Нельзя удалить владельца
	targetLevel, err := s.getMemberLevel(ctx, guildUUID, targetUUID)
	if err != nil {
		return ErrNotFound
	}
	if targetLevel >= RoleLevelOwner {
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

// CreateChannel создаёт новый канал в сервере.
func (s *Service) CreateChannel(ctx context.Context, guildID, requesterID, name, channelType string) (*models.Channel, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}

	// Проверяем права (должен быть администратором или владельцем)
	level, err := s.getMemberLevel(ctx, guildUUID, requesterUUID)
	if err != nil {
		return nil, ErrForbidden
	}
	if level < RoleLevelAdmin {
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

// GetGuildChannels возвращает список каналов сервера для участника.
func (s *Service) GetGuildChannels(ctx context.Context, guildID, userID string) ([]models.Channel, error) {
	guildUUID, err := uuid.Parse(guildID)
	if err != nil {
		return nil, fmt.Errorf("invalid guild id: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Проверяем, что пользователь является участником сервера
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

// requireMember проверяет, что пользователь является активным участником сервера.
func (s *Service) requireMember(ctx context.Context, guildID, userID uuid.UUID) error {
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

// getMemberLevel возвращает уровень роли участника в сервере.
func (s *Service) getMemberLevel(ctx context.Context, guildID, userID uuid.UUID) (int, error) {
	var level int
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(r.level, 0)
		 FROM guild_members gm
		 LEFT JOIN roles r ON r.id = gm.role_id
		 WHERE gm.guild_id = $1 AND gm.user_id = $2 AND gm.banned = FALSE`,
		guildID, userID,
	).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("query member level: %w", err)
	}
	return level, nil
}

// generateInviteCode генерирует случайный код из 8 символов (base64url без padding).
func generateInviteCode() (string, error) {
	b := make([]byte, 6) // 6 байт → 8 символов base64
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
