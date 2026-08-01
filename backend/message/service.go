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

type RecipientDevice struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
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

// ListRecipientDevices exposes only active devices of current channel members.
// SECURITY: callers must be channel members; this prevents device enumeration
// across unrelated guilds.
func (s *Service) ListRecipientDevices(ctx context.Context, channelID, requesterID string) ([]RecipientDevice, error) {
	if err := s.requireChannelAccess(ctx, requesterID, channelID); err != nil {
		return nil, err
	}
	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		return nil, fmt.Errorf("invalid channel id: %w", err)
	}
	rows, err := s.db.Query(ctx,
		`SELECT d.user_id::text, d.id::text
		 FROM channels c
		 JOIN guild_members gm ON gm.guild_id = c.guild_id AND gm.banned = FALSE
		 JOIN devices d ON d.user_id = gm.user_id AND d.revoked_at IS NULL
		 WHERE c.id = $1
		 ORDER BY d.user_id, d.id`, channelUUID)
	if err != nil {
		return nil, fmt.Errorf("list channel recipient devices: %w", err)
	}
	defer rows.Close()
	recipients := make([]RecipientDevice, 0)
	for rows.Next() {
		var recipient RecipientDevice
		if err := rows.Scan(&recipient.UserID, &recipient.DeviceID); err != nil {
			return nil, fmt.Errorf("scan channel recipient device: %w", err)
		}
		recipients = append(recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recipient device rows: %w", err)
	}
	return recipients, nil
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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin message transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.validateRecipientEnvelopeAccess(ctx, tx, channelUUID, payload.RecipientEnvelopes); err != nil {
		return nil, err
	}

	var msg models.Message
	err = tx.QueryRow(ctx,
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
	if err := s.insertRecipientEnvelopes(ctx, tx, msg.ID, payload.RecipientEnvelopes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit message transaction: %w", err)
	}
	msg.Payload.RecipientEnvelopes = payload.RecipientEnvelopes
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
	if err := s.attachRecipientEnvelopes(ctx, messages, userID); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Service) validateRecipientEnvelopeAccess(ctx context.Context, tx pgx.Tx, channelID uuid.UUID, envelopes []models.RecipientKeyEnvelope) error {
	for _, envelope := range envelopes {
		userID, err := uuid.Parse(envelope.RecipientUserID)
		if err != nil {
			return ErrInvalidEnvelope
		}
		deviceID, err := uuid.Parse(envelope.RecipientDeviceID)
		if err != nil {
			return ErrInvalidEnvelope
		}
		var allowed bool
		err = tx.QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1
				FROM channels c
				JOIN guild_members gm ON gm.guild_id = c.guild_id
				JOIN devices d ON d.id = $3 AND d.user_id = $2 AND d.revoked_at IS NULL
				WHERE c.id = $1 AND gm.user_id = $2 AND gm.banned = FALSE
			)`,
			channelID, userID, deviceID,
		).Scan(&allowed)
		if err != nil {
			return fmt.Errorf("validate recipient device access: %w", err)
		}
		if !allowed {
			return ErrInvalidEnvelope
		}
	}
	return nil
}

func (s *Service) insertRecipientEnvelopes(ctx context.Context, tx pgx.Tx, messageID uuid.UUID, envelopes []models.RecipientKeyEnvelope) error {
	for _, envelope := range envelopes {
		userID, _ := uuid.Parse(envelope.RecipientUserID)
		deviceID, _ := uuid.Parse(envelope.RecipientDeviceID)
		payloadJSON, err := json.Marshal(envelope.Payload)
		if err != nil {
			return fmt.Errorf("marshal recipient envelope payload: %w", err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO message_recipient_envelopes
				(message_id, recipient_user_id, recipient_device_id, session_id, ratchet_counter, bootstrap_header, payload)
			 VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::jsonb, $7::jsonb)`,
			messageID, userID, deviceID, envelope.SessionID, envelope.RatchetCounter,
			string(envelope.BootstrapHeader), string(payloadJSON),
		)
		if err != nil {
			return fmt.Errorf("insert recipient envelope: %w", err)
		}
	}
	return nil
}

func (s *Service) attachRecipientEnvelopes(ctx context.Context, messages []models.Message, userID string) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(messages))
	byID := make(map[uuid.UUID]*models.Message, len(messages))
	for i := range messages {
		ids = append(ids, messages[i].ID)
		byID[messages[i].ID] = &messages[i]
	}
	recipientID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("parse recipient user id: %w", err)
	}
	rows, err := s.db.Query(ctx,
		`SELECT message_id, recipient_user_id::text, recipient_device_id::text, session_id, ratchet_counter, COALESCE(bootstrap_header::text, ''), payload
		 FROM message_recipient_envelopes
		 WHERE message_id = ANY($1) AND recipient_user_id = $2`,
		ids, recipientID,
	)
	if err != nil {
		return fmt.Errorf("query recipient envelopes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var messageID uuid.UUID
		var envelope models.RecipientKeyEnvelope
		var bootstrap string
		var payloadJSON []byte
		if err := rows.Scan(&messageID, &envelope.RecipientUserID, &envelope.RecipientDeviceID, &envelope.SessionID, &envelope.RatchetCounter, &bootstrap, &payloadJSON); err != nil {
			return fmt.Errorf("scan recipient envelope: %w", err)
		}
		if err := json.Unmarshal(payloadJSON, &envelope.Payload); err != nil {
			return fmt.Errorf("decode recipient envelope payload: %w", err)
		}
		if bootstrap != "" {
			envelope.BootstrapHeader = []byte(bootstrap)
		}
		if message := byID[messageID]; message != nil {
			message.Payload.RecipientEnvelopes = append(message.Payload.RecipientEnvelopes, envelope)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("recipient envelope rows: %w", err)
	}
	return nil
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
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin encrypted edit transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.validateRecipientEnvelopeAccess(ctx, tx, chID, payload.RecipientEnvelopes); err != nil {
		return nil, err
	}

	var msg models.Message
	err = tx.QueryRow(ctx,
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
	// SECURITY: ciphertext and its device envelopes are replaced in the same
	// transaction. A receiver therefore never observes a mixed key/message pair.
	if _, err := tx.Exec(ctx, `DELETE FROM message_recipient_envelopes WHERE message_id = $1`, msg.ID); err != nil {
		return nil, fmt.Errorf("delete previous recipient envelopes: %w", err)
	}
	if err := s.insertRecipientEnvelopes(ctx, tx, msg.ID, payload.RecipientEnvelopes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit encrypted edit transaction: %w", err)
	}
	msg.Payload.RecipientEnvelopes = payload.RecipientEnvelopes
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
