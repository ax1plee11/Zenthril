package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidCiphertext = errors.New("invalid_ciphertext")
	ErrDecryptionFailed  = errors.New("decryption_failed")
)

// MessageHeader contains metadata needed for Double Ratchet message processing
type MessageHeader struct {
	DHPublicKey     []byte // Sender's current DH public key
	PreviousCounter uint32 // Number of messages in previous sending chain
	MessageCounter  uint32 // Message number in current sending chain
}

// RatchetMessage represents an encrypted message with Double Ratchet metadata
type RatchetMessage struct {
	Header     MessageHeader
	Ciphertext []byte
}

// EncryptMessage encrypts plaintext using the Double Ratchet algorithm
//
// SECURITY: Each message advances the sending chain key and includes the sender's
// current DH public key in the header. This enables the receiver to perform
// DH ratchet turns and maintain forward secrecy.
func EncryptMessage(state *RatchetState, plaintext []byte, aad []byte) (RatchetMessage, error) {
	if state == nil {
		return RatchetMessage{}, fmt.Errorf("%w: state is required", ErrInvalidRatchetState)
	}
	if len(state.DHSendPublic) != x25519KeySize {
		return RatchetMessage{}, fmt.Errorf("%w: DH send public key required", ErrInvalidRatchetState)
	}

	// Get next message key from symmetric ratchet
	msgKey, err := NextSendMessageKey(state)
	if err != nil {
		return RatchetMessage{}, fmt.Errorf("derive send message key: %w", err)
	}
	defer zeroMessageKey(&msgKey)

	// Encrypt with AES-GCM
	block, err := aes.NewCipher(msgKey.Key)
	if err != nil {
		return RatchetMessage{}, fmt.Errorf("create cipher: %w", err)
	}
	
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return RatchetMessage{}, fmt.Errorf("create AEAD: %w", err)
	}

	ciphertext := aead.Seal(nil, msgKey.Nonce, plaintext, aad)

	message := RatchetMessage{
		Header: MessageHeader{
			DHPublicKey:     cloneBytes(state.DHSendPublic),
			PreviousCounter: state.PreviousCounter,
			MessageCounter:  msgKey.Counter,
		},
		Ciphertext: ciphertext,
	}

	return message, nil
}

// DecryptMessage decrypts a Double Ratchet message
//
// SECURITY: If the message header contains a new DH public key, this triggers
// a DH ratchet turn before decryption. Skipped message keys are stored with
// strict limits to handle out-of-order delivery.
func DecryptMessage(state *RatchetState, message RatchetMessage, aad []byte) ([]byte, error) {
	if state == nil {
		return nil, fmt.Errorf("%w: state is required", ErrInvalidRatchetState)
	}

	// Check if we need to perform a DH ratchet turn
	if len(message.Header.DHPublicKey) == x25519KeySize {
		// Only perform ratchet turn if this is a new DH public key
		if len(state.DHRecvPublic) == 0 || !bytesEqual(state.DHRecvPublic, message.Header.DHPublicKey) {
			// Perform DH ratchet turn
			if err := DHRatchetTurn(state, message.Header.DHPublicKey); err != nil {
				return nil, fmt.Errorf("DH ratchet turn: %w", err)
			}
		}
	}

	// Store skipped message keys in current chain if message counter jumped
	if message.Header.MessageCounter > state.RecvCounter {
		gap := uint64(message.Header.MessageCounter) - uint64(state.RecvCounter)
		if gap > maxSkippedMessageKeys {
			return nil, ErrSkippedMessageLimit
		}
	}

	// Get message key for this counter
	msgKey, err := NextRecvMessageKeyFor(state, message.Header.MessageCounter)
	if err != nil {
		return nil, fmt.Errorf("derive receive message key: %w", err)
	}
	defer zeroMessageKey(&msgKey)

	// Decrypt with AES-GCM
	block, err := aes.NewCipher(msgKey.Key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AEAD: %w", err)
	}

	plaintext, err := aead.Open(nil, msgKey.Nonce, message.Ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// storeSkippedMessageKeys stores message keys for skipped messages in the current chain
func storeSkippedMessageKeys(state *RatchetState, fromCounter, toCounter uint32) error {
	if fromCounter >= toCounter {
		return nil
	}

	gap := uint64(toCounter) - uint64(fromCounter)
	if gap > maxSkippedMessageKeys || len(state.SkippedMessageKeys)+int(gap) > maxSkippedMessageKeys {
		return ErrSkippedMessageLimit
	}

	for state.RecvCounter < toCounter {
		skipped, err := deriveNextReceiveMessageKey(state)
		if err != nil {
			return err
		}
		state.SkippedMessageKeys[skipped.Counter] = cloneMessageKey(skipped)
		zeroMessageKey(&skipped)
	}

	return nil
}

// SerializeMessageHeader serializes a message header to bytes
func SerializeMessageHeader(header MessageHeader) []byte {
	buf := make([]byte, 0, x25519KeySize+8)
	buf = append(buf, header.DHPublicKey...)
	
	prevCounterBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(prevCounterBuf, header.PreviousCounter)
	buf = append(buf, prevCounterBuf...)
	
	msgCounterBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgCounterBuf, header.MessageCounter)
	buf = append(buf, msgCounterBuf...)
	
	return buf
}

// DeserializeMessageHeader deserializes a message header from bytes
func DeserializeMessageHeader(data []byte) (MessageHeader, error) {
	if len(data) < x25519KeySize+8 {
		return MessageHeader{}, fmt.Errorf("%w: header too short", ErrInvalidCiphertext)
	}

	header := MessageHeader{
		DHPublicKey:     cloneBytes(data[:x25519KeySize]),
		PreviousCounter: binary.BigEndian.Uint32(data[x25519KeySize : x25519KeySize+4]),
		MessageCounter:  binary.BigEndian.Uint32(data[x25519KeySize+4 : x25519KeySize+8]),
	}

	return header, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
