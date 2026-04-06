package friends

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyFriends  = errors.New("already_friends")
	ErrRequestPending  = errors.New("request_pending")
	ErrNotFound        = errors.New("not_found")
	ErrForbidden       = errors.New("forbidden")
	ErrCannotSelfAdd   = errors.New("cannot_add_self")
)

type FriendUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Status   string `json:"status"` // friendship status: pending/accepted
	// Direction: "incoming" | "outgoing" (only for pending)
	Direction string `json:"direction,omitempty"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// GetUsername возвращает username пользователя по ID.
func (s *Service) GetUsername(ctx context.Context, userID string) (string, error) {
	uUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	var username string
	err = s.db.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, uUUID).Scan(&username)
	return username, err
}

// SendRequest отправляет запрос в друзья.
func (s *Service) SendRequest(ctx context.Context, requesterID, addresseeID string) error {
	if requesterID == addresseeID {
		return ErrCannotSelfAdd
	}
	rUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return fmt.Errorf("invalid requester id: %w", err)
	}
	aUUID, err := uuid.Parse(addresseeID)
	if err != nil {
		return fmt.Errorf("invalid addressee id: %w", err)
	}

	// Проверяем существующую связь в обе стороны
	var existingStatus string
	err = s.db.QueryRow(ctx,
		`SELECT status FROM friendships
		 WHERE (requester_id=$1 AND addressee_id=$2)
		    OR (requester_id=$2 AND addressee_id=$1)`,
		rUUID, aUUID,
	).Scan(&existingStatus)

	if err == nil {
		switch existingStatus {
		case "accepted":
			return ErrAlreadyFriends
		case "pending":
			return ErrRequestPending
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check friendship: %w", err)
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO friendships (requester_id, addressee_id, status)
		 VALUES ($1, $2, 'pending')
		 ON CONFLICT (requester_id, addressee_id) DO UPDATE SET status='pending', updated_at=NOW()`,
		rUUID, aUUID,
	)
	if err != nil {
		return fmt.Errorf("insert friendship: %w", err)
	}
	return nil
}

// AcceptRequest принимает входящий запрос.
func (s *Service) AcceptRequest(ctx context.Context, userID, requesterID string) error {
	uUUID, _ := uuid.Parse(userID)
	rUUID, _ := uuid.Parse(requesterID)

	result, err := s.db.Exec(ctx,
		`UPDATE friendships SET status='accepted', updated_at=NOW()
		 WHERE requester_id=$1 AND addressee_id=$2 AND status='pending'`,
		rUUID, uUUID,
	)
	if err != nil {
		return fmt.Errorf("accept request: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeclineRequest отклоняет или удаляет запрос/дружбу.
func (s *Service) DeclineRequest(ctx context.Context, userID, otherID string) error {
	uUUID, _ := uuid.Parse(userID)
	oUUID, _ := uuid.Parse(otherID)

	_, err := s.db.Exec(ctx,
		`DELETE FROM friendships
		 WHERE (requester_id=$1 AND addressee_id=$2)
		    OR (requester_id=$2 AND addressee_id=$1)`,
		uUUID, oUUID,
	)
	return err
}

// ListFriends возвращает принятых друзей + входящие/исходящие запросы.
func (s *Service) ListFriends(ctx context.Context, userID string) ([]FriendUser, error) {
	uUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT
			CASE WHEN f.requester_id=$1 THEN f.addressee_id ELSE f.requester_id END AS other_id,
			u.username,
			f.status,
			CASE WHEN f.requester_id=$1 THEN 'outgoing' ELSE 'incoming' END AS direction
		 FROM friendships f
		 JOIN users u ON u.id = CASE WHEN f.requester_id=$1 THEN f.addressee_id ELSE f.requester_id END
		 WHERE (f.requester_id=$1 OR f.addressee_id=$1)
		   AND f.status IN ('pending','accepted')
		 ORDER BY f.status DESC, u.username ASC`,
		uUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query friends: %w", err)
	}
	defer rows.Close()

	var result []FriendUser
	for rows.Next() {
		var f FriendUser
		if err := rows.Scan(&f.ID, &f.Username, &f.Status, &f.Direction); err != nil {
			return nil, fmt.Errorf("scan friend: %w", err)
		}
		if f.Status == "accepted" {
			f.Direction = ""
		}
		result = append(result, f)
	}
	if result == nil {
		result = []FriendUser{}
	}
	return result, nil
}
