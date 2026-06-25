package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"golang.org/x/crypto/curve25519"
)

type fakeKeyStore struct {
	bundles      map[string]DeviceKeyBundle
	consumeCalls int
	saveCalls    int
	saved        []SessionState
	consumeErr   error
	consumeKey   []byte
}

func (f *fakeKeyStore) GetBundle(_ context.Context, userID, deviceID string) (DeviceKeyBundle, error) {
	if f.bundles == nil {
		return DeviceKeyBundle{}, errors.New("bundle not found")
	}
	key, ok := f.bundles[userID+":"+deviceID]
	if !ok {
		return DeviceKeyBundle{}, errors.New("bundle not found")
	}
	return key, nil
}

func (f *fakeKeyStore) ConsumeOneTimePreKey(_ context.Context, userID, deviceID string) ([]byte, uint32, error) {
	f.consumeCalls++
	if f.consumeErr != nil {
		return nil, 0, f.consumeErr
	}
	return f.consumeKey, 7, nil
}

func (f *fakeKeyStore) SaveSession(_ context.Context, session SessionState) error {
	f.saveCalls++
	f.saved = append(f.saved, session)
	return nil
}

func generateTestKeyPair(t *testing.T) (privateKey, publicKey []byte) {
	t.Helper()
	privateKey = make([]byte, x25519KeySize)
	if _, err := rand.Read(privateKey); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("X25519: %v", err)
	}
	return privateKey, publicKey
}

func TestStartSessionRejectsIncompleteIdentifiers(t *testing.T) {
	service := NewX3DHService(&fakeKeyStore{})
	_, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "",
		IdentityPrivateKey: bytes.Repeat([]byte{0x01}, 32),
		IdentityPublicKey:  bytes.Repeat([]byte{0x02}, 32),
	}, "bob", "device-1")
	if err == nil || err.Error() != "local user and device identifiers are required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartSessionRejectsIncompletePeerBundle(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	service := NewX3DHService(&fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": {
				UserID:      "bob",
				DeviceID:    "device-1",
				IdentityKey: []byte("peer-id"),
			},
		},
	})
	_, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err == nil {
		t.Fatal("expected peer bundle validation error")
	}
}

func TestStartSessionConsumesPeerOneTimePreKeyWhenPresent(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	_, bobIdentity := generateTestKeyPair(t)
	_, bobSignedPreKey := generateTestKeyPair(t)
	_, bobOTPK := generateTestKeyPair(t)

	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": {
				UserID:        "bob",
				DeviceID:      "device-1",
				IdentityKey:   bobIdentity,
				SignedPreKey:  bobSignedPreKey,
				OneTimePreKey: bobOTPK,
			},
		},
		consumeKey: bobOTPK,
	}
	service := NewX3DHService(store)

	session, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if store.consumeCalls != 1 {
		t.Fatalf("consumeCalls = %d, want 1", store.consumeCalls)
	}
	if session.UserID != "alice" || session.PeerUserID != "bob" {
		t.Fatalf("unexpected session metadata: %+v", session)
	}
	if len(session.EphemeralPublicKey) != x25519KeySize {
		t.Fatalf("ephemeral public key length = %d, want %d", len(session.EphemeralPublicKey), x25519KeySize)
	}
}

func TestStartSessionRejectsMalformedKeyMaterial(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": {
				UserID:       "bob",
				DeviceID:     "device-1",
				IdentityKey:  []byte("peer-id"),
				SignedPreKey: []byte("peer-spk"),
			},
		},
	}
	service := NewX3DHService(store)

	_, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: []byte("short"),
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err == nil {
		t.Fatal("StartSession() expected malformed key material error")
	}

	_, err = service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err == nil {
		t.Fatal("StartSession() expected malformed peer key material error")
	}
}

func TestStartSessionPersistsGeneratedSession(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	_, bobIdentity := generateTestKeyPair(t)
	_, bobSignedPreKey := generateTestKeyPair(t)

	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": {
				UserID:       "bob",
				DeviceID:     "device-1",
				IdentityKey:  bobIdentity,
				SignedPreKey: bobSignedPreKey,
			},
		},
	}
	service := NewX3DHService(store)

	session, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if store.saveCalls != 1 {
		t.Fatalf("saveCalls = %d, want 1", store.saveCalls)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved sessions = %d, want 1", len(store.saved))
	}
	if store.saved[0].UserID != session.UserID || store.saved[0].PeerUserID != session.PeerUserID {
		t.Fatalf("saved session metadata mismatch: %+v vs %+v", store.saved[0], session)
	}
	if len(store.saved[0].RootKey) != x25519KeySize {
		t.Fatalf("root key length = %d, want %d", len(store.saved[0].RootKey), x25519KeySize)
	}
}

func TestX3DHSharedSecretUsesRealECDH(t *testing.T) {
	aliceIdentityPriv, aliceIdentityPub := generateTestKeyPair(t)
	_, bobIdentityPub := generateTestKeyPair(t)
	_, bobSignedPreKeyPub := generateTestKeyPair(t)
	ephemeralPriv, _ := generateTestKeyPair(t)

	// Placeholder hash would produce identical output regardless of private keys.
	secretA, err := deriveX3DHSharedSecret(aliceIdentityPriv, ephemeralPriv, DeviceKeyBundle{
		IdentityKey:  bobIdentityPub,
		SignedPreKey: bobSignedPreKeyPub,
	}, nil)
	if err != nil {
		t.Fatalf("deriveX3DHSharedSecret: %v", err)
	}

	wrongPriv, _ := generateTestKeyPair(t)
	secretB, err := deriveX3DHSharedSecret(wrongPriv, ephemeralPriv, DeviceKeyBundle{
		IdentityKey:  bobIdentityPub,
		SignedPreKey: bobSignedPreKeyPub,
	}, nil)
	if err != nil {
		t.Fatalf("deriveX3DHSharedSecret wrong key: %v", err)
	}

	if bytes.Equal(secretA, secretB) {
		t.Fatal("X3DH shared secret must depend on private key material")
	}
	_ = aliceIdentityPub
}
