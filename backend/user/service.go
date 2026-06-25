// Package user provides user-level operations that do not belong to the auth
// or device packages: search, profile lookups, etc.
package user

import (
	"context"
	"errors"
	"fmt"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxSearchQueryLength is the upper bound for a user search query.
const MaxSearchQueryLength = 32

// MinSearchQueryLength is the minimum number of characters required to search.
const MinSearchQueryLength = 2

var (
	ErrQueryTooShort = errors.New("search query must be at least 2 characters")
	ErrQueryTooLong  = errors.New("search query must not exceed 32 characters")
	ErrQueryInvalid  = errors.New("search query contains disallowed characters")
)

// SearchResult is a minimal public projection of a user returned by search.
type SearchResult struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Service handles user-scoped read operations.
type Service struct {
	db *pgxpool.Pool
}

// NewService creates a new user Service.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// Search finds users whose username contains the query string (case-insensitive).
// Returns at most 20 results.
//
// Validation rules (enforced server-side regardless of client input):
//   - query length: [2, 32]
//   - allowed characters: Unicode letters, digits, underscore, hyphen
//
// The SQL query uses a parameterised ILIKE pattern, so no manual escaping is needed.
func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if err := validateSearchQuery(query); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id::text, username
		 FROM users
		 WHERE username ILIKE $1
		 ORDER BY username ASC
		 LIMIT 20`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("user search query: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Username); err != nil {
			return nil, fmt.Errorf("user search scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user search rows: %w", err)
	}
	return results, nil
}

// validateSearchQuery enforces length and character constraints.
func validateSearchQuery(q string) error {
	if len([]rune(q)) < MinSearchQueryLength {
		return ErrQueryTooShort
	}
	if len([]rune(q)) > MaxSearchQueryLength {
		return ErrQueryTooLong
	}
	for _, r := range q {
		if !isAllowedSearchChar(r) {
			return ErrQueryInvalid
		}
	}
	return nil
}

// isAllowedSearchChar permits Unicode letters, digits, underscore, and hyphen.
// This is more permissive than the original ASCII-only check, correctly
// supporting non-ASCII usernames while still blocking special SQL characters.
func isAllowedSearchChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
