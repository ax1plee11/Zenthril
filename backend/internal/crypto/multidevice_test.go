package crypto

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type fakeSessionManager struct {
	sessions      map[string]SessionState
	devices       map[string][]DeviceInfo
	createCalled  int
	updateCalled  int
	shouldFailGet bool
}

func newFakeSessionManager() *fakeSessionManager {
	return &fakeSessionManager{
		sessions: make(map[string]SessionState),
		devices:  make(map[string][]DeviceInfo),
	}
}

func (f *fakeSessionManager) GetSession(ctx context.Context, localDeviceID, remoteUserID, remoteDeviceID string) (SessionState, error) {
	if f.shouldFailGet {
		return SessionState{}, errors.New("session not found")
	}
	key := sessionKey(localDeviceID, remoteUserID, remoteDeviceID)
	session, ok := f.sessions[key]
	if !ok {
		return SessionState{}, ErrSessionNotFound
	}
	return session, nil
}

func (f *fakeSessionManager) GetOrCreateSession(ctx context.Context, localDeviceID, remoteUserID, remoteDeviceID string) (SessionState, bool, error) {
	f.createCalled++
	key := sessionKey(localDeviceID, remoteDeviceID, remoteDeviceID)
	
	session, ok := f.sessions[key]
	if ok {
		return session, false, nil
	}

	// Create new session with compatible ratchet state
	shared := bytes.Repeat([]byte{0x42}, 32)
	info := []byte(localDeviceID + ":" + remoteUserID + ":" + remoteDeviceID)
	
	ratchet, _ := DeriveInitialRatchetState(shared, info, true)
	
	dhPriv, dhPub, _ := generateEphemeralKeyPair()
	peerDHPriv, peerDHPub, _ := generateEphemeralKeyPair()
	
	ratchet.DHSendPrivate = dhPriv
	ratchet.DHSendPublic = dhPub
	ratchet.DHRecvPublic = peerDHPub

	session = SessionState{
		UserID:       localDeviceID,
		PeerUserID:   remoteUserID,
		DeviceID:     localDeviceID,
		PeerDeviceID: remoteDeviceID,
	}
	session.ApplyRatchetState(ratchet)

	// Also create reciprocal session for peer
	peerRatchet, _ := DeriveInitialRatchetState(shared, info, false)
	peerRatchet.DHSendPrivate = peerDHPriv
	peerRatchet.DHSendPublic = peerDHPub
	peerRatchet.DHRecvPublic = dhPub

	peerSession := SessionState{
		UserID:       remoteDeviceID,
		PeerUserID:   localDeviceID,
		DeviceID:     remoteDeviceID,
		PeerDeviceID: localDeviceID,
	}
	peerSession.ApplyRatchetState(peerRatchet)

	f.sessions[key] = session
	f.sessions[sessionKey(remoteDeviceID, localDeviceID, localDeviceID)] = peerSession

	return session, true, nil
}

func (f *fakeSessionManager) UpdateSession(ctx context.Context, session SessionState) error {
	f.updateCalled++
	key := sessionKey(session.DeviceID, session.PeerUserID, session.PeerDeviceID)
	f.sessions[key] = session
	return nil
}

func (f *fakeSessionManager) ListRecipientDevices(ctx context.Context, userID string) ([]DeviceInfo, error) {
	devices, ok := f.devices[userID]
	if !ok {
		return nil, nil
	}
	return devices, nil
}

func sessionKey(local, remoteUser, remoteDevice string) string {
	return local + ":" + remoteUser + ":" + remoteDevice
}

func TestEncryptForRecipientsMultipleDevices(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.devices["bob"] = []DeviceInfo{
		{UserID: "bob", DeviceID: "device1"},
		{UserID: "bob", DeviceID: "device2"},
	}

	svc := NewMultiDeviceService(mgr)

	envelopes, err := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		[]byte("Hello Bob from all your devices"),
		[]byte("channel:123"),
	)
	if err != nil {
		t.Fatalf("EncryptForRecipients: %v", err)
	}

	if len(envelopes) != 2 {
		t.Fatalf("expected 2 envelopes, got %d", len(envelopes))
	}

	// Verify each device has its own envelope
	envelope1 := GetDeviceEnvelope(envelopes, "bob", "device1")
	envelope2 := GetDeviceEnvelope(envelopes, "bob", "device2")

	if envelope1 == nil {
		t.Fatal("envelope for device1 not found")
	}
	if envelope2 == nil {
		t.Fatal("envelope for device2 not found")
	}

	// Verify ciphertexts are different (different sessions)
	if bytes.Equal(envelope1.RatchetMessage.Ciphertext, envelope2.RatchetMessage.Ciphertext) {
		t.Fatal("ciphertexts should be different for different devices")
	}

	if mgr.createCalled != 2 {
		t.Fatalf("expected 2 session creations, got %d", mgr.createCalled)
	}
}

func TestEncryptForRecipientsMultipleUsers(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.devices["bob"] = []DeviceInfo{
		{UserID: "bob", DeviceID: "device1"},
	}
	mgr.devices["charlie"] = []DeviceInfo{
		{UserID: "charlie", DeviceID: "device1"},
		{UserID: "charlie", DeviceID: "device2"},
	}

	svc := NewMultiDeviceService(mgr)

	envelopes, err := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob", "charlie"},
		[]byte("Hello everyone"),
		nil,
	)
	if err != nil {
		t.Fatalf("EncryptForRecipients: %v", err)
	}

	// 1 device for bob + 2 devices for charlie = 3 envelopes
	if len(envelopes) != 3 {
		t.Fatalf("expected 3 envelopes, got %d", len(envelopes))
	}

	bobEnv := GetDeviceEnvelope(envelopes, "bob", "device1")
	charlieEnv1 := GetDeviceEnvelope(envelopes, "charlie", "device1")
	charlieEnv2 := GetDeviceEnvelope(envelopes, "charlie", "device2")

	if bobEnv == nil || charlieEnv1 == nil || charlieEnv2 == nil {
		t.Fatal("some envelopes are missing")
	}
}

func TestEncryptForRecipientsNoDevices(t *testing.T) {
	mgr := newFakeSessionManager()
	// bob has no devices registered

	svc := NewMultiDeviceService(mgr)

	_, err := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		[]byte("Hello"),
		nil,
	)
	
	if err == nil {
		t.Fatal("expected error when recipient has no devices")
	}
	if !errors.Is(err, ErrPartialEncryption) {
		t.Fatalf("expected ErrPartialEncryption, got %v", err)
	}
}

func TestDecryptForDevice(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.devices["bob"] = []DeviceInfo{
		{UserID: "bob", DeviceID: "device1"},
	}

	svc := NewMultiDeviceService(mgr)

	plaintext := []byte("Secret message")
	aad := []byte("context")

	// Alice encrypts for Bob's device
	envelopes, err := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		plaintext,
		aad,
	)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if len(envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envelopes))
	}

	// Bob decrypts with his device
	// Bob needs to specify that Alice is the sender
	decrypted, err := svc.DecryptForDevice(
		context.Background(),
		"device1",           // Bob's local device
		"alice-device",      // Sender user ID (Alice's device acts as user here in test)
		"alice-device",      // Sender device ID
		envelopes[0],
		aad,
	)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptForDeviceWrongAAD(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.devices["bob"] = []DeviceInfo{
		{UserID: "bob", DeviceID: "device1"},
	}

	svc := NewMultiDeviceService(mgr)

	envelopes, _ := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		[]byte("message"),
		[]byte("original-aad"),
	)

	_, err := svc.DecryptForDevice(
		context.Background(),
		"device1",
		"alice-device",
		"alice-device",
		envelopes[0],
		[]byte("wrong-aad"),
	)

	if err == nil {
		t.Fatal("expected decryption to fail with wrong AAD")
	}
}

func TestDecryptForDeviceNoSession(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.shouldFailGet = true

	svc := NewMultiDeviceService(mgr)

	envelope := RecipientDeviceEnvelope{
		RecipientUserID:   "bob",
		RecipientDeviceID: "device1",
		SessionID:         "session123",
		RatchetMessage: RatchetMessage{
			Header:     MessageHeader{DHPublicKey: make([]byte, 32)},
			Ciphertext: []byte("ciphertext"),
		},
	}

	_, err := svc.DecryptForDevice(
		context.Background(),
		"device1",
		"sender-user",
		"sender-device",
		envelope,
		nil,
	)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestGetDeviceEnvelope(t *testing.T) {
	envelopes := []RecipientDeviceEnvelope{
		{RecipientUserID: "alice", RecipientDeviceID: "device1"},
		{RecipientUserID: "bob", RecipientDeviceID: "device1"},
		{RecipientUserID: "bob", RecipientDeviceID: "device2"},
	}

	env := GetDeviceEnvelope(envelopes, "bob", "device2")
	if env == nil {
		t.Fatal("envelope not found")
	}
	if env.RecipientUserID != "bob" || env.RecipientDeviceID != "device2" {
		t.Fatalf("wrong envelope: %+v", env)
	}

	env = GetDeviceEnvelope(envelopes, "charlie", "device1")
	if env != nil {
		t.Fatal("expected nil for non-existent device")
	}
}

func TestValidateDeviceEnvelope(t *testing.T) {
	tests := []struct {
		name      string
		envelope  RecipientDeviceEnvelope
		wantError bool
	}{
		{
			name: "valid envelope",
			envelope: RecipientDeviceEnvelope{
				RecipientUserID:   "bob",
				RecipientDeviceID: "device1",
				SessionID:         "session123",
				RatchetMessage: RatchetMessage{
					Header:     MessageHeader{DHPublicKey: make([]byte, 32)},
					Ciphertext: []byte("ciphertext"),
				},
			},
			wantError: false,
		},
		{
			name: "missing user ID",
			envelope: RecipientDeviceEnvelope{
				RecipientDeviceID: "device1",
				SessionID:         "session123",
				RatchetMessage: RatchetMessage{
					Header:     MessageHeader{DHPublicKey: make([]byte, 32)},
					Ciphertext: []byte("ciphertext"),
				},
			},
			wantError: true,
		},
		{
			name: "missing DH key",
			envelope: RecipientDeviceEnvelope{
				RecipientUserID:   "bob",
				RecipientDeviceID: "device1",
				SessionID:         "session123",
				RatchetMessage: RatchetMessage{
					Header:     MessageHeader{DHPublicKey: []byte{}},
					Ciphertext: []byte("ciphertext"),
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceEnvelope(tt.envelope)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDeviceEnvelope() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSessionReuseAcrossMessages(t *testing.T) {
	mgr := newFakeSessionManager()
	mgr.devices["bob"] = []DeviceInfo{
		{UserID: "bob", DeviceID: "device1"},
	}

	svc := NewMultiDeviceService(mgr)

	// First message
	_, err := svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		[]byte("Message 1"),
		nil,
	)
	if err != nil {
		t.Fatalf("first encrypt: %v", err)
	}

	initialCreateCount := mgr.createCalled

	// Second message - should reuse session
	_, err = svc.EncryptForRecipients(
		context.Background(),
		"alice-device",
		[]string{"bob"},
		[]byte("Message 2"),
		nil,
	)
	if err != nil {
		t.Fatalf("second encrypt: %v", err)
	}

	// Session should be reused, not created again
	if mgr.createCalled != initialCreateCount+1 {
		t.Fatalf("expected session reuse, but createCalled changed from %d to %d", 
			initialCreateCount, mgr.createCalled)
	}
}
