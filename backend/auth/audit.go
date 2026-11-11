package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLog represents a single admin audit log entry.
type AuditLog struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"admin_id"`
	Action    string    `json:"action"`
	TargetID  string    `json:"target_id,omitempty"`
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// LogAdminAction records a privileged admin operation to the audit log table
// and emits a structured slog entry.
//
// SECURITY: never log secret values (tokens, passwords, OTP codes, private keys).
// Use only IDs, action names, and non-sensitive context.
func (s *Service) LogAdminAction(ctx context.Context, adminID, action, targetID, details, ipAddress string) error {
	// Structured log for observability pipelines (ELK, Loki, CloudWatch, etc.)
	slog.InfoContext(ctx, "admin action",
		"event", "admin_audit",
		"admin_id", adminID,
		"action", action,
		"target_id", targetID,
		"ip_address", ipAddress,
		// details may contain free-form text; callers must not put secrets here.
	)

	_, err := s.db.Exec(ctx,
		`INSERT INTO admin_audit_log (admin_id, action, target_id, details, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		adminID, action, targetID, details, ipAddress,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to persist admin audit log",
			"event", "admin_audit_error",
			"admin_id", adminID,
			"action", action,
			"error", err,
		)
	}

	// Return nil intentionally: a logging failure must never block the admin action.
	return nil
}

// GetAuditLog returns the most recent admin audit entries, up to limit (max 1000).
func (s *Service) GetAuditLog(ctx context.Context, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, admin_id, action, target_id, details, ip_address, created_at
		 FROM admin_audit_log
		 ORDER BY created_at DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.AdminID,
			&l.Action,
			&l.TargetID,
			&l.Details,
			&l.IPAddress,
			&l.CreatedAt,
		); err != nil {
			// Skip malformed rows rather than aborting the whole result set.
			slog.Warn("audit log scan error", "error", err)
			continue
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// CreateAuditLogTable creates the admin_audit_log table and its indexes if they
// do not already exist. Intended for use in migration scripts or test setup.
func CreateAuditLogTable(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS admin_audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			admin_id UUID NOT NULL,
			action VARCHAR(100) NOT NULL,
			target_id UUID,
			details TEXT,
			ip_address VARCHAR(45),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_admin_audit_admin_id ON admin_audit_log(admin_id);
		CREATE INDEX IF NOT EXISTS idx_admin_audit_created_at ON admin_audit_log(created_at DESC);
	`)
	return err
}
