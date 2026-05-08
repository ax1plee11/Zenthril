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

type Node struct {
	ID        uuid.UUID  `json:"id"`
	Domain    string     `json:"domain"`
	PublicKey string     `json:"public_key"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Status    string     `json:"status"`
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
