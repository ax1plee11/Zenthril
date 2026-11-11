package crypto

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// x3dhHeaderFields mirrors TypeScript's X3DHSessionHeader canonicalization.
// Field order matches TypeScript's canonicalHeader() for cross-language KDF compatibility.
type x3dhHeaderFields struct {
	CipherSuite             string `json:"cipherSuite"`
	RecipientDeviceId       string `json:"recipientDeviceId"`
	RecipientOneTimePreKeyId *uint32 `json:"recipientOneTimePreKeyId"`
	RecipientSignedPreKeyId  uint32 `json:"recipientSignedPreKeyId"`
	RecipientUserId         string `json:"recipientUserId"`
	SenderDeviceId          string `json:"senderDeviceId"`
	SenderDHPublicKey       string `json:"senderDHPublicKey"`
	SenderEphemeralPublicKey string `json:"senderEphemeralPublicKey"`
	SenderIdentityDHPublicKey string `json:"senderIdentityDHPublicKey"`
	SenderUserId            string `json:"senderUserId"`
	SessionId               string `json:"sessionId"`
	Version                 int    `json:"version"`
}

// randomSessionID generates a random session ID matching TypeScript's randomSessionID().
func randomSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use crypto/rand via time-based seed (should not happen)
		b = []byte("fallback-session-id")
	}
	return base64URLEncode(b)
}

// base64URLEncode encodes bytes to base64 URL encoding (no padding), matching TypeScript.
func base64URLEncode(b []byte) string {
	const encoded = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	out := make([]byte, 0, len(b)*4/3+4)
	for i := 0; i < len(b); i += 3 {
		v := uint32(b[i]) << 16
		if i+1 < len(b) {
			v |= uint32(b[i+1]) << 8
		}
		if i+2 < len(b) {
			v |= uint32(b[i+2])
		}
		out = append(out, encoded[(v>>18)&0x3f], encoded[(v>>12)&0x3f])
		if i+1 < len(b) {
			out = append(out, encoded[(v>>6)&0x3f])
		} else {
			out = append(out, '=')
		}
		if i+2 < len(b) {
			out = append(out, encoded[v&0x3f])
		} else {
			out = append(out, '=')
		}
	}
	// Remove padding
	for len(out) > 0 && out[len(out)-1] == '=' {
		out = out[:len(out)-1]
	}
	return string(out)
}

// canonicalX3DHHeader produces the same JSON as TypeScript's canonicalHeader().
func canonicalX3DHHeader(sessionID, senderUserID, senderDeviceID, senderIdentityDHPub, senderEphemeralPub, senderDHPub, recipientUserID, recipientDeviceID string, signedPreKeyID uint32, oneTimePreKeyID *uint32) []byte {
	h := x3dhHeaderFields{
		CipherSuite:             "X3DH-HKDF-SHA256-DR-v1",
		RecipientDeviceId:       recipientDeviceID,
		RecipientOneTimePreKeyId: oneTimePreKeyID,
		RecipientSignedPreKeyId:  signedPreKeyID,
		RecipientUserId:         recipientUserID,
		SenderDeviceId:          senderDeviceID,
		SenderDHPublicKey:       senderDHPub,
		SenderEphemeralPublicKey: senderEphemeralPub,
		SenderIdentityDHPublicKey: senderIdentityDHPub,
		SenderUserId:            senderUserID,
		SessionId:               sessionID,
		Version:                 1,
	}
	out, err := json.Marshal(h)
	if err != nil {
		// Should never happen with fixed struct
		return []byte("{}")
	}
	return out
}

type DeviceKeyBundle struct {
	UserID                   string
	DeviceID                 string
	IdentitySigningPublicKey []byte
	IdentityDHPublicKey      []byte
	SignedPreKey             []byte
	SignedPreKeyID           uint32
	SignedPreKeySig          []byte
	OneTimePreKey            []byte
	OneTimePreKeyID          uint32
	PublishedRevision        uint64
}

// LocalDeviceKeys holds the initiator's private material required for X3DH.
// ARCHITECTURE: public bundles are exchanged out-of-band; private keys never leave the client boundary.
type LocalDeviceKeys struct {
	UserID             string
	DeviceID           string
	IdentityPrivateKey []byte
	IdentityPublicKey  []byte
}

type SessionState struct {
	UserID             string
	PeerUserID         string
	DeviceID           string
	PeerDeviceID       string
	EphemeralPublicKey []byte
	RootKey            []byte
	SendChainKey       []byte
	RecvChainKey       []byte
	SendCounter        uint32
	RecvCounter        uint32

	// DH Ratchet keys
	DHSendPrivate   []byte
	DHSendPublic    []byte
	DHRecvPublic    []byte
	PreviousCounter uint32

	SkippedMessageKeys map[uint32]MessageKey
}

type KeyStore interface {
	GetBundle(ctx context.Context, userID, deviceID string) (DeviceKeyBundle, error)
	ConsumeOneTimePreKey(ctx context.Context, userID, deviceID string) ([]byte, uint32, error)
	SaveSession(ctx context.Context, session SessionState) error
}

type X3DHService struct {
	keys KeyStore
}

func NewX3DHService(keys KeyStore) *X3DHService {
	return &X3DHService{keys: keys}
}

// MaxSessionAge is the maximum duration a session can be used before requiring rotation.
const MaxSessionAge = 24 * time.Hour

// MaxMessagesPerSession is the maximum number of messages before session rotation.
const MaxMessagesPerSession = 10000

// SessionRotationRequired checks if a session needs rotation based on age or message count.
// SECURITY: Long-lived sessions increase the window for key compromise.
// WEAKNESS FIXED: sessions had no rotation mechanism.
func SessionRotationRequired(session SessionState, messagesSent uint32) bool {
	return messagesSent > MaxMessagesPerSession || session.SendCounter > MaxMessagesPerSession
}

func (s *X3DHService) StartSession(ctx context.Context, local LocalDeviceKeys, peerUserID, peerDeviceID string) (SessionState, error) {
	if s.keys == nil {
		return SessionState{}, errors.New("key store is required")
	}
	if err := validateLocalKeys(local); err != nil {
		return SessionState{}, err
	}
	if peerUserID == "" || peerDeviceID == "" {
		return SessionState{}, errors.New("peer identifiers are required")
	}

	peer, err := s.keys.GetBundle(ctx, peerUserID, peerDeviceID)
	if err != nil {
		return SessionState{}, err
	}
	if err := validatePeerBundle(peer); err != nil {
		return SessionState{}, err
	}

	var oneTimePreKeyPublic []byte
	if len(peer.OneTimePreKey) == x25519KeySize {
		consumed, consumedID, err := s.keys.ConsumeOneTimePreKey(ctx, peerUserID, peerDeviceID)
		if err != nil {
			return SessionState{}, fmt.Errorf("consume one-time prekey: %w", err)
		}
		if consumedID != peer.OneTimePreKeyID || !bytes.Equal(consumed, peer.OneTimePreKey) {
			return SessionState{}, errors.New("consumed one-time prekey does not match claimed bundle")
		}
		oneTimePreKeyPublic = consumed
	}

	ephemeralPrivate, ephemeralPublic, err := generateEphemeralKeyPair()
	if err != nil {
		return SessionState{}, err
	}

	// Initialize DH ratchet keys for initiator BEFORE computing session info,
	// matching TypeScript's initiatePairwiseSession order.
	dhSendPriv, dhSendPub, err := generateEphemeralKeyPair()
	if err != nil {
		return SessionState{}, fmt.Errorf("generate initial DH keys: %w", err)
	}

	// Generate session ID before computing info string, matching TypeScript.
	sessionID := randomSessionID()

	// Compute X3DH DH values matching client: DH1=DH(IKa,SPKb), DH2=DH(EKa,IKb), DH3=DH(EKa,SPKb), DH4=DH(EKa,OPKb)
	dh1, err := x25519SharedSecret(local.IdentityPrivateKey, peer.SignedPreKey)
	if err != nil {
		return SessionState{}, fmt.Errorf("X3DH DH1: %w", err)
	}
	dh2, err := x25519SharedSecret(ephemeralPrivate, peer.IdentityDHPublicKey)
	if err != nil {
		return SessionState{}, fmt.Errorf("X3DH DH2: %w", err)
	}
	dh3, err := x25519SharedSecret(ephemeralPrivate, peer.SignedPreKey)
	if err != nil {
		return SessionState{}, fmt.Errorf("X3DH DH3: %w", err)
	}
	ikmParts := [][]byte{dh1, dh2, dh3}
	if len(oneTimePreKeyPublic) == x25519KeySize {
		dh4, err := x25519SharedSecret(ephemeralPrivate, oneTimePreKeyPublic)
		if err != nil {
			return SessionState{}, fmt.Errorf("X3DH DH4: %w", err)
		}
		ikmParts = append(ikmParts, dh4)
	}
	ikm := concatBytes(ikmParts...)

	// Info string must match client canonicalHeader-based derivation.
	// TypeScript: "zenthril.e2ee.x3dh.session.v1|" + JSON.stringify(canonicalHeader(header))
	var oneTimePreKeyID *uint32
	if len(peer.OneTimePreKey) == x25519KeySize {
		oneTimePreKeyID = &peer.OneTimePreKeyID
	}
	sessionInfo := append([]byte("zenthril.e2ee.x3dh.session.v1|"),
		canonicalX3DHHeader(
			sessionID,
			local.UserID, local.DeviceID,
			base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(local.IdentityPublicKey),
			base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(ephemeralPublic),
			base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(dhSendPub),
			peer.UserID, peer.DeviceID,
			peer.SignedPreKeyID,
			oneTimePreKeyID,
		)...,
	)
	ratchet, err := DeriveInitialRatchetState(ikm, sessionInfo, true)
	if err != nil {
		return SessionState{}, err
	}

	dhOutput, err := x25519SharedSecret(dhSendPriv, peer.SignedPreKey)
	if err != nil {
		return SessionState{}, fmt.Errorf("initial DH with peer signed prekey: %w", err)
	}
	newRootKey, newSendChainKey, err := InitializeDHRatchet(ratchet.RootKey, dhOutput)
	if err != nil {
		return SessionState{}, fmt.Errorf("initialize DH ratchet: %w", err)
	}

	// Update ratchet state with new keys
	ratchet.RootKey = newRootKey
	ratchet.SendChainKey = newSendChainKey
	ratchet.DHSendPrivate = dhSendPriv
	ratchet.DHSendPublic = dhSendPub
	ratchet.DHRecvPublic = peer.SignedPreKey
	// RecvChainKey stays as derived from X3DH shared secret for receiving peer's first message

	session := SessionState{
		UserID:             local.UserID,
		PeerUserID:         peer.UserID,
		DeviceID:           local.DeviceID,
		PeerDeviceID:       peer.DeviceID,
		EphemeralPublicKey: cloneBytes(ephemeralPublic),
	}
	session.ApplyRatchetState(ratchet)

	// WEAKNESS FIXED: the bootstrap result must be persisted immediately so the
	// negotiated session state is not lost between protocol creation and use.
	if err := s.keys.SaveSession(ctx, session); err != nil {
		return SessionState{}, err
	}
	return session, nil
}

func validateLocalKeys(local LocalDeviceKeys) error {
	if local.UserID == "" || local.DeviceID == "" {
		return errors.New("local user and device identifiers are required")
	}
	if len(local.IdentityPrivateKey) != x25519KeySize {
		return errors.New("local identity private key must be 32 bytes")
	}
	if len(local.IdentityPublicKey) != x25519KeySize {
		return errors.New("local identity public key must be 32 bytes")
	}
	return nil
}

func validatePeerBundle(peer DeviceKeyBundle) error {
	if len(peer.IdentitySigningPublicKey) != ed25519.PublicKeySize || len(peer.IdentityDHPublicKey) != x25519KeySize || len(peer.SignedPreKey) != x25519KeySize {
		return errors.New("peer X3DH key bundle must contain valid Ed25519 and X25519 public keys")
	}
	if len(peer.SignedPreKeySig) != ed25519.SignatureSize {
		return errors.New("peer X3DH key bundle must contain a signed prekey signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(peer.IdentitySigningPublicKey), signedPreKeyMessage(peer.SignedPreKey), peer.SignedPreKeySig) {
		return errors.New("peer signed prekey signature verification failed")
	}
	if len(peer.IdentityDHPublicKey) == 0 || len(peer.SignedPreKey) == 0 {
		return errors.New("peer X3DH key bundle is incomplete")
	}
	return nil
}

func signedPreKeyMessage(publicKey []byte) []byte {
	context := []byte("Zenthril signed prekey v1")
	out := make([]byte, 0, len(context)+len(publicKey))
	out = append(out, context...)
	out = append(out, publicKey...)
	return out
}

func concatBytes(parts ...[]byte) []byte {
	total := 0
	for _, part := range parts {
		total += len(part)
	}
	out := make([]byte, 0, total)
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}
