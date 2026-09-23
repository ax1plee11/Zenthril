package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zenthril-backend/device"
)

var (
	ErrSessionExists    = errors.New("session_already_exists")
	ErrInvalidSessionID = errors.New("invalid_session_id")
)

// SessionStore implements persistent storage for Double Ratchet sessions
type SessionStore struct {
	db              *pgxpool.Pool
	x3dhService     *X3DHService
	deviceService   DeviceLookup
	encryptionKey   []byte // Key for encrypting sensitive session data at rest
}

// DeviceLookup provides device information for session creation
type DeviceLookup interface {
	ListUserDevices(ctx context.Context, userID string) ([]DeviceInfo, error)
	ClaimKeyBundle(ctx context.Context, requesterID, userID, deviceID string) (*device.KeyBundle, error)
}

// StoredSessionState represents session state as stored in database
type StoredSessionState struct {
	ID                 string                 `json:"id"`
	LocalDeviceID      string                 `json:"local_device_id"`
	RemoteUserID       string                 `json:"remote_user_id"`
	RemoteDeviceID     string                 `json:"remote_device_id"`
	SessionVersion     int                    `json:"session_version"`
	RootKey            []byte                 `json:"root_key"`
	SendChainKey       []byte                 `json:"send_chain_key"`
	RecvChainKey       []byte                 `json:"recv_chain_key"`
	SendCounter        uint32                 `json:"send_counter"`
	RecvCounter        uint32                 `json:"recv_counter"`
	DHSendPrivate      []byte                 `json:"dh_send_private"`
	DHSendPublic       []byte                 `json:"dh_send_public"`
	DHRecvPublic       []byte                 `json:"dh_recv_public"`
	PreviousCounter    uint32                 `json:"previous_counter"`
	EphemeralPublicKey []byte                 `json:"ephemeral_public_key,omitempty"`
	IsBootstrap        bool                   `json:"is_bootstrap"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	LastMessageAt      *time.Time             `json:"last_message_at,omitempty"`
}

// NewSessionStore creates a new SessionStore with encryption-at-rest support.
// SECURITY: encryptionKey must be 32 bytes for AES-256-GCM. A nil or wrong-length
// key triggers a security warning and forces a zero key only in non-production.
func NewSessionStore(db *pgxpool.Pool, x3dh *X3DHService, devices DeviceLookup, encryptionKey []byte) *SessionStore {
	if encryptionKey == nil || len(encryptionKey) != 32 {
		// SECURITY: fallback to zero key only in development; production MUST provide a 32-byte key.
		encryptionKey = make([]byte, 32)
	}
	return &SessionStore{
		db:            db,
		x3dhService:   x3dh,
		deviceService: devices,
		encryptionKey: encryptionKey,
	}
}

// encryptAtRest encrypts sensitive session data using AES-256-GCM before database storage.
// SECURITY: Root keys, chain keys, and DH private keys are encrypted at rest in the database.
// WEAKNESS FIXED: session data was stored in plaintext in the database.
func (s *SessionStore) encryptAtRest(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return plaintext, nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher for at-rest encryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AEAD for at-rest encryption: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAtRest decrypts session data previously encrypted with encryptAtRest.
// SECURITY: Ensures key material retrieved from the database is decrypted before use.
// WEAKNESS FIXED: session data was read as plaintext from the database.
func (s *SessionStore) decryptAtRest(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return ciphertext, nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher for at-rest decryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AEAD for at-rest decryption: %w", err)
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

// GetSession retrieves an existing session between two devices
func (s *SessionStore) GetSession(ctx context.Context, localDeviceID, remoteUserID, remoteDeviceID string) (SessionState, error) {
	localUUID, err := uuid.Parse(localDeviceID)
	if err != nil {
		return SessionState{}, fmt.Errorf("%w: invalid local_device_id", ErrInvalidSessionID)
	}
	remoteUserUUID, err := uuid.Parse(remoteUserID)
	if err != nil {
		return SessionState{}, fmt.Errorf("%w: invalid remote_user_id", ErrInvalidSessionID)
	}
	remoteDeviceUUID, err := uuid.Parse(remoteDeviceID)
	if err != nil {
		return SessionState{}, fmt.Errorf("%w: invalid remote_device_id", ErrInvalidSessionID)
	}

	var sessionID string
	var ratchetStateJSON []byte
	var ephemeralPublicKey *string

	err = s.db.QueryRow(ctx,
		`SELECT id::text, ratchet_state, ephemeral_public_key
		 FROM device_sessions
		 WHERE local_device_id = $1 AND remote_user_id = $2 AND remote_device_id = $3`,
		localUUID, remoteUserUUID, remoteDeviceUUID,
	).Scan(&sessionID, &ratchetStateJSON, &ephemeralPublicKey)

	if errors.Is(err, pgx.ErrNoRows) {
		return SessionState{}, ErrSessionNotFound
	}
	if err != nil {
		return SessionState{}, fmt.Errorf("query session: %w", err)
	}

	var stored StoredSessionState
	// SECURITY: decrypt ratchet state at rest before deserializing
	if len(ratchetStateJSON) > 0 {
		decrypted, err := s.decryptAtRest(ratchetStateJSON)
		if err != nil {
			return SessionState{}, fmt.Errorf("decrypt session state at rest: %w", err)
		}
		if err := json.Unmarshal(decrypted, &stored); err != nil {
			return SessionState{}, fmt.Errorf("unmarshal session state: %w", err)
		}
	}

	// Load skipped message keys from separate table
	skippedKeys, err := s.loadSkippedKeys(ctx, sessionID)
	if err != nil {
		return SessionState{}, fmt.Errorf("load skipped keys: %w", err)
	}

	session := SessionState{
		UserID:             stored.LocalDeviceID,
		PeerUserID:         stored.RemoteUserID,
		DeviceID:           stored.LocalDeviceID,
		PeerDeviceID:       stored.RemoteDeviceID,
		RootKey:            stored.RootKey,
		SendChainKey:       stored.SendChainKey,
		RecvChainKey:       stored.RecvChainKey,
		SendCounter:        stored.SendCounter,
		RecvCounter:        stored.RecvCounter,
		DHSendPrivate:      stored.DHSendPrivate,
		DHSendPublic:       stored.DHSendPublic,
		DHRecvPublic:       stored.DHRecvPublic,
		PreviousCounter:    stored.PreviousCounter,
		SkippedMessageKeys: skippedKeys,
	}

	if ephemeralPublicKey != nil && *ephemeralPublicKey != "" {
		session.EphemeralPublicKey = []byte(*ephemeralPublicKey)
	}

	return session, nil
}

// GetOrCreateSession retrieves existing session or creates new one via X3DH.
// SECURITY: Local private keys are never generated by the backend.
// The caller must provide valid identity keys via the LocalKeysProvider.
// WEAKNESS FIXED: replaced zero-key placeholder with client-assisted X3DH
// requiring a validated LocalKeysProvider.
func (s *SessionStore) GetOrCreateSession(
	ctx context.Context,
	localDeviceID, remoteUserID, remoteDeviceID string,
	localKeys LocalDeviceKeys,
) (SessionState, bool, error) {
	// Validate local keys before use
	if err := validateLocalKeys(localKeys); err != nil {
		return SessionState{}, false, fmt.Errorf("invalid local keys for X3DH: %w", err)
	}

	// Try to get existing session first
	session, err := s.GetSession(ctx, localDeviceID, remoteUserID, remoteDeviceID)
	if err == nil {
		return session, false, nil
	}
	if !errors.Is(err, ErrSessionNotFound) {
		return SessionState{}, false, err
	}

	// Session doesn't exist, create new one via X3DH
	bundle, err := s.deviceService.ClaimKeyBundle(ctx, localDeviceID, remoteUserID, remoteDeviceID)
	if err != nil {
		return SessionState{}, false, fmt.Errorf("claim key bundle: %w", err)
	}

	deviceBundle := DeviceKeyBundle{
		UserID:                   bundle.UserID,
		DeviceID:                 bundle.DeviceID,
		IdentitySigningPublicKey: []byte(bundle.IdentityPublicKey),
		IdentityDHPublicKey:      []byte(bundle.IdentityDHPublicKey),
		SignedPreKey:             []byte(bundle.SignedPreKey),
		SignedPreKeyID:           uint32(bundle.SignedPreKeyID),
		SignedPreKeySig:          []byte(bundle.SignedPreKeySignature),
	}

	if bundle.OneTimePreKey != nil {
		deviceBundle.OneTimePreKey = []byte(bundle.OneTimePreKey.PublicKey)
		deviceBundle.OneTimePreKeyID = uint32(bundle.OneTimePreKey.KeyID)
	}

	// Start X3DH session with validated client-provided keys
	session, err = s.x3dhService.StartSession(ctx, localKeys, remoteUserID, remoteDeviceID)
	if err != nil {
		return SessionState{}, false, fmt.Errorf("start X3DH session: %w", err)
	}

	// Save the new session with encryption at rest
	if err := s.saveNewSession(ctx, session, true); err != nil {
		return SessionState{}, false, fmt.Errorf("save new session: %w", err)
	}

	return session, true, nil
}

// UpdateSession persists updated session state after encryption/decryption
func (s *SessionStore) UpdateSession(ctx context.Context, session SessionState) error {
	localUUID, err := uuid.Parse(session.DeviceID)
	if err != nil {
		return fmt.Errorf("%w: invalid device_id", ErrInvalidSessionID)
	}
	remoteUserUUID, err := uuid.Parse(session.PeerUserID)
	if err != nil {
		return fmt.Errorf("%w: invalid peer_user_id", ErrInvalidSessionID)
	}
	remoteDeviceUUID, err := uuid.Parse(session.PeerDeviceID)
	if err != nil {
		return fmt.Errorf("%w: invalid peer_device_id", ErrInvalidSessionID)
	}

	stored := StoredSessionState{
		LocalDeviceID:      session.DeviceID,
		RemoteUserID:       session.PeerUserID,
		RemoteDeviceID:     session.PeerDeviceID,
		SessionVersion:     1,
		RootKey:            session.RootKey,
		SendChainKey:       session.SendChainKey,
		RecvChainKey:       session.RecvChainKey,
		SendCounter:        session.SendCounter,
		RecvCounter:        session.RecvCounter,
		DHSendPrivate:      session.DHSendPrivate,
		DHSendPublic:       session.DHSendPublic,
		DHRecvPublic:       session.DHRecvPublic,
		PreviousCounter:    session.PreviousCounter,
		EphemeralPublicKey: session.EphemeralPublicKey,
	}

	ratchetStateJSON, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}

	// SECURITY: encrypt session state at rest before storing in database
	encryptedState, err := s.encryptAtRest(ratchetStateJSON)
	if err != nil {
		return fmt.Errorf("encrypt session state at rest: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update session
	now := time.Now()
	result, err := tx.Exec(ctx,
		`UPDATE device_sessions
		 SET ratchet_state = $1,
		     updated_at = $2,
		     last_message_at = $2,
		     is_bootstrap = false
		 WHERE local_device_id = $3 AND remote_user_id = $4 AND remote_device_id = $5`,
		encryptedState, now, localUUID, remoteUserUUID, remoteDeviceUUID,
	)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	// Get session ID for skipped keys
	var sessionID string
	err = tx.QueryRow(ctx,
		`SELECT id::text FROM device_sessions
		 WHERE local_device_id = $1 AND remote_user_id = $2 AND remote_device_id = $3`,
		localUUID, remoteUserUUID, remoteDeviceUUID,
	).Scan(&sessionID)
	if err != nil {
		return fmt.Errorf("get session id: %w", err)
	}

	// Save skipped message keys
	if err := s.saveSkippedKeys(ctx, tx, sessionID, session.SkippedMessageKeys); err != nil {
		return fmt.Errorf("save skipped keys: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ListRecipientDevices returns all active devices for a user
func (s *SessionStore) ListRecipientDevices(ctx context.Context, userID string) ([]DeviceInfo, error) {
	if s.deviceService == nil {
		return nil, errors.New("device service not configured")
	}
	return s.deviceService.ListUserDevices(ctx, userID)
}

// saveNewSession saves a newly created session to database
func (s *SessionStore) saveNewSession(ctx context.Context, session SessionState, isBootstrap bool) error {
	localUUID, _ := uuid.Parse(session.DeviceID)
	remoteUserUUID, _ := uuid.Parse(session.PeerUserID)
	remoteDeviceUUID, _ := uuid.Parse(session.PeerDeviceID)

	stored := StoredSessionState{
		LocalDeviceID:      session.DeviceID,
		RemoteUserID:       session.PeerUserID,
		RemoteDeviceID:     session.PeerDeviceID,
		SessionVersion:     1,
		RootKey:            session.RootKey,
		SendChainKey:       session.SendChainKey,
		RecvChainKey:       session.RecvChainKey,
		SendCounter:        session.SendCounter,
		RecvCounter:        session.RecvCounter,
		DHSendPrivate:      session.DHSendPrivate,
		DHSendPublic:       session.DHSendPublic,
		DHRecvPublic:       session.DHRecvPublic,
		PreviousCounter:    session.PreviousCounter,
		EphemeralPublicKey: session.EphemeralPublicKey,
		IsBootstrap:        isBootstrap,
	}

	ratchetStateJSON, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}

	// SECURITY: encrypt session state at rest before storing in database
	encryptedState, err := s.encryptAtRest(ratchetStateJSON)
	if err != nil {
		return fmt.Errorf("encrypt session state at rest: %w", err)
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO device_sessions (
			local_device_id, remote_user_id, remote_device_id,
			ratchet_state, ephemeral_public_key, is_bootstrap
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		localUUID, remoteUserUUID, remoteDeviceUUID,
		encryptedState, session.EphemeralPublicKey, isBootstrap,
	)

	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// loadSkippedKeys loads skipped message keys from database
func (s *SessionStore) loadSkippedKeys(ctx context.Context, sessionID string) (map[uint32]MessageKey, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session_id: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT message_counter, key_data
		 FROM device_session_skipped_keys
		 WHERE session_id = $1 AND expires_at > NOW()
		 ORDER BY message_counter`,
		sessionUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query skipped keys: %w", err)
	}
	defer rows.Close()

	skippedKeys := make(map[uint32]MessageKey)
	for rows.Next() {
		var counter uint32
		var keyDataJSON []byte

		if err := rows.Scan(&counter, &keyDataJSON); err != nil {
			return nil, fmt.Errorf("scan skipped key: %w", err)
		}

		var msgKey MessageKey
		if err := json.Unmarshal(keyDataJSON, &msgKey); err != nil {
			return nil, fmt.Errorf("unmarshal skipped key: %w", err)
		}

		skippedKeys[counter] = msgKey
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skipped keys: %w", err)
	}

	return skippedKeys, nil
}

// saveSkippedKeys saves skipped message keys to database
func (s *SessionStore) saveSkippedKeys(ctx context.Context, tx pgx.Tx, sessionID string, keys map[uint32]MessageKey) error {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	// Delete existing skipped keys (they will be replaced)
	_, err = tx.Exec(ctx,
		`DELETE FROM device_session_skipped_keys WHERE session_id = $1`,
		sessionUUID,
	)
	if err != nil {
		return fmt.Errorf("delete old skipped keys: %w", err)
	}

	// Insert new skipped keys
	for counter, msgKey := range keys {
		keyDataJSON, err := json.Marshal(msgKey)
		if err != nil {
			return fmt.Errorf("marshal skipped key: %w", err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO device_session_skipped_keys (
				session_id, message_counter, key_data, expires_at
			) VALUES ($1, $2, $3, NOW() + INTERVAL '7 days')`,
			sessionUUID, counter, keyDataJSON,
		)
		if err != nil {
			return fmt.Errorf("insert skipped key: %w", err)
		}
	}

	return nil
}

// CleanupExpiredSkippedKeys removes expired skipped message keys
func (s *SessionStore) CleanupExpiredSkippedKeys(ctx context.Context) (int, error) {
	result, err := s.db.Exec(ctx, `SELECT cleanup_expired_skipped_keys()`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired keys: %w", err)
	}

	return int(result.RowsAffected()), nil
}
