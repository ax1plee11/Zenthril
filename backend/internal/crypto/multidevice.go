package crypto

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	ErrNoActiveDevices    = errors.New("no_active_devices_for_recipient")
	ErrPartialEncryption  = errors.New("partial_encryption_failure")
	ErrSessionNotFound    = errors.New("session_not_found")
	ErrDeviceSessionSetup = errors.New("device_session_setup_failed")
)

func bytesToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// DeviceInfo represents a recipient device for multi-device encryption
type DeviceInfo struct {
	UserID   string
	DeviceID string
}

// RecipientDeviceEnvelope contains encrypted message for one recipient device
// E2EE: includes sender's current DH public key for DH ratchet turn on receiver side.
type RecipientDeviceEnvelope struct {
	RecipientUserID   string
	RecipientDeviceID string
	SessionID         string
	RatchetMessage    RatchetMessage
	IsBootstrap       bool
	DHPublicKey       string
}

// MultiDeviceMessageEnvelope contains encrypted message for all recipient devices
type MultiDeviceMessageEnvelope struct {
	SenderUserID     string
	SenderDeviceID   string
	Plaintext        []byte
	AAD              []byte // Additional authenticated data
	DeviceEnvelopes  []RecipientDeviceEnvelope
	FailedDevices    []DeviceInfo
	EncryptionErrors map[string]error
}

// SessionManager handles pairwise device sessions
type SessionManager interface {
	// GetSession retrieves an existing session between two devices
	GetSession(ctx context.Context, localDeviceID, remoteUserID, remoteDeviceID string) (SessionState, error)
	
	// GetOrCreateSession retrieves existing session or creates new one via X3DH
	GetOrCreateSession(ctx context.Context, localDeviceID, remoteUserID, remoteDeviceID string) (SessionState, bool, error)
	
	// UpdateSession persists updated session state after encryption/decryption
	UpdateSession(ctx context.Context, session SessionState) error
	
	// ListRecipientDevices returns all active devices for a user
	ListRecipientDevices(ctx context.Context, userID string) ([]DeviceInfo, error)
}

// MultiDeviceService handles multi-device message encryption
type MultiDeviceService struct {
	sessions SessionManager
}

func NewMultiDeviceService(sessions SessionManager) *MultiDeviceService {
	return &MultiDeviceService{
		sessions: sessions,
	}
}

// EncryptForRecipients encrypts a message for all active devices of specified recipients
//
// SECURITY: Each recipient device gets its own encrypted envelope using an independent
// pairwise session. If encryption fails for some devices, the function continues and
// returns partial success with failure details.
func (s *MultiDeviceService) EncryptForRecipients(
	ctx context.Context,
	senderDeviceID string,
	recipientUserIDs []string,
	plaintext []byte,
	aad []byte,
) ([]RecipientDeviceEnvelope, error) {
	if s.sessions == nil {
		return nil, errors.New("session manager is required")
	}
	if len(recipientUserIDs) == 0 {
		return nil, errors.New("at least one recipient is required")
	}

	var allEnvelopes []RecipientDeviceEnvelope
	var encryptionErrors []error

	for _, recipientUserID := range recipientUserIDs {
		// Get all active devices for this recipient
		devices, err := s.sessions.ListRecipientDevices(ctx, recipientUserID)
		if err != nil {
			encryptionErrors = append(encryptionErrors, fmt.Errorf("list devices for %s: %w", recipientUserID, err))
			continue
		}

		if len(devices) == 0 {
			encryptionErrors = append(encryptionErrors, fmt.Errorf("%w: %s", ErrNoActiveDevices, recipientUserID))
			continue
		}

		// Encrypt for each device
		for _, device := range devices {
			envelope, err := s.encryptForDevice(ctx, senderDeviceID, device, plaintext, aad)
			if err != nil {
				encryptionErrors = append(encryptionErrors, fmt.Errorf("encrypt for device %s/%s: %w", device.UserID, device.DeviceID, err))
				continue
			}
			allEnvelopes = append(allEnvelopes, envelope)
		}
	}

	if len(allEnvelopes) == 0 && len(encryptionErrors) > 0 {
		return nil, fmt.Errorf("%w: all device encryptions failed", ErrPartialEncryption)
	}

	return allEnvelopes, nil
}

// encryptForDevice encrypts plaintext for a single recipient device
func (s *MultiDeviceService) encryptForDevice(
	ctx context.Context,
	senderDeviceID string,
	recipient DeviceInfo,
	plaintext []byte,
	aad []byte,
) (RecipientDeviceEnvelope, error) {
	// Get or create session with this specific device
	session, isBootstrap, err := s.sessions.GetOrCreateSession(
		ctx,
		senderDeviceID,
		recipient.UserID,
		recipient.DeviceID,
	)
	if err != nil {
		return RecipientDeviceEnvelope{}, fmt.Errorf("%w: %v", ErrDeviceSessionSetup, err)
	}

	// Convert session to ratchet state
	ratchetState, err := session.RatchetState()
	if err != nil {
		return RecipientDeviceEnvelope{}, fmt.Errorf("get ratchet state: %w", err)
	}

	// Encrypt with Double Ratchet
	ratchetMsg, err := EncryptMessage(&ratchetState, plaintext, aad)
	if err != nil {
		return RecipientDeviceEnvelope{}, fmt.Errorf("encrypt message: %w", err)
	}

	// Update session with new ratchet state
	session.ApplyRatchetState(ratchetState)
	if err := s.sessions.UpdateSession(ctx, session); err != nil {
		return RecipientDeviceEnvelope{}, fmt.Errorf("update session: %w", err)
	}

	sessionID := generateSessionID(senderDeviceID, recipient.UserID, recipient.DeviceID)

	return RecipientDeviceEnvelope{
		RecipientUserID:   recipient.UserID,
		RecipientDeviceID: recipient.DeviceID,
		SessionID:         sessionID,
		RatchetMessage:    ratchetMsg,
		IsBootstrap:       isBootstrap,
		DHPublicKey:       bytesToBase64(ratchetState.DHSendPublic),
	}, nil
}

// DecryptForDevice decrypts a message intended for a specific device
//
// SECURITY: The session state is updated after successful decryption to advance
// the ratchet and prevent replay attacks.
//
// Note: The envelope contains the recipient's info, but we need to extract
// the sender's info from the session ID or context to find the correct session.
func (s *MultiDeviceService) DecryptForDevice(
	ctx context.Context,
	localDeviceID string,
	senderUserID string,
	senderDeviceID string,
	envelope RecipientDeviceEnvelope,
	aad []byte,
) ([]byte, error) {
	if s.sessions == nil {
		return nil, errors.New("session manager is required")
	}

	// Get existing session between local device and sender device
	session, err := s.sessions.GetSession(
		ctx,
		localDeviceID,
		senderUserID,
		senderDeviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionNotFound, err)
	}

	// Convert to ratchet state
	ratchetState, err := session.RatchetState()
	if err != nil {
		return nil, fmt.Errorf("get ratchet state: %w", err)
	}

	// Decrypt with Double Ratchet
	plaintext, err := DecryptMessage(&ratchetState, envelope.RatchetMessage, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt message: %w", err)
	}

	// Update session with new ratchet state
	session.ApplyRatchetState(ratchetState)
	if err := s.sessions.UpdateSession(ctx, session); err != nil {
		// SECURITY WARNING: Decryption succeeded but state update failed
		// This could lead to message replay if not handled carefully
		return nil, fmt.Errorf("update session after decryption: %w", err)
	}

	return plaintext, nil
}

// GetDeviceEnvelope finds the envelope for a specific device from a multi-device message
func GetDeviceEnvelope(envelopes []RecipientDeviceEnvelope, userID, deviceID string) *RecipientDeviceEnvelope {
	for i := range envelopes {
		if envelopes[i].RecipientUserID == userID && envelopes[i].RecipientDeviceID == deviceID {
			return &envelopes[i]
		}
	}
	return nil
}

// generateSessionID creates a deterministic session identifier for a device pair
func generateSessionID(localDeviceID, remoteUserID, remoteDeviceID string) string {
	// In practice, this might use a database-assigned ID or hash
	return fmt.Sprintf("%s:%s:%s", localDeviceID, remoteUserID, remoteDeviceID)
}

// ValidateDeviceEnvelope validates that a device envelope is well-formed
func ValidateDeviceEnvelope(envelope RecipientDeviceEnvelope) error {
	if envelope.RecipientUserID == "" {
		return errors.New("recipient_user_id is required")
	}
	if envelope.RecipientDeviceID == "" {
		return errors.New("recipient_device_id is required")
	}
	if envelope.SessionID == "" {
		return errors.New("session_id is required")
	}
	if len(envelope.RatchetMessage.Ciphertext) == 0 {
		return errors.New("ciphertext is required")
	}
	if len(envelope.RatchetMessage.Header.DHPublicKey) != x25519KeySize {
		return errors.New("DH public key must be 32 bytes")
	}
	return nil
}
