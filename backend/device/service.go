package device

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidDeviceKey = errors.New("invalid_device_key")
	ErrDeviceNotFound   = errors.New("device_not_found")
)

type OneTimePreKey struct {
	KeyID     int    `json:"key_id"`
	PublicKey string `json:"public_key"`
}

type RegisterDeviceRequest struct {
	DeviceID              string          `json:"device_id,omitempty"`
	Name                  string          `json:"name,omitempty"`
	IdentityPublicKey     string          `json:"identity_public_key"`
	SignedPreKeyID        int             `json:"signed_pre_key_id"`
	SignedPreKey          string          `json:"signed_pre_key"`
	SignedPreKeySignature string          `json:"signed_pre_key_signature"`
	OneTimePreKeys        []OneTimePreKey `json:"one_time_prekeys,omitempty"`
}

type Device struct {
	ID                    string    `json:"device_id"`
	UserID                string    `json:"user_id"`
	Name                  string    `json:"name"`
	IdentityPublicKey     string    `json:"identity_public_key"`
	SignedPreKeyID        int       `json:"signed_pre_key_id"`
	SignedPreKey          string    `json:"signed_pre_key"`
	SignedPreKeySignature string    `json:"signed_pre_key_signature"`
	Fingerprint           string    `json:"fingerprint"`
	TrustState            string    `json:"trust_state"`
	OneTimePreKeyCount    int       `json:"one_time_prekey_count"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	LastSeenAt            time.Time `json:"last_seen_at"`
}

type KeyBundle struct {
	UserID                string         `json:"user_id"`
	DeviceID              string         `json:"device_id"`
	IdentityPublicKey     string         `json:"identity_public_key"`
	SignedPreKeyID        int            `json:"signed_pre_key_id"`
	SignedPreKey          string         `json:"signed_pre_key"`
	SignedPreKeySignature string         `json:"signed_pre_key_signature"`
	OneTimePreKey         *OneTimePreKey `json:"one_time_prekey,omitempty"`
	Fingerprint           string         `json:"fingerprint"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) RegisterDevice(ctx context.Context, userID string, req RegisterDeviceRequest) (*Device, error) {
	if err := validateRegisterDeviceRequest(req); err != nil {
		return nil, err
	}

	deviceID := uuid.New()
	if req.DeviceID != "" {
		parsed, err := uuid.Parse(req.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid device_id", ErrInvalidDeviceKey)
		}
		deviceID = parsed
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id", ErrInvalidDeviceKey)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin register device: %w", err)
	}
	defer tx.Rollback(ctx)

	fingerprint := DeviceFingerprint(userUUID.String(), deviceID.String(), req.IdentityPublicKey)
	row := tx.QueryRow(ctx,
		`INSERT INTO devices (
			id, user_id, name, identity_public_key, signed_pre_key_id,
			signed_pre_key, signed_pre_key_signature, fingerprint, last_seen_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			identity_public_key = EXCLUDED.identity_public_key,
			signed_pre_key_id = EXCLUDED.signed_pre_key_id,
			signed_pre_key = EXCLUDED.signed_pre_key,
			signed_pre_key_signature = EXCLUDED.signed_pre_key_signature,
			fingerprint = EXCLUDED.fingerprint,
			updated_at = NOW(),
			last_seen_at = NOW()
		WHERE devices.user_id = EXCLUDED.user_id
		RETURNING id::text, user_id::text, name, identity_public_key, signed_pre_key_id,
			signed_pre_key, signed_pre_key_signature, fingerprint, trust_state,
			created_at, updated_at, last_seen_at`,
		deviceID, userUUID, strings.TrimSpace(req.Name), req.IdentityPublicKey, req.SignedPreKeyID,
		req.SignedPreKey, req.SignedPreKeySignature, fingerprint,
	)

	var out Device
	if err := scanDevice(row, &out); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("upsert device: %w", err)
	}

	for _, preKey := range req.OneTimePreKeys {
		_, err := tx.Exec(ctx,
			`INSERT INTO device_one_time_prekeys (device_id, key_id, public_key)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (device_id, key_id) DO UPDATE SET
				public_key = EXCLUDED.public_key,
				consumed_at = NULL,
				consumed_by = NULL`,
			deviceID, preKey.KeyID, preKey.PublicKey,
		)
		if err != nil {
			return nil, fmt.Errorf("upsert one-time prekey: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit register device: %w", err)
	}

	count, err := s.CountAvailablePreKeys(ctx, out.ID)
	if err != nil {
		return nil, err
	}
	out.OneTimePreKeyCount = count
	return &out, nil
}

func (s *Service) ListUserDevices(ctx context.Context, userID string) ([]Device, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id", ErrInvalidDeviceKey)
	}

	rows, err := s.db.Query(ctx,
		`SELECT d.id::text, d.user_id::text, d.name, d.identity_public_key, d.signed_pre_key_id,
			d.signed_pre_key, d.signed_pre_key_signature, d.fingerprint, d.trust_state,
			d.created_at, d.updated_at, d.last_seen_at,
			COUNT(p.key_id) FILTER (WHERE p.consumed_at IS NULL)::int AS one_time_prekey_count
		 FROM devices d
		 LEFT JOIN device_one_time_prekeys p ON p.device_id = d.id
		 WHERE d.user_id = $1 AND d.revoked_at IS NULL
		 GROUP BY d.id
		 ORDER BY d.created_at ASC`,
		userUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user devices: %w", err)
	}
	defer rows.Close()

	devices := []Device{}
	for rows.Next() {
		var d Device
		if err := scanDeviceWithPreKeyCount(rows, &d); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}
	return devices, nil
}

func (s *Service) ClaimKeyBundle(ctx context.Context, requesterID, userID, deviceID string) (*KeyBundle, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id", ErrInvalidDeviceKey)
	}
	deviceUUID, err := uuid.Parse(deviceID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid device_id", ErrInvalidDeviceKey)
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid requester_id", ErrInvalidDeviceKey)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim key bundle: %w", err)
	}
	defer tx.Rollback(ctx)

	var bundle KeyBundle
	err = tx.QueryRow(ctx,
		`SELECT user_id::text, id::text, identity_public_key, signed_pre_key_id,
			signed_pre_key, signed_pre_key_signature, fingerprint
		 FROM devices
		 WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL`,
		userUUID, deviceUUID,
	).Scan(
		&bundle.UserID,
		&bundle.DeviceID,
		&bundle.IdentityPublicKey,
		&bundle.SignedPreKeyID,
		&bundle.SignedPreKey,
		&bundle.SignedPreKeySignature,
		&bundle.Fingerprint,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get device key bundle: %w", err)
	}

	var preKey OneTimePreKey
	err = tx.QueryRow(ctx,
		`SELECT key_id, public_key
		 FROM device_one_time_prekeys
		 WHERE device_id = $1 AND consumed_at IS NULL
		 ORDER BY key_id ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		deviceUUID,
	).Scan(&preKey.KeyID, &preKey.PublicKey)
	if err == nil {
		_, err = tx.Exec(ctx,
			`UPDATE device_one_time_prekeys
			 SET consumed_at = NOW(), consumed_by = $1
			 WHERE device_id = $2 AND key_id = $3`,
			requesterUUID, deviceUUID, preKey.KeyID,
		)
		if err != nil {
			return nil, fmt.Errorf("consume one-time prekey: %w", err)
		}
		bundle.OneTimePreKey = &preKey
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("claim one-time prekey: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim key bundle: %w", err)
	}
	return &bundle, nil
}

func (s *Service) CountAvailablePreKeys(ctx context.Context, deviceID string) (int, error) {
	var count int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*)::int
		 FROM device_one_time_prekeys
		 WHERE device_id = $1 AND consumed_at IS NULL`,
		deviceID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count one-time prekeys: %w", err)
	}
	return count, nil
}

func DeviceFingerprint(userID, deviceID, identityPublicKey string) string {
	sum := sha256.Sum256([]byte(userID + ":" + deviceID + ":" + identityPublicKey))
	return hex.EncodeToString(sum[:])
}

func validateRegisterDeviceRequest(req RegisterDeviceRequest) error {
	if len(req.Name) > 100 {
		return fmt.Errorf("%w: name too long", ErrInvalidDeviceKey)
	}
	if err := validateBase64Key(req.IdentityPublicKey, 32, "identity_public_key"); err != nil {
		return err
	}
	if req.SignedPreKeyID <= 0 {
		return fmt.Errorf("%w: signed_pre_key_id is required", ErrInvalidDeviceKey)
	}
	if err := validateBase64Key(req.SignedPreKey, 32, "signed_pre_key"); err != nil {
		return err
	}
	if err := validateBase64Key(req.SignedPreKeySignature, 64, "signed_pre_key_signature"); err != nil {
		return err
	}
	if len(req.OneTimePreKeys) > 100 {
		return fmt.Errorf("%w: too many one_time_prekeys", ErrInvalidDeviceKey)
	}
	seen := map[int]struct{}{}
	for _, preKey := range req.OneTimePreKeys {
		if preKey.KeyID <= 0 {
			return fmt.Errorf("%w: one_time_prekeys.key_id must be positive", ErrInvalidDeviceKey)
		}
		if _, ok := seen[preKey.KeyID]; ok {
			return fmt.Errorf("%w: duplicate one_time_prekeys.key_id", ErrInvalidDeviceKey)
		}
		seen[preKey.KeyID] = struct{}{}
		if err := validateBase64Key(preKey.PublicKey, 32, "one_time_prekeys.public_key"); err != nil {
			return err
		}
	}
	return nil
}

func validateBase64Key(value string, wantLen int, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidDeviceKey, field)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(value)
	}
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(value)
	}
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(value)
	}
	if err != nil {
		return fmt.Errorf("%w: %s must be base64", ErrInvalidDeviceKey, field)
	}
	if len(decoded) != wantLen {
		return fmt.Errorf("%w: %s must decode to %d bytes", ErrInvalidDeviceKey, field, wantLen)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(row rowScanner, d *Device) error {
	return row.Scan(
		&d.ID,
		&d.UserID,
		&d.Name,
		&d.IdentityPublicKey,
		&d.SignedPreKeyID,
		&d.SignedPreKey,
		&d.SignedPreKeySignature,
		&d.Fingerprint,
		&d.TrustState,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.LastSeenAt,
	)
}

func scanDeviceWithPreKeyCount(row rowScanner, d *Device) error {
	return row.Scan(
		&d.ID,
		&d.UserID,
		&d.Name,
		&d.IdentityPublicKey,
		&d.SignedPreKeyID,
		&d.SignedPreKey,
		&d.SignedPreKeySignature,
		&d.Fingerprint,
		&d.TrustState,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.LastSeenAt,
		&d.OneTimePreKeyCount,
	)
}
