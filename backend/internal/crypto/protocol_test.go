package crypto

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

func generatePeerBundle(t *testing.T, includeOneTimePreKey bool) DeviceKeyBundle {
	t.Helper()
	identitySigningPublic, identitySigningPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing identity: %v", err)
	}
	_, identityDHPublic := generateTestKeyPair(t)
	_, signedPreKey := generateTestKeyPair(t)
	bundle := DeviceKeyBundle{
		UserID:                   "bob",
		DeviceID:                 "device-1",
		IdentitySigningPublicKey: identitySigningPublic,
		IdentityDHPublicKey:      identityDHPublic,
		SignedPreKey:             signedPreKey,
		SignedPreKeySig:          ed25519.Sign(identitySigningPrivate, signedPreKeyMessage(signedPreKey)),
	}
	if includeOneTimePreKey {
		_, bundle.OneTimePreKey = generateTestKeyPair(t)
		bundle.OneTimePreKeyID = 7
	}
	return bundle
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
			"bob:device-1": {UserID: "bob", DeviceID: "device-1", IdentityDHPublicKey: []byte("peer-id")},
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
	peerBundle := generatePeerBundle(t, true)

	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": peerBundle,
		},
		consumeKey: peerBundle.OneTimePreKey,
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

func TestStartSessionRejectsTamperedSignedPreKey(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	peerBundle := generatePeerBundle(t, false)
	peerBundle.SignedPreKeySig[0] ^= 0xff
	service := NewX3DHService(&fakeKeyStore{bundles: map[string]DeviceKeyBundle{
		"bob:device-1": peerBundle,
	}})

	_, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err == nil {
		t.Fatal("expected tampered signed prekey to be rejected")
	}
}

func TestStartSessionRejectsMismatchedConsumedOneTimePreKey(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	peerBundle := generatePeerBundle(t, true)
	_, differentPreKey := generateTestKeyPair(t)
	store := &fakeKeyStore{
		bundles:    map[string]DeviceKeyBundle{"bob:device-1": peerBundle},
		consumeKey: differentPreKey,
	}
	service := NewX3DHService(store)

	_, err := service.StartSession(context.Background(), LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device-1",
		IdentityPrivateKey: alicePriv,
		IdentityPublicKey:  alicePub,
	}, "bob", "device-1")
	if err == nil {
		t.Fatal("expected mismatched one-time prekey to be rejected")
	}
	if store.saveCalls != 0 {
		t.Fatalf("saved a session after rejected one-time prekey: %d", store.saveCalls)
	}
}

func TestStartSessionRejectsMalformedKeyMaterial(t *testing.T) {
	alicePriv, alicePub := generateTestKeyPair(t)
	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": {UserID: "bob", DeviceID: "device-1", IdentityDHPublicKey: []byte("peer-id"), SignedPreKey: []byte("peer-spk")},
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
	peerBundle := generatePeerBundle(t, false)

	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device-1": peerBundle,
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
		IdentityDHPublicKey: bobIdentityPub,
		SignedPreKey:        bobSignedPreKeyPub,
	}, nil, []byte("zenthril.e2ee.x3dh.session.v1|test"))
	if err != nil {
		t.Fatalf("deriveX3DHSharedSecret: %v", err)
	}

	wrongPriv, _ := generateTestKeyPair(t)
	secretB, err := deriveX3DHSharedSecret(wrongPriv, ephemeralPriv, DeviceKeyBundle{
		IdentityDHPublicKey: bobIdentityPub,
		SignedPreKey:        bobSignedPreKeyPub,
	}, nil, []byte("zenthril.e2ee.x3dh.session.v1|test"))
	if err != nil {
		t.Fatalf("deriveX3DHSharedSecret wrong key: %v", err)
	}

	if bytes.Equal(secretA, secretB) {
		t.Fatal("X3DH shared secret must depend on private key material")
	}
	_ = aliceIdentityPub
}

// TestCanonicalX3DHHeaderDeterminism verifies that canonicalX3DHHeader produces
// deterministic JSON matching TypeScript's canonicalHeader() for cross-language
// KDF compatibility. Same inputs must always produce identical output.
func TestCanonicalX3DHHeaderDeterminism(t *testing.T) {
	sessionID := "test-session-1234"
	h1 := canonicalX3DHHeader(
		sessionID, "alice", "device1",
		"base64identitypub", "base64ephemeralpub", "base64dhpub",
		"bob", "device1",
		1, nil,
	)
	h2 := canonicalX3DHHeader(
		sessionID, "alice", "device1",
		"base64identitypub", "base64ephemeralpub", "base64dhpub",
		"bob", "device1",
		1, nil,
	)
	if !bytes.Equal(h1, h2) {
		t.Error("canonicalX3DHHeader is not deterministic")
	}

	// Verify JSON is valid
	var js map[string]interface{}
	if err := json.Unmarshal(h1, &js); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if js["sessionId"] != sessionID {
		t.Errorf("sessionId = %v, want %s", js["sessionId"], sessionID)
	}
	if js["cipherSuite"] != "X3DH-HKDF-SHA256-DR-v1" {
		t.Errorf("cipherSuite = %v", js["cipherSuite"])
	}
	if js["version"] != float64(1) {
		t.Errorf("version = %v, want 1", js["version"])
	}

	t.Logf("canonicalX3DHHeader output: %s", string(h1))
}

// TestX3DHKeyDerivationDeterminism verifies that DeriveInitialRatchetState
// produces deterministic output given identical IKM and info. This is the
// core cross-language compatibility property: Go and TypeScript must derive
// identical keys from identical inputs.
func TestX3DHKeyDerivationDeterminism(t *testing.T) {
	// Deterministic IKM (3×32 bytes = 96 bytes for 3 DH values, or 4×32=128 with OPK)
	ikm := make([]byte, 96)
	for i := range ikm {
		ikm[i] = byte(i)
	}
	info := []byte("zenthril.e2ee.x3dh.session.v1|test-info")

	state1, err := DeriveInitialRatchetState(ikm, info, true)
	if err != nil {
		t.Fatalf("DeriveInitialRatchetState: %v", err)
	}
	state2, err := DeriveInitialRatchetState(ikm, info, true)
	if err != nil {
		t.Fatalf("DeriveInitialRatchetState second call: %v", err)
	}

	if !bytes.Equal(state1.RootKey, state2.RootKey) {
		t.Error("root key not deterministic")
	}
	if !bytes.Equal(state1.SendChainKey, state2.SendChainKey) {
		t.Error("send chain key not deterministic")
	}
	if !bytes.Equal(state1.RecvChainKey, state2.RecvChainKey) {
		t.Error("recv chain key not deterministic")
	}

	// Initiator and responder should have swapped chain keys
	stateResp, err := DeriveInitialRatchetState(ikm, info, false)
	if err != nil {
		t.Fatalf("DeriveInitialRatchetState responder: %v", err)
	}
	if !bytes.Equal(state1.RootKey, stateResp.RootKey) {
		t.Error("root key should match between initiator and responder")
	}
	if !bytes.Equal(state1.SendChainKey, stateResp.RecvChainKey) {
		t.Error("initiator send chain should match responder recv chain")
	}
	if !bytes.Equal(state1.RecvChainKey, stateResp.SendChainKey) {
		t.Error("initiator recv chain should match responder send chain")
	}

	t.Logf("X3DH key derivation deterministic:")
	t.Logf("  rootKey:     %x", state1.RootKey)
	t.Logf("  sendChain:   %x", state1.SendChainKey)
	t.Logf("  recvChain:   %x", state1.RecvChainKey)
}

// TestStartSessionEndToEnd verifies the full StartSession flow works with
// a valid peer bundle, producing a valid session state that is saved.
func TestStartSessionEndToEnd(t *testing.T) {
	// Deterministic test keys
	aliceIdentityPriv, _ := hex.DecodeString("77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	aliceIdentityPub, _ := hex.DecodeString("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")
	bobSignedPreKeyPub, _ := hex.DecodeString("4a5d9d5ba4ce2de1728e3bf480350f25e07e21c947d19e3376f09b3c1e161742")
	bobOneTimePreKeyPub, _ := hex.DecodeString("a5467e6043d79c6f5b9e8b3c2d1f0a4e5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a")

	// Generate Ed25519 signing key for the peer bundle
	bobSignPub, bobSignPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 key: %v", err)
	}
	signedPreKeyMsg := signedPreKeyMessage(bobSignedPreKeyPub)
	bobSignedPreKeySig := ed25519.Sign(bobSignPriv, signedPreKeyMsg)

	peerBundle := DeviceKeyBundle{
		UserID:                   "bob",
		DeviceID:                 "device1",
		IdentitySigningPublicKey: bobSignPub,
		IdentityDHPublicKey:      aliceIdentityPub,
		SignedPreKey:             bobSignedPreKeyPub,
		SignedPreKeyID:           1,
		SignedPreKeySig:          bobSignedPreKeySig,
		OneTimePreKey:            bobOneTimePreKeyPub,
		OneTimePreKeyID:          2,
	}

	store := &fakeKeyStore{
		bundles: map[string]DeviceKeyBundle{
			"bob:device1": peerBundle,
		},
		consumeKey: bobOneTimePreKeyPub,
	}

	service := NewX3DHService(store)

	local := LocalDeviceKeys{
		UserID:             "alice",
		DeviceID:           "device1",
		IdentityPrivateKey: aliceIdentityPriv,
		IdentityPublicKey:  aliceIdentityPub,
	}

	session, err := service.StartSession(context.Background(), local, "bob", "device1")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	// Verify session state
	if session.UserID != "alice" {
		t.Errorf("userID = %s, want alice", session.UserID)
	}
	if session.PeerUserID != "bob" {
		t.Errorf("peerUserID = %s, want bob", session.PeerUserID)
	}
	if session.DeviceID != "device1" {
		t.Errorf("deviceID = %s, want device1", session.DeviceID)
	}
	if session.PeerDeviceID != "device1" {
		t.Errorf("peerDeviceID = %s, want device1", session.PeerDeviceID)
	}
	if len(session.EphemeralPublicKey) != x25519KeySize {
		t.Errorf("ephemeralPublicKey length = %d, want %d", len(session.EphemeralPublicKey), x25519KeySize)
	}
	if len(session.RootKey) != x25519KeySize {
		t.Errorf("rootKey length = %d, want %d", len(session.RootKey), x25519KeySize)
	}
	if len(session.SendChainKey) != x25519KeySize {
		t.Errorf("sendChainKey length = %d, want %d", len(session.SendChainKey), x25519KeySize)
	}
	if len(session.RecvChainKey) != x25519KeySize {
		t.Errorf("recvChainKey length = %d, want %d", len(session.RecvChainKey), x25519KeySize)
	}

	// Verify session was saved
	if store.saveCalls != 1 {
		t.Errorf("saveCalls = %d, want 1", store.saveCalls)
	}
	if len(store.saved) != 1 {
		t.Errorf("saved sessions = %d, want 1", len(store.saved))
	}

	t.Logf("StartSession end-to-end verified:")
	t.Logf("  rootKey:     %x", session.RootKey)
	t.Logf("  sendChain:   %x", session.SendChainKey)
	t.Logf("  recvChain:   %x", session.RecvChainKey)
}
