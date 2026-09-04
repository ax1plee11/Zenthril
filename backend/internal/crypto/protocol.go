package crypto

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
)

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

	// WEAKNESS FIXED: replace deterministic SHA256(public-key concat) with audited X25519 X3DH.
	sharedSecret, err := deriveX3DHSharedSecret(local.IdentityPrivateKey, ephemeralPrivate, peer, oneTimePreKeyPublic)
	if err != nil {
		return SessionState{}, fmt.Errorf("derive X3DH shared secret: %w", err)
	}

	sessionInfo := []byte(local.UserID + ":" + local.DeviceID + "->" + peer.UserID + ":" + peer.DeviceID)
	ratchet, err := DeriveInitialRatchetState(sharedSecret, sessionInfo, true)
	if err != nil {
		return SessionState{}, err
	}
	
	// Initialize DH ratchet keys for initiator
	// Per Signal Protocol: initiator performs initial DH ratchet step with peer's signed prekey
	dhSendPriv, dhSendPub, err := generateEphemeralKeyPair()
	if err != nil {
		return SessionState{}, fmt.Errorf("generate initial DH keys: %w", err)
	}
	
	// Perform initial DH with peer's signed prekey to establish first sending chain
	dhOutput, err := x25519SharedSecret(dhSendPriv, peer.SignedPreKey)
	if err != nil {
		return SessionState{}, fmt.Errorf("initial DH with peer signed prekey: %w", err)
	}
	
	// Perform root ratchet to get new send chain key
	rootOutput, err := RootRatchet(ratchet.RootKey, dhOutput)
	if err != nil {
		return SessionState{}, fmt.Errorf("initial root ratchet: %w", err)
	}
	
	// Update ratchet state with new keys
	ratchet.RootKey = rootOutput.RootKey
	ratchet.SendChainKey = rootOutput.ChainKey
	ratchet.DHSendPrivate = dhSendPriv
	ratchet.DHSendPublic = dhSendPub
	ratchet.DHRecvPublic = peer.SignedPreKey // Peer's signed prekey is their current DH public key
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
