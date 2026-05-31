package federation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidNode = errors.New("invalid_node")
var ErrInvalidFederationMessage = errors.New("invalid_federation_message")

type Node struct {
	ID        uuid.UUID  `json:"id"`
	Domain    string     `json:"domain"`
	PublicKey string     `json:"public_key"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Status    string     `json:"status"`
}

type MessageEnvelope struct {
	MessageID    string         `json:"message_id"`
	SourceDomain string         `json:"source_domain"`
	TargetDomain string         `json:"target_domain"`
	SenderUserID string         `json:"sender_user_id"`
	TargetUserID string         `json:"target_user_id"`
	Payload      map[string]any `json:"payload"`
}

type StoredMessage struct {
	ID         uuid.UUID      `json:"id"`
	Envelope   MessageEnvelope `json:"envelope"`
	ReceivedAt time.Time      `json:"received_at"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Announce(ctx context.Context, domain, publicKey string) (*Node, error) {
	domain, err := normalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" || len(publicKey) > 4096 {
		return nil, fmt.Errorf("%w: public_key is required", ErrInvalidNode)
	}

	return s.upsertNode(ctx, domain, publicKey)
}

func (s *Service) ListPeers(ctx context.Context) ([]Node, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, domain, public_key, last_seen, status
		 FROM federation_nodes
		 WHERE status = 'active'
		 ORDER BY COALESCE(last_seen, 'epoch'::timestamptz) DESC, domain ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query federation nodes: %w", err)
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		node, err := scanNode(rows.Scan)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("federation rows: %w", err)
	}
	return nodes, nil
}

func (s *Service) ReceiveMessage(ctx context.Context, envelope MessageEnvelope) (*StoredMessage, error) {
	if err := validateMessageEnvelope(envelope); err != nil {
		return nil, err
	}

	var stored StoredMessage
	err := s.db.QueryRow(ctx,
		`INSERT INTO federation_messages (
			message_id, source_domain, target_domain, sender_user_id, target_user_id, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (message_id) DO UPDATE SET message_id = EXCLUDED.message_id
		RETURNING id, message_id, source_domain, target_domain, sender_user_id,
			target_user_id, payload, received_at`,
		envelope.MessageID,
		envelope.SourceDomain,
		envelope.TargetDomain,
		envelope.SenderUserID,
		envelope.TargetUserID,
		envelope.Payload,
	).Scan(
		&stored.ID,
		&stored.Envelope.MessageID,
		&stored.Envelope.SourceDomain,
		&stored.Envelope.TargetDomain,
		&stored.Envelope.SenderUserID,
		&stored.Envelope.TargetUserID,
		&stored.Envelope.Payload,
		&stored.ReceivedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store federation message: %w", err)
	}
	return &stored, nil
}

func (s *Service) upsertNode(ctx context.Context, domain, publicKey string) (*Node, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO federation_nodes (domain, public_key, last_seen, status)
		 VALUES ($1, $2, NOW(), 'active')
		 ON CONFLICT (domain) DO UPDATE
		 SET public_key = EXCLUDED.public_key,
		     last_seen = NOW(),
		     status = 'active'
		 RETURNING id, domain, public_key, last_seen, status`,
		domain, publicKey,
	)
	return scanNode(row.Scan)
}

type scanner func(dest ...any) error

func scanNode(scan scanner) (*Node, error) {
	var node Node
	var lastSeen sql.NullTime
	if err := scan(&node.ID, &node.Domain, &node.PublicKey, &lastSeen, &node.Status); err != nil {
		return nil, fmt.Errorf("scan federation node: %w", err)
	}
	if lastSeen.Valid {
		node.LastSeen = &lastSeen.Time
	}
	return &node, nil
}

func normalizeDomain(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: domain is required", ErrInvalidNode)
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("%w: invalid domain", ErrInvalidNode)
		}
		raw = u.Host
	}
	raw = strings.Trim(strings.ToLower(raw), "/")
	if raw == "" || len(raw) > 255 || strings.ContainsAny(raw, "/?#") {
		return "", fmt.Errorf("%w: invalid domain", ErrInvalidNode)
	}
	return raw, nil
}

func validateMessageEnvelope(envelope MessageEnvelope) error {
	if strings.TrimSpace(envelope.MessageID) == "" || len(envelope.MessageID) > 128 {
		return fmt.Errorf("%w: message_id is required", ErrInvalidFederationMessage)
	}
	if _, err := normalizeDomain(envelope.SourceDomain); err != nil {
		return fmt.Errorf("%w: invalid source_domain", ErrInvalidFederationMessage)
	}
	if _, err := normalizeDomain(envelope.TargetDomain); err != nil {
		return fmt.Errorf("%w: invalid target_domain", ErrInvalidFederationMessage)
	}
	if strings.TrimSpace(envelope.SenderUserID) == "" || strings.TrimSpace(envelope.TargetUserID) == "" {
		return fmt.Errorf("%w: sender_user_id and target_user_id are required", ErrInvalidFederationMessage)
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("%w: payload is required", ErrInvalidFederationMessage)
	}
	// SECURITY: federation payloads must remain encrypted envelopes; plaintext
	// message content is intentionally not modeled in this server-to-server type.
	if _, ok := envelope.Payload["ciphertext"]; !ok {
		return fmt.Errorf("%w: encrypted payload ciphertext is required", ErrInvalidFederationMessage)
	}
	return nil
}
