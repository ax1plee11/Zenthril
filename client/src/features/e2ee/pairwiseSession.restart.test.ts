import { describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  performDHRatchetTurn,
  restorePairwiseSession,
  serializePairwiseSession,
  type PairwiseSessionState,
} from "./pairwiseSession";
import type { KeyBundleAPI, StoredDeviceKeyBundle } from "./types";

/**
 * Restart behaviour of a pairwise session.
 *
 * A restart is modelled by a serialize/restore round-trip, because that is
 * exactly what crosses the persistence boundary: serializePairwiseSession writes
 * the stored representation and restorePairwiseSession rebuilds the runtime
 * state. The round-trip therefore reproduces a restart faithfully, including the
 * field that persistence does not carry.
 */

function publicBundle(userId: string, deviceName: string): {
  local: StoredDeviceKeyBundle;
  publicBundle: KeyBundleAPI;
} {
  const local = createDeviceKeyBundle(userId, deviceName, 2) as StoredDeviceKeyBundle;
  const request = toRegisterDeviceRequest(local as never);
  const firstPreKey = request.one_time_prekeys[0];
  if (!firstPreKey) throw new Error("test device has no one-time prekey");
  return {
    local,
    publicBundle: {
      user_id: userId,
      device_id: request.device_id,
      identity_public_key: request.identity_public_key,
      identity_dh_public_key: request.identity_dh_public_key,
      signed_pre_key_id: request.signed_pre_key_id,
      signed_pre_key: request.signed_pre_key,
      signed_pre_key_signature: request.signed_pre_key_signature,
      one_time_prekey: { key_id: firstPreKey.key_id, public_key: firstPreKey.public_key },
      fingerprint: "test-fingerprint",
    },
  };
}

function establish(): {
  alice: StoredDeviceKeyBundle;
  bob: StoredDeviceKeyBundle;
  aliceState: PairwiseSessionState;
  bobState: PairwiseSessionState;
} {
  const alice = publicBundle("alice", "Alice device");
  const bob = publicBundle("bob", "Bob device");
  const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
  const accepted = acceptPairwiseSession(bob.local, initiated.header);
  return {
    alice: alice.local,
    bob: accepted.updatedLocalBundle,
    aliceState: initiated.state,
    bobState: accepted.state,
  };
}

/** Simulates an application restart for one side. */
function restart(state: PairwiseSessionState): PairwiseSessionState {
  const restored = restorePairwiseSession(serializePairwiseSession(state));
  if (!restored) throw new Error("session did not survive serialization");
  return restored;
}

function seal(step: { messageKey: Uint8Array; messageNonce: Uint8Array }): Uint8Array {
  return Uint8Array.from(step.messageKey);
}

describe("pairwise session restart behaviour", () => {
  it("restores root key, chains and counters across a restart", () => {
    const { aliceState, bobState } = establish();

    let alice = aliceState;
    let bob = bobState;
    for (let i = 0; i < 3; i++) {
      const sent = nextSendMessageKey(alice);
      alice = sent.state;
      const received = nextReceiveMessageKey(bob, sent.counter);
      bob = received.state;
    }

    const aliceAfterRestart = restart(alice);
    const bobAfterRestart = restart(bob);

    expect(Array.from(aliceAfterRestart.rootKey)).toEqual(Array.from(alice.rootKey));
    expect(Array.from(aliceAfterRestart.sendChainKey)).toEqual(Array.from(alice.sendChainKey));
    expect(Array.from(aliceAfterRestart.receiveChainKey)).toEqual(Array.from(alice.receiveChainKey));
    expect(aliceAfterRestart.sendCounter).toBe(alice.sendCounter);
    expect(aliceAfterRestart.receiveCounter).toBe(alice.receiveCounter);

    expect(Array.from(bobAfterRestart.rootKey)).toEqual(Array.from(bob.rootKey));
    expect(bobAfterRestart.receiveCounter).toBe(bob.receiveCounter);
  });

  it("continues an established session in both directions after one side restarts", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    // Pre-restart traffic so both chains have advanced.
    for (let i = 0; i < 2; i++) {
      const sent = nextSendMessageKey(alice);
      alice = sent.state;
      bob = nextReceiveMessageKey(bob, sent.counter).state;
    }

    // Only Alice restarts.
    alice = restart(alice);

    const sentAfterRestart = nextSendMessageKey(alice);
    alice = sentAfterRestart.state;
    const receivedByBob = nextReceiveMessageKey(bob, sentAfterRestart.counter);

    // Bob derives the same key as Alice produced, so the session survives.
    expect(Array.from(seal(receivedByBob))).toEqual(Array.from(sentAfterRestart.messageKey));
    expect(receivedByBob.state.receiveCounter).toBe(sentAfterRestart.counter + 1);

    // And the reply direction still works on the restored receive chain.
    const reply = nextSendMessageKey(bob);
    const replyReceived = nextReceiveMessageKey(alice, reply.counter);
    expect(Array.from(replyReceived.messageKey)).toEqual(Array.from(reply.messageKey));
  });

  it("survives a restart on both sides simultaneously", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    const first = nextSendMessageKey(alice);
    alice = first.state;
    bob = nextReceiveMessageKey(bob, first.counter).state;

    alice = restart(alice);
    bob = restart(bob);

    const second = nextSendMessageKey(alice);
    alice = second.state;
    const received = nextReceiveMessageKey(bob, second.counter);
    expect(Array.from(received.messageKey)).toEqual(Array.from(second.messageKey));
  });

  it("preserves skipped-message-key handling across a restart", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;

    const m0 = nextSendMessageKey(alice); alice = m0.state;
    const m1 = nextSendMessageKey(alice); alice = m1.state;
    const m2 = nextSendMessageKey(alice); alice = m2.state;

    // Bob receives counter 2 first, creating skipped entries for 0 and 1.
    let bob = nextReceiveMessageKey(bobState, 2).state;
    expect(bob.skippedMessageKeys.size).toBe(2);

    // Bob restarts, then consumes the retained skipped key.
    bob = restart(bob);
    expect(bob.skippedMessageKeys.size).toBe(2);

    const late = nextReceiveMessageKey(bob, 0);
    expect(Array.from(late.messageKey)).toEqual(Array.from(m0.messageKey));
    expect(late.state.skippedMessageKeys.has(0)).toBe(false);
    void m1;
    void m2;
  });

  it("keeps replay protection after a restart", () => {
    const { aliceState, bobState } = establish();
    const sent = nextSendMessageKey(aliceState);
    let bob = nextReceiveMessageKey(bobState, sent.counter).state;

    bob = restart(bob);

    // The same counter must not be served twice, even after a restart.
    expect(() => nextReceiveMessageKey(bob, sent.counter)).toThrow(
      "Message key unavailable for skipped counter",
    );
  });

  it("does not roll back the chain when a restart is followed by a stale counter", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    for (let i = 0; i < 3; i++) {
      const sent = nextSendMessageKey(alice);
      alice = sent.state;
      bob = nextReceiveMessageKey(bob, sent.counter).state;
    }
    const counterBeforeRestart = bob.receiveCounter;

    bob = restart(bob);
    expect(bob.receiveCounter).toBe(counterBeforeRestart);

    // A counter already consumed is refused rather than re-derived.
    expect(() => nextReceiveMessageKey(bob, 0)).toThrow();
  });

  it("fails closed on a DH ratchet turn after a restart", () => {
    const { aliceState } = establish();
    const restored = restart(aliceState);

    // Persistence does not carry dhSendPrivate, so a restored session cannot
    // complete a DH turn. This is the documented CRY-203 limitation and it must
    // surface as a refusal, never as a derived key.
    expect(() => performDHRatchetTurn(restored, new Uint8Array(32).fill(0x42))).toThrow(
      "Pairwise session has no recoverable DH private key",
    );
  });

  it("fails closed on a DH ratchet turn when the persisted key is all zeros", () => {
    const { aliceState } = establish();
    const tampered: PairwiseSessionState = { ...aliceState, dhSendPrivate: new Uint8Array(32) };
    expect(() => performDHRatchetTurn(tampered, new Uint8Array(32).fill(0x42))).toThrow(
      "Pairwise session has no recoverable DH private key",
    );
  });

  it("keeps the peer DH public key stable across a restart so no turn is forced", () => {
    const { aliceState } = establish();
    const restored = restart(aliceState);
    // A restart must not change the advertised DH public key, otherwise the peer
    // would observe a spurious rotation and attempt a ratchet turn.
    expect(Array.from(restored.dhSendPublic)).toEqual(Array.from(aliceState.dhSendPublic));
  });

  it("retains the session identity and peer binding across a restart", () => {
    const { aliceState } = establish();
    const restored = restart(aliceState);
    expect(restored.sessionId).toBe(aliceState.sessionId);
    expect(restored.peerUserId).toBe(aliceState.peerUserId);
    expect(restored.peerDeviceId).toBe(aliceState.peerDeviceId);
    expect(restored.version).toBe(aliceState.version);
  });
});
