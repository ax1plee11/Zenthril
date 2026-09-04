package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Setup two peers with matching initial state
	shared := bytes.Repeat([]byte{0x42}, 32)
	info := []byte("alice:device1->bob:device2")

	aliceState, err := DeriveInitialRatchetState(shared, info, true)
	if err != nil {
		t.Fatalf("alice state: %v", err)
	}
	bobState, err := DeriveInitialRatchetState(shared, info, false)
	if err != nil {
		t.Fatalf("bob state: %v", err)
	}

	// Add DH keys
	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	plaintext := []byte("Hello, secure world!")
	aad := []byte("context:channel-123")

	// Alice encrypts
	encrypted, err := EncryptMessage(&aliceState, plaintext, aad)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Bob decrypts
	decrypted, err := DecryptMessage(&bobState, encrypted, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptRejectsWrongAAD(t *testing.T) {
	shared := bytes.Repeat([]byte{0x55}, 32)
	info := []byte("test-session")

	aliceState, _ := DeriveInitialRatchetState(shared, info, true)
	bobState, _ := DeriveInitialRatchetState(shared, info, false)

	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	plaintext := []byte("secret")
	encrypted, _ := EncryptMessage(&aliceState, plaintext, []byte("original-aad"))

	_, err := DecryptMessage(&bobState, encrypted, []byte("wrong-aad"))
	if err == nil {
		t.Fatal("expected decryption to fail with wrong AAD")
	}
}

func TestDHRatchetTurnOnNewPublicKey(t *testing.T) {
	// Simplified test: two peers start with compatible symmetric ratchet state
	// and perform DH ratchet turns as they exchange messages
	
	shared := bytes.Repeat([]byte{0x33}, 32)
	info := []byte("ratchet-turn-test")

	aliceState, _ := DeriveInitialRatchetState(shared, info, true)
	bobState, _ := DeriveInitialRatchetState(shared, info, false)

	// Both start with DH keys and know each other's keys (as if after X3DH)
	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	// Test basic encryption/decryption works
	msg1, err := EncryptMessage(&aliceState, []byte("message 1"), nil)
	if err != nil {
		t.Fatalf("Alice encrypt: %v", err)
	}

	plaintext1, err := DecryptMessage(&bobState, msg1, nil)
	if err != nil {
		t.Fatalf("Bob decrypt: %v", err)
	}
	if string(plaintext1) != "message 1" {
		t.Fatalf("wrong plaintext: %q", plaintext1)
	}

	// Bob replies - Alice should receive successfully
	msg2, err := EncryptMessage(&bobState, []byte("message 2"), nil)
	if err != nil {
		t.Fatalf("Bob encrypt: %v", err)
	}

	plaintext2, err := DecryptMessage(&aliceState, msg2, nil)
	if err != nil {
		t.Fatalf("Alice decrypt: %v", err)
	}
	if string(plaintext2) != "message 2" {
		t.Fatalf("wrong plaintext: %q", plaintext2)
	}

	// Verify DH keys are still valid
	if len(aliceState.DHSendPublic) != x25519KeySize {
		t.Fatal("Alice should have valid DH send public key")
	}
	if len(bobState.DHSendPublic) != x25519KeySize {
		t.Fatal("Bob should have valid DH send public key")
	}
}

func TestOutOfOrderMessageDelivery(t *testing.T) {
	shared := bytes.Repeat([]byte{0x77}, 32)
	info := []byte("out-of-order-test")

	aliceState, _ := DeriveInitialRatchetState(shared, info, true)
	bobState, _ := DeriveInitialRatchetState(shared, info, false)

	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	// Alice sends 3 messages
	msg1, _ := EncryptMessage(&aliceState, []byte("first"), nil)
	msg2, _ := EncryptMessage(&aliceState, []byte("second"), nil)
	msg3, _ := EncryptMessage(&aliceState, []byte("third"), nil)

	// Bob receives them out of order: 3, 1, 2
	decrypted3, err := DecryptMessage(&bobState, msg3, nil)
	if err != nil {
		t.Fatalf("decrypt message 3: %v", err)
	}
	if !bytes.Equal(decrypted3, []byte("third")) {
		t.Fatalf("wrong plaintext for message 3: %q", decrypted3)
	}

	decrypted1, err := DecryptMessage(&bobState, msg1, nil)
	if err != nil {
		t.Fatalf("decrypt message 1: %v", err)
	}
	if !bytes.Equal(decrypted1, []byte("first")) {
		t.Fatalf("wrong plaintext for message 1: %q", decrypted1)
	}

	decrypted2, err := DecryptMessage(&bobState, msg2, nil)
	if err != nil {
		t.Fatalf("decrypt message 2: %v", err)
	}
	if !bytes.Equal(decrypted2, []byte("second")) {
		t.Fatalf("wrong plaintext for message 2: %q", decrypted2)
	}
}

func TestMessageHeaderSerialization(t *testing.T) {
	_, dhPub, _ := generateEphemeralKeyPair()
	
	original := MessageHeader{
		DHPublicKey:     dhPub,
		PreviousCounter: 42,
		MessageCounter:  123,
	}

	serialized := SerializeMessageHeader(original)
	deserialized, err := DeserializeMessageHeader(serialized)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if !bytes.Equal(original.DHPublicKey, deserialized.DHPublicKey) {
		t.Fatal("DH public key mismatch")
	}
	if original.PreviousCounter != deserialized.PreviousCounter {
		t.Fatalf("previous counter mismatch: got %d, want %d", deserialized.PreviousCounter, original.PreviousCounter)
	}
	if original.MessageCounter != deserialized.MessageCounter {
		t.Fatalf("message counter mismatch: got %d, want %d", deserialized.MessageCounter, original.MessageCounter)
	}
}

func TestEncryptRequiresDHPublicKey(t *testing.T) {
	state, _ := DeriveInitialRatchetState(bytes.Repeat([]byte{0x11}, 32), []byte("test"), true)
	
	// State without DH keys should fail
	_, err := EncryptMessage(&state, []byte("test"), nil)
	if err == nil {
		t.Fatal("expected encryption to fail without DH public key")
	}
}

func TestReplayProtection(t *testing.T) {
	shared := bytes.Repeat([]byte{0x99}, 32)
	info := []byte("replay-test")

	aliceState, _ := DeriveInitialRatchetState(shared, info, true)
	bobState, _ := DeriveInitialRatchetState(shared, info, false)

	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	// Alice sends a message
	msg, _ := EncryptMessage(&aliceState, []byte("original"), nil)

	// Bob decrypts it successfully
	_, err := DecryptMessage(&bobState, msg, nil)
	if err != nil {
		t.Fatalf("first decrypt: %v", err)
	}

	// Attempt to decrypt the same message again (replay attack)
	_, err = DecryptMessage(&bobState, msg, nil)
	if err == nil {
		t.Fatal("replayed message should be rejected")
	}
}

func TestDHRatchetTurnPreservesSkippedKeys(t *testing.T) {
	shared := bytes.Repeat([]byte{0xAA}, 32)
	info := []byte("skipped-keys-ratchet-test")

	aliceState, _ := DeriveInitialRatchetState(shared, info, true)
	bobState, _ := DeriveInitialRatchetState(shared, info, false)

	aliceDHPriv, aliceDHPub, _ := generateEphemeralKeyPair()
	bobDHPriv, bobDHPub, _ := generateEphemeralKeyPair()

	aliceState.DHSendPrivate = aliceDHPriv
	aliceState.DHSendPublic = aliceDHPub
	aliceState.DHRecvPublic = bobDHPub

	bobState.DHSendPrivate = bobDHPriv
	bobState.DHSendPublic = bobDHPub
	bobState.DHRecvPublic = aliceDHPub

	// Alice sends 2 messages
	msg1, _ := EncryptMessage(&aliceState, []byte("first"), nil)
	msg2, _ := EncryptMessage(&aliceState, []byte("second"), nil)

	// Bob receives only the second message (skips first)
	_, _ = DecryptMessage(&bobState, msg2, nil)

	// Verify skipped key was stored
	if len(bobState.SkippedMessageKeys) != 1 {
		t.Fatalf("expected 1 skipped key, got %d", len(bobState.SkippedMessageKeys))
	}

	// Now Bob sends a message (triggers DH ratchet turn on Alice's side)
	msg3, _ := EncryptMessage(&bobState, []byte("bob's message"), nil)
	_, _ = DecryptMessage(&aliceState, msg3, nil)

	// Bob should still be able to decrypt the skipped message
	decrypted1, err := DecryptMessage(&bobState, msg1, nil)
	if err != nil {
		t.Fatalf("decrypt skipped message after ratchet turn: %v", err)
	}
	if !bytes.Equal(decrypted1, []byte("first")) {
		t.Fatalf("wrong plaintext for skipped message: %q", decrypted1)
	}
}
