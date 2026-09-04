package crypto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/curve25519"
)

// Test vectors for X25519 DH operations
// These ensure our X25519 implementation is compatible with standard implementations
func TestX25519TestVectors(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		publicKey  string
		expected   string
	}{
		{
			name:       "RFC 7748 Test Vector 1",
			privateKey: "77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a",
			publicKey:  "de9edb7d7b7dc1b4d35b61c2ece435373f8343c85b78674dadfc7e146f882b4f",
			expected:   "4a5d9d5ba4ce2de1728e3bf480350f25e07e21c947d19e3376f09b3c1e161742",
		},
		{
			name:       "RFC 7748 Test Vector 2",
			privateKey: "5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb",
			publicKey:  "8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a",
			expected:   "4a5d9d5ba4ce2de1728e3bf480350f25e07e21c947d19e3376f09b3c1e161742",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, err := hex.DecodeString(tt.privateKey)
			if err != nil {
				t.Fatalf("decode private key: %v", err)
			}
			publicKey, err := hex.DecodeString(tt.publicKey)
			if err != nil {
				t.Fatalf("decode public key: %v", err)
			}
			expected, err := hex.DecodeString(tt.expected)
			if err != nil {
				t.Fatalf("decode expected: %v", err)
			}

			result, err := x25519SharedSecret(privateKey, publicKey)
			if err != nil {
				t.Fatalf("X25519: %v", err)
			}

			if !bytes.Equal(result, expected) {
				t.Errorf("X25519 mismatch\ngot:  %x\nwant: %x", result, expected)
			}
		})
	}
}

// Test vector for X25519 base point multiplication
func TestX25519BasePointMultiplication(t *testing.T) {
	// Private scalar
	privateKeyHex := "77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a"
	expectedPublicHex := "8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a"

	privateKey, _ := hex.DecodeString(privateKeyHex)
	expectedPublic, _ := hex.DecodeString(expectedPublicHex)

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("X25519 basepoint multiplication: %v", err)
	}

	if !bytes.Equal(publicKey, expectedPublic) {
		t.Errorf("public key mismatch\ngot:  %x\nwant: %x", publicKey, expectedPublic)
	}
}

// Test vectors for Ed25519 signature verification
func TestEd25519SignatureVectors(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		message   string
		signature string
		valid     bool
	}{
		{
			name:      "RFC 8032 Test Vector 1",
			publicKey: "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a",
			message:   "",
			signature: "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b",
			valid:     true,
		},
		{
			name:      "RFC 8032 Test Vector 2",
			publicKey: "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
			message:   "72",
			signature: "92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00",
			valid:     true,
		},
		{
			name:      "Invalid signature",
			publicKey: "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a",
			message:   "",
			signature: "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publicKey, _ := hex.DecodeString(tt.publicKey)
			message, _ := hex.DecodeString(tt.message)
			signature, _ := hex.DecodeString(tt.signature)

			valid := ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)

			if valid != tt.valid {
				t.Errorf("signature validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// Test vector for HKDF derivation used in ratchet
func TestHKDFTestVectors(t *testing.T) {
	// Test vector from RFC 5869
	ikm := mustDecodeHex("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
	salt := mustDecodeHex("000102030405060708090a0b0c")
	info := mustDecodeHex("f0f1f2f3f4f5f6f7f8f9")
	expectedOKM := mustDecodeHex("3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865")

	okm, err := hkdfBytes(ikm, salt, info, len(expectedOKM))
	if err != nil {
		t.Fatalf("HKDF: %v", err)
	}

	if !bytes.Equal(okm, expectedOKM) {
		t.Errorf("HKDF output mismatch\ngot:  %x\nwant: %x", okm, expectedOKM)
	}
}

// Test vector for symmetric ratchet chain derivation
func TestSymmetricRatchetDeterminism(t *testing.T) {
	// Ensure symmetric ratchet produces deterministic results
	sharedSecret := bytes.Repeat([]byte{0x42}, 32)
	info := []byte("test-session")

	state1, err := DeriveInitialRatchetState(sharedSecret, info, true)
	if err != nil {
		t.Fatalf("derive state 1: %v", err)
	}

	state2, err := DeriveInitialRatchetState(sharedSecret, info, true)
	if err != nil {
		t.Fatalf("derive state 2: %v", err)
	}

	// Same inputs should produce same initial state
	if !bytes.Equal(state1.RootKey, state2.RootKey) {
		t.Error("root keys don't match")
	}
	if !bytes.Equal(state1.SendChainKey, state2.SendChainKey) {
		t.Error("send chain keys don't match")
	}
	if !bytes.Equal(state1.RecvChainKey, state2.RecvChainKey) {
		t.Error("recv chain keys don't match")
	}

	// Advancing should produce same sequence
	key1a, _ := NextSendMessageKey(&state1)
	key1b, _ := NextSendMessageKey(&state1)

	state3, _ := DeriveInitialRatchetState(sharedSecret, info, true)
	key2a, _ := NextSendMessageKey(&state3)
	key2b, _ := NextSendMessageKey(&state3)

	if !bytes.Equal(key1a.Key, key2a.Key) {
		t.Error("first message keys don't match")
	}
	if !bytes.Equal(key1b.Key, key2b.Key) {
		t.Error("second message keys don't match")
	}
}

// Test vector for message encryption/decryption consistency
func TestEncryptDecryptTestVector(t *testing.T) {
	// Known plaintext and state
	plaintext := []byte("The quick brown fox jumps over the lazy dog")
	aad := []byte("channel:123:user:alice")

	sharedSecret := mustDecodeHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	info := []byte("alice:device1->bob:device2")

	aliceState, _ := DeriveInitialRatchetState(sharedSecret, info, true)
	bobState, _ := DeriveInitialRatchetState(sharedSecret, info, false)

	// Add DH keys
	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	// Encrypt
	encrypted, err := EncryptMessage(&aliceState, plaintext, aad)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Verify message structure
	if len(encrypted.Header.DHPublicKey) != 32 {
		t.Errorf("DH public key length = %d, want 32", len(encrypted.Header.DHPublicKey))
	}
	if encrypted.Header.MessageCounter != 0 {
		t.Errorf("message counter = %d, want 0", encrypted.Header.MessageCounter)
	}

	// Decrypt
	decrypted, err := DecryptMessage(&bobState, encrypted, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("plaintext mismatch\ngot:  %s\nwant: %s", decrypted, plaintext)
	}
}

// Test vector for X3DH shared secret computation
func TestX3DHSharedSecretVector(t *testing.T) {
	// Create deterministic keys for reproducible test
	aliceIdentityPriv := mustDecodeHex("70076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	aliceEphemeralPriv := mustDecodeHex("5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb")
	
	bobIdentityPub := mustDecodeHex("de9edb7d7b7dc1b4d35b61c2ece435373f8343c85b78674dadfc7e146f882b4f")
	bobSignedPreKeyPub := mustDecodeHex("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")

	bundle := DeviceKeyBundle{
		IdentityDHPublicKey: bobIdentityPub,
		SignedPreKey:        bobSignedPreKeyPub,
	}

	// Compute X3DH shared secret (without one-time prekey)
	sharedSecret1, err := deriveX3DHSharedSecret(aliceIdentityPriv, aliceEphemeralPriv, bundle, nil)
	if err != nil {
		t.Fatalf("derive shared secret 1: %v", err)
	}

	// Compute again with same inputs
	sharedSecret2, err := deriveX3DHSharedSecret(aliceIdentityPriv, aliceEphemeralPriv, bundle, nil)
	if err != nil {
		t.Fatalf("derive shared secret 2: %v", err)
	}

	// Should produce identical output
	if !bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("X3DH shared secrets don't match for same inputs")
	}

	// Should produce 32-byte output
	if len(sharedSecret1) != 32 {
		t.Errorf("shared secret length = %d, want 32", len(sharedSecret1))
	}

	// Output should be deterministic - store for regression testing
	t.Logf("X3DH shared secret (no OPK): %x", sharedSecret1)
}

// Test vector for complete X3DH with one-time prekey
func TestX3DHWithOneTimePreKeyVector(t *testing.T) {
	aliceIdentityPriv := mustDecodeHex("70076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	aliceEphemeralPriv := mustDecodeHex("5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb")
	
	bobIdentityPub := mustDecodeHex("de9edb7d7b7dc1b4d35b61c2ece435373f8343c85b78674dadfc7e146f882b4f")
	bobSignedPreKeyPub := mustDecodeHex("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")
	bobOneTimePreKeyPub := mustDecodeHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	bundle := DeviceKeyBundle{
		IdentityDHPublicKey: bobIdentityPub,
		SignedPreKey:        bobSignedPreKeyPub,
	}

	// Without one-time prekey
	withoutOPK, err := deriveX3DHSharedSecret(aliceIdentityPriv, aliceEphemeralPriv, bundle, nil)
	if err != nil {
		t.Fatalf("derive without OPK: %v", err)
	}

	// With one-time prekey
	withOPK, err := deriveX3DHSharedSecret(aliceIdentityPriv, aliceEphemeralPriv, bundle, bobOneTimePreKeyPub)
	if err != nil {
		t.Fatalf("derive with OPK: %v", err)
	}

	// Should produce different outputs
	if bytes.Equal(withoutOPK, withOPK) {
		t.Error("X3DH should produce different outputs with/without one-time prekey")
	}

	t.Logf("X3DH without OPK: %x", withoutOPK)
	t.Logf("X3DH with OPK:    %x", withOPK)
}

// Test vector for root ratchet advancement
func TestRootRatchetVector(t *testing.T) {
	rootKey := mustDecodeHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	dhOutput := mustDecodeHex("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")

	output1, err := RootRatchet(rootKey, dhOutput)
	if err != nil {
		t.Fatalf("root ratchet 1: %v", err)
	}

	output2, err := RootRatchet(rootKey, dhOutput)
	if err != nil {
		t.Fatalf("root ratchet 2: %v", err)
	}

	// Should be deterministic
	if !bytes.Equal(output1.RootKey, output2.RootKey) {
		t.Error("root keys don't match")
	}
	if !bytes.Equal(output1.ChainKey, output2.ChainKey) {
		t.Error("chain keys don't match")
	}

	// Should produce 32-byte keys
	if len(output1.RootKey) != 32 {
		t.Errorf("root key length = %d, want 32", len(output1.RootKey))
	}
	if len(output1.ChainKey) != 32 {
		t.Errorf("chain key length = %d, want 32", len(output1.ChainKey))
	}

	// Should be different from input
	if bytes.Equal(output1.RootKey, rootKey) {
		t.Error("root key should change after ratchet")
	}

	t.Logf("Root ratchet output:\n  new root: %x\n  chain:    %x", output1.RootKey, output1.ChainKey)
}

// Test vector for message header serialization
func TestMessageHeaderSerializationVector(t *testing.T) {
	dhKey := mustDecodeHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	
	header := MessageHeader{
		DHPublicKey:     dhKey,
		PreviousCounter: 42,
		MessageCounter:  123,
	}

	serialized := SerializeMessageHeader(header)
	
	// Should be 40 bytes (32 + 4 + 4)
	if len(serialized) != 40 {
		t.Errorf("serialized length = %d, want 40", len(serialized))
	}

	deserialized, err := DeserializeMessageHeader(serialized)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if !bytes.Equal(header.DHPublicKey, deserialized.DHPublicKey) {
		t.Error("DH public key mismatch")
	}
	if header.PreviousCounter != deserialized.PreviousCounter {
		t.Errorf("previous counter = %d, want %d", deserialized.PreviousCounter, header.PreviousCounter)
	}
	if header.MessageCounter != deserialized.MessageCounter {
		t.Errorf("message counter = %d, want %d", deserialized.MessageCounter, header.MessageCounter)
	}

	t.Logf("Serialized header (%d bytes): %x", len(serialized), serialized)
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
