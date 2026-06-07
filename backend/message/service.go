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

	"zenthril-backend/guild"
	"zenthril-backend/hub"
	"zenthril-backend/models"
)

var (
	ErrNotFound         = errors.New("not_found")
	ErrForbidden        = errors.New("forbidden")
	ErrNotChannelMember = errors.New("not_channel_member")
	ErrInvalidEnvelope  = errors.New("invalid_envelope")
)

const maxLimit = 50

type Service struct {
	db    *pgxpool.Pool
	hub   *hub.Hub
	guild *guild.Service
}

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
		`INSERT INTO messages (channel_id, author_id, ciphertext, iv, key_id, tag, protocol_version, sender_device_id, session_id, client_message_id, cipher_suite)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, channel_id, author_id, ciphertext, iv, key_id, tag, protocol_version, COALESCE(sender_device_id, ''), COALESCE(session_id, ''), COALESCE(client_message_id, ''), COALESCE(cipher_suite, ''), edited, deleted, created_at, updated_at`,
		channelUUID, authorUUID, payload.Ciphertext, payload.IV, payload.KeyID, payload.Tag, payload.ProtocolVersion,
		payload.SenderDeviceID, payload.SessionID, payload.ClientMessageID, payload.CipherSuite,
	).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID,
		&msg.Payload.Ciphertext, &msg.Payload.IV, &msg.Payload.KeyID, &msg.Payload.Tag, &msg.Payload.ProtocolVersion,
		&msg.Payload.SenderDeviceID, &msg.Payload.SessionID, &msg.Payload.ClientMessageID, &msg.Payload.CipherSuite,
		&msg.Edited, &msg.Deleted, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}
	attachAADMetadata(&msg)

	s.broadcastEvent(channelID, "message.new", &msg)
	return &msg, nil
}

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
			`SELECT id, channel_id, author_id, ciphertext, iv, key_id, tag, protocol_version, COALESCE(sender_device_id, ''), COALESCE(session_id, ''), COALESCE(client_message_id, ''), COALESCE(cipher_suite, ''), edited, deleted, created_at, updated_at
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
			`SELECT id, channel_id, author_id, ciphertext, iv, key_id, tag, protocol_version, COALESCE(sender_device_id, ''), COALESCE(session_id, ''), COALESCE(client_message_id, ''), COALESCE(cipher_suite, ''), edited, deleted, created_at, updated_at
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
			&m.Payload.Ciphertext, &m.Payload.IV, &m.Payload.KeyID, &m.Payload.Tag, &m.Payload.ProtocolVersion,
			&m.Payload.SenderDeviceID, &m.Payload.SessionID, &m.Payload.ClientMessageID, &m.Payload.CipherSuite,
			&m.Edited, &m.Deleted, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		attachAADMetadata(&m)
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
	if err := validateStoredEnvelopeClaims(payload, chID.String(), authorID); err != nil {
		return nil, err
	}
	if err := s.requireChannelAccess(ctx, authorID, chID.String()); err != nil {
		return nil, err
	}

	var msg models.Message
	err = s.db.QueryRow(ctx,
		`UPDATE messages
		 SET ciphertext = $1, iv = $2, key_id = $3, tag = $4, protocol_version = $5, sender_device_id = $6, session_id = $7, client_message_id = $8, cipher_suite = $9, edited = TRUE, updated_at = $10
		 WHERE id = $11 AND author_id = $12 AND deleted = FALSE
		 RETURNING id, channel_id, author_id, ciphertext, iv, key_id, tag, protocol_version, COALESCE(sender_device_id, ''), COALESCE(session_id, ''), COALESCE(client_message_id, ''), COALESCE(cipher_suite, ''), edited, deleted, created_at, updated_at`,
		payload.Ciphertext, payload.IV, payload.KeyID, payload.Tag, payload.ProtocolVersion,
		payload.SenderDeviceID, payload.SessionID, payload.ClientMessageID, payload.CipherSuite, time.Now(),
		msgUUID, authorUUID,
	).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID,
		&msg.Payload.Ciphertext, &msg.Payload.IV, &msg.Payload.KeyID, &msg.Payload.Tag, &msg.Payload.ProtocolVersion,
		&msg.Payload.SenderDeviceID, &msg.Payload.SessionID, &msg.Payload.ClientMessageID, &msg.Payload.CipherSuite,
		&msg.Edited, &msg.Deleted, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	attachAADMetadata(&msg)

	s.broadcastEvent(msg.ChannelID.String(), "message.edited", &msg)
	return &msg, nil
}

func attachAADMetadata(msg *models.Message) {
	if msg == nil || msg.Payload.ProtocolVersion != models.CryptoProtocolVersion {
		return
	}
	msg.Payload.ChannelID = msg.ChannelID.String()
	msg.Payload.SenderUserID = msg.AuthorID.String()
}

func validateStoredEnvelopeClaims(payload models.EncryptedPayload, channelID, authorID string) error {
	if payload.ProtocolVersion != models.CryptoProtocolVersion {
		return nil
	}
	// SECURITY-HARDENING: edit routes do not carry channel_id, so the service
	// must bind protocol-v2 AAD claims to the stored message context.
	if payload.ChannelID != "" && payload.ChannelID != channelID {
		return ErrInvalidEnvelope
	}
	if payload.SenderUserID != "" && payload.SenderUserID != authorID {
		return ErrInvalidEnvelope
	}
	return nil
}

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
		 SET deleted = TRUE, ciphertext = '', iv = '', tag = '', updated_at = $1
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
