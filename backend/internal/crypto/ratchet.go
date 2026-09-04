package crypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	ratchetKeySize           = 32
	maxSkippedMessageKeys    = 2000
	maxRatchetMessageCounter = ^uint32(0)
)

var (
	ErrInvalidRatchetState     = errors.New("invalid_ratchet_state")
	ErrSkippedMessageLimit     = errors.New("skipped_message_limit_exceeded")
	ErrMessageKeyUnavailable   = errors.New("message_key_unavailable")
	ErrRatchetCounterExhausted = errors.New("ratchet_counter_exhausted")
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
	
	// DH Ratchet keys for asymmetric ratchet turns
	DHSendPrivate     []byte // Our current DH private key
	DHSendPublic      []byte // Our current DH public key
	DHRecvPublic      []byte // Peer's current DH public key
	PreviousCounter   uint32 // Messages sent in previous sending chain
	
	// SECURITY: skipped message keys are retained only for bounded, out-of-order
	// delivery. Each entry is deleted and zeroed immediately when consumed.
	SkippedMessageKeys map[uint32]MessageKey
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
		RootKey:            cloneBytes(rootKey),
		SendChainKey:       cloneBytes(sendChainKey),
		RecvChainKey:       cloneBytes(recvChainKey),
		SkippedMessageKeys: make(map[uint32]MessageKey),
	}, nil
}

// NewRatchetStateWithDH creates a ratchet state with DH keys for full Double Ratchet
func NewRatchetStateWithDH(rootKey, sendChainKey, recvChainKey []byte, dhSendPriv, dhSendPub, dhRecvPub []byte) (RatchetState, error) {
	if len(rootKey) != ratchetKeySize || len(sendChainKey) != ratchetKeySize || len(recvChainKey) != ratchetKeySize {
		return RatchetState{}, fmt.Errorf("%w: root/send/recv keys must be 32 bytes", ErrInvalidRatchetState)
	}
	if len(dhSendPriv) != x25519KeySize || len(dhSendPub) != x25519KeySize {
		return RatchetState{}, fmt.Errorf("%w: DH send keys must be 32 bytes", ErrInvalidRatchetState)
	}
	if len(dhRecvPub) > 0 && len(dhRecvPub) != x25519KeySize {
		return RatchetState{}, fmt.Errorf("%w: DH recv key must be 32 bytes or empty", ErrInvalidRatchetState)
	}
	
	return RatchetState{
		RootKey:            cloneBytes(rootKey),
		SendChainKey:       cloneBytes(sendChainKey),
		RecvChainKey:       cloneBytes(recvChainKey),
		DHSendPrivate:      cloneBytes(dhSendPriv),
		DHSendPublic:       cloneBytes(dhSendPub),
		DHRecvPublic:       cloneBytes(dhRecvPub),
		SkippedMessageKeys: make(map[uint32]MessageKey),
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

// DHRatchetTurn performs a DH ratchet turn when receiving a new peer public key.
// This provides forward secrecy and post-compromise security.
//
// SECURITY: This must be called when receiving a message with a new DH public key
// that differs from DHRecvPublic. The receiver:
// 1. Performs DH with peer's new public key
// 2. Advances root key
// 3. Initializes new receive chain
// 4. Generates new DH key pair
// 5. Performs DH with own new private key
// 6. Advances root key again
// 7. Initializes new send chain
func DHRatchetTurn(state *RatchetState, newPeerDHPublic []byte) error {
	if state == nil {
		return fmt.Errorf("%w: state is required", ErrInvalidRatchetState)
	}
	if len(newPeerDHPublic) != x25519KeySize {
		return fmt.Errorf("%w: peer DH public key must be 32 bytes", ErrInvalidRatchetState)
	}
	
	// Step 1: Perform DH with new peer public key using our current DH private key
	dhOutput, err := x25519SharedSecret(state.DHSendPrivate, newPeerDHPublic)
	if err != nil {
		return fmt.Errorf("DH with new peer key: %w", err)
	}
	
	// Step 2: Advance root key and get new receive chain key
	rootOutput, err := RootRatchet(state.RootKey, dhOutput)
	if err != nil {
		return fmt.Errorf("root ratchet for receive: %w", err)
	}
	
	state.RootKey = rootOutput.RootKey
	state.RecvChainKey = rootOutput.ChainKey
	state.PreviousCounter = state.SendCounter
	state.RecvCounter = 0
	state.DHRecvPublic = cloneBytes(newPeerDHPublic)
	
	// Step 3: Generate new DH key pair for sending
	newDHPrivate, newDHPublic, err := generateEphemeralKeyPair()
	if err != nil {
		return fmt.Errorf("generate new DH pair: %w", err)
	}
	
	// Zero out old DH private key before replacing
	zeroBytes(state.DHSendPrivate)
	state.DHSendPrivate = newDHPrivate
	state.DHSendPublic = newDHPublic
	
	// Step 4: Perform DH with peer's public key using new private key
	dhOutput2, err := x25519SharedSecret(state.DHSendPrivate, newPeerDHPublic)
	if err != nil {
		return fmt.Errorf("DH with new send key: %w", err)
	}
	
	// Step 5: Advance root key again and get new send chain key
	rootOutput2, err := RootRatchet(state.RootKey, dhOutput2)
	if err != nil {
		return fmt.Errorf("root ratchet for send: %w", err)
	}
	
	state.RootKey = rootOutput2.RootKey
	state.SendChainKey = rootOutput2.ChainKey
	state.SendCounter = 0
	
	return nil
}

func NextSendMessageKey(state *RatchetState) (MessageKey, error) {
	if state == nil || len(state.SendChainKey) != ratchetKeySize {
		return MessageKey{}, fmt.Errorf("%w: send chain key is required", ErrInvalidRatchetState)
	}
	if state.SendCounter == maxRatchetMessageCounter {
		return MessageKey{}, ErrRatchetCounterExhausted
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
	if state == nil {
		return MessageKey{}, fmt.Errorf("%w: receive chain key is required", ErrInvalidRatchetState)
	}
	return NextRecvMessageKeyFor(state, state.RecvCounter)
}

// NextRecvMessageKeyFor derives a receive key for a message counter while
// safely supporting bounded out-of-order delivery.
//
// SECURITY: a retained skipped key is consumed exactly once. Requests below
// RecvCounter therefore fail after the matching key has been used, preventing
// replayed ciphertext from being accepted through the ratchet layer.
func NextRecvMessageKeyFor(state *RatchetState, counter uint32) (MessageKey, error) {
	if state == nil || len(state.RecvChainKey) != ratchetKeySize {
		return MessageKey{}, fmt.Errorf("%w: receive chain key is required", ErrInvalidRatchetState)
	}
	if state.SkippedMessageKeys == nil {
		state.SkippedMessageKeys = make(map[uint32]MessageKey)
	}

	if counter < state.RecvCounter {
		return consumeSkippedMessageKey(state, counter)
	}

	gap := uint64(counter) - uint64(state.RecvCounter)
	if gap > maxSkippedMessageKeys || len(state.SkippedMessageKeys)+int(gap) > maxSkippedMessageKeys {
		return MessageKey{}, ErrSkippedMessageLimit
	}

	for state.RecvCounter < counter {
		skipped, err := deriveNextReceiveMessageKey(state)
		if err != nil {
			return MessageKey{}, err
		}
		state.SkippedMessageKeys[skipped.Counter] = cloneMessageKey(skipped)
		zeroMessageKey(&skipped)
	}

	return deriveNextReceiveMessageKey(state)
}

func deriveNextReceiveMessageKey(state *RatchetState) (MessageKey, error) {
	if state.RecvCounter == maxRatchetMessageCounter {
		return MessageKey{}, ErrRatchetCounterExhausted
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
	state, err := NewRatchetState(s.RootKey, s.SendChainKey, s.RecvChainKey)
	if err != nil {
		return RatchetState{}, err
	}
	state.SendCounter = s.SendCounter
	state.RecvCounter = s.RecvCounter
	state.DHSendPrivate = cloneBytes(s.DHSendPrivate)
	state.DHSendPublic = cloneBytes(s.DHSendPublic)
	state.DHRecvPublic = cloneBytes(s.DHRecvPublic)
	state.PreviousCounter = s.PreviousCounter
	state.SkippedMessageKeys = cloneSkippedMessageKeys(s.SkippedMessageKeys)
	return state, nil
}

func (s *SessionState) ApplyRatchetState(state RatchetState) {
	// SECURITY: copy key material so caller-owned slices cannot mutate session state.
	s.RootKey = cloneBytes(state.RootKey)
	s.SendChainKey = cloneBytes(state.SendChainKey)
	s.RecvChainKey = cloneBytes(state.RecvChainKey)
	s.SendCounter = state.SendCounter
	s.RecvCounter = state.RecvCounter
	s.DHSendPrivate = cloneBytes(state.DHSendPrivate)
	s.DHSendPublic = cloneBytes(state.DHSendPublic)
	s.DHRecvPublic = cloneBytes(state.DHRecvPublic)
	s.PreviousCounter = state.PreviousCounter
	s.SkippedMessageKeys = cloneSkippedMessageKeys(state.SkippedMessageKeys)
}

func deriveMessageAndNextChain(chainKey []byte, counter uint32) (MessageKey, []byte, error) {
	material, err := hkdfBytes(chainKey, nil, []byte("zenthril-ratchet-v1:chain"), ratchetKeySize+12+ratchetKeySize)
	if err != nil {
		return MessageKey{}, nil, err
	}
	message := MessageKey{
		Key:     cloneBytes(material[:ratchetKeySize]),
		Nonce:   cloneBytes(material[ratchetKeySize : ratchetKeySize+12]),
		Counter: counter,
	}
	nextChainKey := cloneBytes(material[ratchetKeySize+12:])
	zeroBytes(material)
	return message, nextChainKey, nil
}

func consumeSkippedMessageKey(state *RatchetState, counter uint32) (MessageKey, error) {
	stored, ok := state.SkippedMessageKeys[counter]
	if !ok {
		return MessageKey{}, ErrMessageKeyUnavailable
	}
	delete(state.SkippedMessageKeys, counter)
	message := cloneMessageKey(stored)
	zeroMessageKey(&stored)
	return message, nil
}

func cloneSkippedMessageKeys(in map[uint32]MessageKey) map[uint32]MessageKey {
	if len(in) == 0 {
		return make(map[uint32]MessageKey)
	}
	out := make(map[uint32]MessageKey, len(in))
	for counter, message := range in {
		out[counter] = cloneMessageKey(message)
	}
	return out
}

func cloneMessageKey(in MessageKey) MessageKey {
	return MessageKey{
		Key:     cloneBytes(in.Key),
		Nonce:   cloneBytes(in.Nonce),
		Counter: in.Counter,
	}
}

func zeroMessageKey(key *MessageKey) {
	if key == nil {
		return
	}
	zeroBytes(key.Key)
	zeroBytes(key.Nonce)
	key.Key = nil
	key.Nonce = nil
}

func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
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
