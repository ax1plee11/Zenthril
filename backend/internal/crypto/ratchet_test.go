package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func TestDeriveInitialRatchetStateMatchesBetweenPeers(t *testing.T) {
	shared := bytes.Repeat([]byte{0x42}, 32)
	info := []byte("alice-device:bob-device")

	alice, err := DeriveInitialRatchetState(shared, info, true)
	if err != nil {
		t.Fatalf("alice derive: %v", err)
	}
	bob, err := DeriveInitialRatchetState(shared, info, false)
	if err != nil {
		t.Fatalf("bob derive: %v", err)
	}

	if !bytes.Equal(alice.RootKey, bob.RootKey) {
		t.Fatal("root keys must match")
	}
	if !bytes.Equal(alice.SendChainKey, bob.RecvChainKey) {
		t.Fatal("initiator send chain must equal responder receive chain")
	}
	if !bytes.Equal(alice.RecvChainKey, bob.SendChainKey) {
		t.Fatal("initiator receive chain must equal responder send chain")
	}
}

func TestNextMessageKeyAdvancesCountersAndChains(t *testing.T) {
	state, err := DeriveInitialRatchetState(bytes.Repeat([]byte{0x11}, 32), []byte("session"), true)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	originalChain := cloneBytes(state.SendChainKey)

	first, err := NextSendMessageKey(&state)
	if err != nil {
		t.Fatalf("first message key: %v", err)
	}
	second, err := NextSendMessageKey(&state)
	if err != nil {
		t.Fatalf("second message key: %v", err)
	}

	if first.Counter != 0 || second.Counter != 1 || state.SendCounter != 2 {
		t.Fatalf("unexpected counters: first=%d second=%d state=%d", first.Counter, second.Counter, state.SendCounter)
	}
	if bytes.Equal(first.Key, second.Key) {
		t.Fatal("message keys must be unique across chain steps")
	}
	if bytes.Equal(originalChain, state.SendChainKey) {
		t.Fatal("send chain key must advance")
	}
	if len(first.Key) != 32 || len(first.Nonce) != 12 {
		t.Fatalf("message key sizes = key %d nonce %d", len(first.Key), len(first.Nonce))
	}
}

func TestNextRecvMessageKeyForHandlesOutOfOrderDeliveryAndReplay(t *testing.T) {
	shared := bytes.Repeat([]byte{0x34}, 32)
	info := []byte("out-of-order-session")
	sender, err := DeriveInitialRatchetState(shared, info, true)
	if err != nil {
		t.Fatalf("sender state: %v", err)
	}
	receiver, err := DeriveInitialRatchetState(shared, info, false)
	if err != nil {
		t.Fatalf("receiver state: %v", err)
	}

	sent := make([]MessageKey, 3)
	for i := range sent {
		sent[i], err = NextSendMessageKey(&sender)
		if err != nil {
			t.Fatalf("send key %d: %v", i, err)
		}
	}

	latest, err := NextRecvMessageKeyFor(&receiver, 2)
	if err != nil {
		t.Fatalf("receive latest key: %v", err)
	}
	if !bytes.Equal(latest.Key, sent[2].Key) || !bytes.Equal(latest.Nonce, sent[2].Nonce) {
		t.Fatal("out-of-order target key does not match sender key")
	}
	if receiver.RecvCounter != 3 || len(receiver.SkippedMessageKeys) != 2 {
		t.Fatalf("receiver state after gap = counter %d, skipped %d", receiver.RecvCounter, len(receiver.SkippedMessageKeys))
	}

	for _, counter := range []uint32{0, 1} {
		message, err := NextRecvMessageKeyFor(&receiver, counter)
		if err != nil {
			t.Fatalf("consume skipped key %d: %v", counter, err)
		}
		if !bytes.Equal(message.Key, sent[counter].Key) || !bytes.Equal(message.Nonce, sent[counter].Nonce) {
			t.Fatalf("skipped key %d does not match sender key", counter)
		}
	}
	if len(receiver.SkippedMessageKeys) != 0 {
		t.Fatalf("skipped keys remaining = %d, want 0", len(receiver.SkippedMessageKeys))
	}

	if _, err := NextRecvMessageKeyFor(&receiver, 0); !errors.Is(err, ErrMessageKeyUnavailable) {
		t.Fatalf("replayed counter error = %v, want %v", err, ErrMessageKeyUnavailable)
	}
}

func TestNextRecvMessageKeyForRejectsExcessiveGapWithoutMutation(t *testing.T) {
	state, err := DeriveInitialRatchetState(bytes.Repeat([]byte{0x22}, 32), []byte("gap-limit"), false)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	originalChain := cloneBytes(state.RecvChainKey)

	_, err = NextRecvMessageKeyFor(&state, uint32(maxSkippedMessageKeys+1))
	if !errors.Is(err, ErrSkippedMessageLimit) {
		t.Fatalf("gap error = %v, want %v", err, ErrSkippedMessageLimit)
	}
	if state.RecvCounter != 0 || !bytes.Equal(state.RecvChainKey, originalChain) || len(state.SkippedMessageKeys) != 0 {
		t.Fatal("excessive skipped-message request must not mutate ratchet state")
	}
}

func TestRatchetRejectsCounterExhaustion(t *testing.T) {
	state, err := DeriveInitialRatchetState(bytes.Repeat([]byte{0x66}, 32), []byte("counter-limit"), true)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	state.SendCounter = maxRatchetMessageCounter
	if _, err := NextSendMessageKey(&state); !errors.Is(err, ErrRatchetCounterExhausted) {
		t.Fatalf("send counter error = %v, want %v", err, ErrRatchetCounterExhausted)
	}
	state.RecvCounter = maxRatchetMessageCounter
	if _, err := NextRecvMessageKey(&state); !errors.Is(err, ErrRatchetCounterExhausted) {
		t.Fatalf("receive counter error = %v, want %v", err, ErrRatchetCounterExhausted)
	}
}

func TestRootRatchetDerivesNewRootAndChain(t *testing.T) {
	root := bytes.Repeat([]byte{0x21}, 32)
	dh := bytes.Repeat([]byte{0x33}, 32)

	out, err := RootRatchet(root, dh)
	if err != nil {
		t.Fatalf("root ratchet: %v", err)
	}
	if len(out.RootKey) != 32 || len(out.ChainKey) != 32 {
		t.Fatalf("unexpected output sizes: root=%d chain=%d", len(out.RootKey), len(out.ChainKey))
	}
	if bytes.Equal(out.RootKey, root) {
		t.Fatal("root ratchet must change the root key")
	}
}

func TestRatchetRejectsInvalidState(t *testing.T) {
	if _, err := NewRatchetState(nil, make([]byte, 32), make([]byte, 32)); err == nil {
		t.Fatal("expected invalid root key error")
	}

	var state RatchetState
	if _, err := NextSendMessageKey(&state); err == nil {
		t.Fatal("expected invalid send chain error")
	}
}

func TestSessionStateCopiesRatchetMaterial(t *testing.T) {
	state, err := DeriveInitialRatchetState(bytes.Repeat([]byte{0x55}, 32), []byte("copy-test"), true)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	var session SessionState
	session.ApplyRatchetState(state)
	state.RootKey[0] ^= 0xff

	if bytes.Equal(session.RootKey, state.RootKey) {
		t.Fatal("session must copy ratchet key material")
	}

	state.SkippedMessageKeys[7] = MessageKey{Key: bytes.Repeat([]byte{0x01}, 32), Nonce: bytes.Repeat([]byte{0x02}, 12), Counter: 7}
	session.ApplyRatchetState(state)
	state.SkippedMessageKeys[7].Key[0] ^= 0xff
	if bytes.Equal(session.SkippedMessageKeys[7].Key, state.SkippedMessageKeys[7].Key) {
		t.Fatal("session must copy skipped message key material")
	}
}
