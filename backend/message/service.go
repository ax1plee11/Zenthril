package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"veltrix-backend/guild"
	"veltrix-backend/hub"
	"veltrix-backend/models"
)

var (
	ErrNotFound         = errors.New("not_found")
	ErrForbidden        = errors.New("forbidden")
	ErrNotChannelMember = errors.New("not_channel_member")
)

const maxLimit = 50

// Service реализует бизнес-логику работы с сообщениями.
type Service struct {
	db    *pgxpool.Pool
	hub   *hub.Hub
	guild *guild.Service
}

// NewService создаёт новый MessageService.
func NewService(db *pgxpool.Pool, h *hub.Hub, g *guild.Service) *Service {
	return &Service{db: db, hub: h, guild: g}
}

func (s *Service) requireChannelAccess(ctx context.Context, userID, channelID string) error {
	if s.guild == nil {
		return fmt.Errorf("guild service not configured")
	}
	ok, err := s.guild.UserHasChannelAccess(ctx, userID, channelID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotChannelMember
	}
	return nil
}

// SendMessage сохраняет сообщение в БД и рассылает событие через hub.
func (s *Service) SendMessage(ctx context.Context, channelID, authorID string, payload models.EncryptedPayload) (*models.Message, error) {
	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		return nil, fmt.Errorf("invalid channel id: %w", err)
	}
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, fmt.Errorf("invalid author id: %w", err)
	}

	if err := s.requireChannelAccess(ctx, authorID, channelID); err != nil {
		return nil, err
	}

	var msg models.Message
	err = s.db.QueryRow(ctx,
		`INSERT INTO messages (channel_id, author_id, ciphertext, iv, key_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, channel_id, author_id, ciphertext, iv, key_id, edited, deleted, created_at, updated_at`,
		channelUUID, authorUUID, payload.Ciphertext, payload.IV, payload.KeyID,
	).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID,
		&msg.Payload.Ciphertext, &msg.Payload.IV, &msg.Payload.KeyID,
		&msg.Edited, &msg.Deleted, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	s.broadcastEvent(channelID, "message.new", &msg)
	return &msg, nil
}

// GetHistory возвращает историю сообщений канала с пагинацией по before (message ID).
func (s *Service) GetHistory(ctx context.Context, channelID, userID string, before *string, limit int) ([]models.Message, error) {
	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		return nil, fmt.Errorf("invalid channel id: %w", err)
	}

	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}

	if err := s.requireChannelAccess(ctx, userID, channelID); err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if before != nil && *before != "" {
		beforeUUID, err := uuid.Parse(*before)
		if err != nil {
			return nil, fmt.Errorf("invalid before id: %w", err)
		}
		rows, err = s.db.Query(ctx,
			`SELECT id, channel_id, author_id, ciphertext, iv, key_id, edited, deleted, created_at, updated_at
			 FROM messages
			 WHERE channel_id = $1 AND created_at < (SELECT created_at FROM messages WHERE id = $2)
			 ORDER BY created_at DESC
			 LIMIT $3`,
			channelUUID, beforeUUID, limit,
		)
		if err != nil {
			return nil, fmt.Errorf("query messages: %w", err)
		}
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT id, channel_id, author_id, ciphertext, iv, key_id, edited, deleted, created_at, updated_at
			 FROM messages
			 WHERE channel_id = $1
			 ORDER BY created_at DESC
			 LIMIT $2`,
			channelUUID, limit,
		)
		if err != nil {
			return nil, fmt.Errorf("query messages: %w", err)
		}
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.AuthorID,
			&m.Payload.Ciphertext, &m.Payload.IV, &m.Payload.KeyID,
			&m.Edited, &m.Deleted, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if messages == nil {
		messages = []models.Message{}
	}
	return messages, nil
}

// EditMessage редактирует сообщение (только автор).
func (s *Service) EditMessage(ctx context.Context, messageID, authorID string, payload models.EncryptedPayload) (*models.Message, error) {
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return nil, fmt.Errorf("invalid message id: %w", err)
	}
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, fmt.Errorf("invalid author id: %w", err)
	}

	var chID uuid.UUID
	err = s.db.QueryRow(ctx,
		`SELECT channel_id FROM messages WHERE id = $1 AND deleted = FALSE`,
		msgUUID,
	).Scan(&chID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load message channel: %w", err)
	}
	if err := s.requireChannelAccess(ctx, authorID, chID.String()); err != nil {
		return nil, err
	}

	var msg models.Message
	err = s.db.QueryRow(ctx,
		`UPDATE messages
		 SET ciphertext = $1, iv = $2, key_id = $3, edited = TRUE, updated_at = $4
		 WHERE id = $5 AND author_id = $6 AND deleted = FALSE
		 RETURNING id, channel_id, author_id, ciphertext, iv, key_id, edited, deleted, created_at, updated_at`,
		payload.Ciphertext, payload.IV, payload.KeyID, time.Now(),
		msgUUID, authorUUID,
	).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID,
		&msg.Payload.Ciphertext, &msg.Payload.IV, &msg.Payload.KeyID,
		&msg.Edited, &msg.Deleted, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Проверяем: сообщение существует, но не принадлежит автору?
			var exists bool
			_ = s.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1 AND deleted = FALSE)`,
				msgUUID,
			).Scan(&exists)
			if exists {
				return nil, ErrForbidden
			}
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update message: %w", err)
	}

	s.broadcastEvent(msg.ChannelID.String(), "message.edited", &msg)
	return &msg, nil
}

// DeleteMessage помечает сообщение удалённым (только автор).
func (s *Service) DeleteMessage(ctx context.Context, messageID, authorID string) error {
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return fmt.Errorf("invalid message id: %w", err)
	}
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return fmt.Errorf("invalid author id: %w", err)
	}

	var chID uuid.UUID
	err = s.db.QueryRow(ctx,
		`SELECT channel_id FROM messages WHERE id = $1 AND deleted = FALSE`,
		msgUUID,
	).Scan(&chID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load message channel: %w", err)
	}
	if err := s.requireChannelAccess(ctx, authorID, chID.String()); err != nil {
		return err
	}

	var channelID uuid.UUID
	err = s.db.QueryRow(ctx,
		`UPDATE messages
		 SET deleted = TRUE, ciphertext = '', iv = '', updated_at = $1
		 WHERE id = $2 AND author_id = $3 AND deleted = FALSE
		 RETURNING channel_id`,
		time.Now(), msgUUID, authorUUID,
	).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			_ = s.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1 AND deleted = FALSE)`,
				msgUUID,
			).Scan(&exists)
			if exists {
				return ErrForbidden
			}
			return ErrNotFound
		}
		return fmt.Errorf("delete message: %w", err)
	}

	event, _ := json.Marshal(map[string]interface{}{
		"type":       "message.deleted",
		"message_id": messageID,
		"channel_id": channelID.String(),
	})
	s.hub.Broadcast(channelID.String(), event)
	return nil
}

// broadcastEvent сериализует событие и рассылает в канал.
func (s *Service) broadcastEvent(channelID, eventType string, msg *models.Message) {
	event, err := json.Marshal(map[string]interface{}{
		"type":    eventType,
		"message": msg,
	})
	if err != nil {
		return
	}
	s.hub.Broadcast(channelID, event)
}
