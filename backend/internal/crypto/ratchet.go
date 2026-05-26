package crypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	ratchetKeySize = 32
)

var (
	ErrInvalidRatchetState = errors.New("invalid_ratchet_state")
)

type MessageKey struct {
	Key     []byte
	Nonce   []byte
	Counter uint32
}

type RatchetState struct {
	RootKey      []byte
	SendChainKey []byte
	RecvChainKey []byte
	SendCounter  uint32
	RecvCounter  uint32
}

type RootRatchetOutput struct {
	RootKey  []byte
	ChainKey []byte
}

func NewRatchetState(rootKey, sendChainKey, recvChainKey []byte) (RatchetState, error) {
	if len(rootKey) != ratchetKeySize || len(sendChainKey) != ratchetKeySize || len(recvChainKey) != ratchetKeySize {
		return RatchetState{}, fmt.Errorf("%w: root/send/recv keys must be 32 bytes", ErrInvalidRatchetState)
	}
	return RatchetState{
		RootKey:      cloneBytes(rootKey),
		SendChainKey: cloneBytes(sendChainKey),
		RecvChainKey: cloneBytes(recvChainKey),
	}, nil
}

func DeriveInitialRatchetState(sharedSecret []byte, sessionInfo []byte, initiator bool) (RatchetState, error) {
	if len(sharedSecret) == 0 {
		return RatchetState{}, fmt.Errorf("%w: shared secret is required", ErrInvalidRatchetState)
	}

	material, err := hkdfBytes(sharedSecret, nil, append([]byte("zenthril-ratchet-v1:init:"), sessionInfo...), ratchetKeySize*3)
	if err != nil {
		return RatchetState{}, err
	}
	root := material[:ratchetKeySize]
	first := material[ratchetKeySize : ratchetKeySize*2]
	second := material[ratchetKeySize*2:]

	if initiator {
		return NewRatchetState(root, first, second)
	}
	return NewRatchetState(root, second, first)
}

func RootRatchet(rootKey, dhOutput []byte) (RootRatchetOutput, error) {
	if len(rootKey) != ratchetKeySize || len(dhOutput) == 0 {
		return RootRatchetOutput{}, fmt.Errorf("%w: root key and DH output are required", ErrInvalidRatchetState)
	}

	material, err := hkdfBytes(dhOutput, rootKey, []byte("zenthril-ratchet-v1:root"), ratchetKeySize*2)
	if err != nil {
		return RootRatchetOutput{}, err
	}
	return RootRatchetOutput{
		RootKey:  material[:ratchetKeySize],
		ChainKey: material[ratchetKeySize:],
	}, nil
}

func NextSendMessageKey(state *RatchetState) (MessageKey, error) {
	if state == nil || len(state.SendChainKey) != ratchetKeySize {
		return MessageKey{}, fmt.Errorf("%w: send chain key is required", ErrInvalidRatchetState)
	}
	msg, next, err := deriveMessageAndNextChain(state.SendChainKey, state.SendCounter)
	if err != nil {
		return MessageKey{}, err
	}
	state.SendChainKey = next
	state.SendCounter++
	return msg, nil
}

func NextRecvMessageKey(state *RatchetState) (MessageKey, error) {
	if state == nil || len(state.RecvChainKey) != ratchetKeySize {
		return MessageKey{}, fmt.Errorf("%w: receive chain key is required", ErrInvalidRatchetState)
	}
	msg, next, err := deriveMessageAndNextChain(state.RecvChainKey, state.RecvCounter)
	if err != nil {
		return MessageKey{}, err
	}
	state.RecvChainKey = next
	state.RecvCounter++
	return msg, nil
}

func (s *SessionState) RatchetState() (RatchetState, error) {
	return NewRatchetState(s.RootKey, s.SendChainKey, s.RecvChainKey)
}

func (s *SessionState) ApplyRatchetState(state RatchetState) {
	// SECURITY: copy key material so caller-owned slices cannot mutate session state.
	s.RootKey = cloneBytes(state.RootKey)
	s.SendChainKey = cloneBytes(state.SendChainKey)
	s.RecvChainKey = cloneBytes(state.RecvChainKey)
	s.SendCounter = state.SendCounter
	s.RecvCounter = state.RecvCounter
}

func deriveMessageAndNextChain(chainKey []byte, counter uint32) (MessageKey, []byte, error) {
	material, err := hkdfBytes(chainKey, nil, []byte("zenthril-ratchet-v1:chain"), ratchetKeySize+12+ratchetKeySize)
	if err != nil {
		return MessageKey{}, nil, err
	}
	return MessageKey{
		Key:     material[:ratchetKeySize],
		Nonce:   material[ratchetKeySize : ratchetKeySize+12],
		Counter: counter,
	}, material[ratchetKeySize+12:], nil
}

func hkdfBytes(secret, salt, info []byte, size int) ([]byte, error) {
	out := make([]byte, size)
	reader := hkdf.New(sha256.New, secret, salt, info)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, fmt.Errorf("derive ratchet key material: %w", err)
	}
	return out, nil
}

func cloneBytes(in []byte) []byte {
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
