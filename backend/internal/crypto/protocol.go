package crypto

import (
	"context"
	"errors"
)

type DeviceKeyBundle struct {
	UserID            string
	DeviceID          string
	IdentityKey       []byte
	SignedPreKey      []byte
	SignedPreKeyID    uint32
	SignedPreKeySig   []byte
	OneTimePreKey     []byte
	OneTimePreKeyID   uint32
	PublishedRevision uint64
}

type SessionState struct {
	UserID       string
	PeerUserID   string
	DeviceID     string
	PeerDeviceID string
	RootKey      []byte
	SendChainKey []byte
	RecvChainKey []byte
	SendCounter  uint32
	RecvCounter  uint32
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

func (s *X3DHService) StartSession(ctx context.Context, local DeviceKeyBundle, peerUserID, peerDeviceID string) (SessionState, error) {
	if s.keys == nil {
		return SessionState{}, errors.New("key store is required")
	}
	peer, err := s.keys.GetBundle(ctx, peerUserID, peerDeviceID)
	if err != nil {
		return SessionState{}, err
	}
	if len(local.IdentityKey) == 0 || len(peer.IdentityKey) == 0 || len(peer.SignedPreKey) == 0 {
		return SessionState{}, errors.New("incomplete X3DH key bundle")
	}

	// The concrete DH/KDF steps intentionally live behind this boundary. The
	// next implementation phase wires audited primitives and test vectors here.
	return SessionState{
		UserID:       local.UserID,
		PeerUserID:   peer.UserID,
		DeviceID:     local.DeviceID,
		PeerDeviceID: peer.DeviceID,
	}, nil
}
