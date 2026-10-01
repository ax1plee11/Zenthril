import { describe, expect, it } from "vitest";
import { dhRatchetTurn } from "../../crypto/ratchet";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  restorePairwiseSession,
  serializePairwiseSession,
  type PairwiseSessionState,
} from "./pairwiseSession";
import type { KeyBundleAPI, StoredDeviceKeyBundle, StoredPairwiseSession } from "./types";

/**
 * Double Ratchet lifecycle as actually implemented.
 *
 * These tests establish when the DH ratchet advances, because that behaviour
 * determines both whether post-compromise security holds and whether the DH
 * private key has to survive a restart.
 */

function peerBundle(userId: string): { local: StoredDeviceKeyBundle; api: KeyBundleAPI } {
  const local = createDeviceKeyBundle(userId, `${userId} device`, 2) as StoredDeviceKeyBundle;
  const request = toRegisterDeviceRequest(local as never);
  const firstPreKey = request.one_time_prekeys[0];
  if (!firstPreKey) throw new Error("test device has no one-time prekey");
  return {
    local,
    api: {
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
  const alice = peerBundle("alice");
  const bob = peerBundle("bob");
  const initiated = initiatePairwiseSession(alice.local, bob.api);
  const accepted = acceptPairwiseSession(bob.local, initiated.header);
  return {
    alice: alice.local,
    bob: accepted.updatedLocalBundle,
    aliceState: initiated.state,
    bobState: accepted.state,
  };
}

/**
 * Mirrors decryptChannelMessage in messageEnvelopes.ts:116-118. Returns the
 * state that would actually be used, without invoking the private-key guard.
 */
function receive(
  state: PairwiseSessionState,
  envelopeDhPublicKey: Uint8Array,
  counter: number,
): PairwiseSessionState {
  const current = new Uint8Array(state.dhRecvPublic);
  const differs =
    envelopeDhPublicKey.length > 0 &&
    (envelopeDhPublicKey.length !== current.length ||
      envelopeDhPublicKey.some((byte, i) => byte !== current[i]));

  let next = state;
  if (differs) {
    // performDHRatchetTurn, called only from the decrypt path.
    next = turnForTest(state, envelopeDhPublicKey);
  }
  return nextReceiveMessageKey(next, counter).state;
}

/**
 * The exact body of performDHRatchetTurn after its fail-closed guard. Kept
 * separate so the lifecycle can be observed for a session that still holds a
 * usable private key, which is the case immediately after bootstrap.
 */
function turnForTest(state: PairwiseSessionState, newPeerDHPublic: Uint8Array): PairwiseSessionState {
  const turn = dhRatchetTurn(
    state.rootKey,
    state.dhSendPrivate,
    state.dhSendPublic,
    newPeerDHPublic,
  );
  return {
    ...state,
    rootKey: turn.newRootKey,
    receiveChainKey: turn.newRecvChainKey,
    sendChainKey: turn.newSendChainKey,
    dhSendPrivate: turn.newDHPrivate,
    dhSendPublic: turn.newDHPublic,
    dhRecvPublic: Uint8Array.from(newPeerDHPublic),
    previousCounter: state.sendCounter,
    sendCounter: 0,
    receiveCounter: 0,
    skippedMessageKeys: new Map(),
  };
}

describe("Double Ratchet lifecycle as implemented", () => {
  it("bootstrap leaves each side's send key equal to what the peer already expects", () => {
    const { aliceState, bobState } = establish();

    // The responder advertises its signed prekey as its DH send key, and the
    // initiator stores exactly that value as the responder's DH key. The
    // responder's first message therefore cannot look like a rotation.
    expect(Array.from(aliceState.dhRecvPublic)).toEqual(Array.from(bobState.dhSendPublic));

    // Symmetrically the initiator's header DH key is what the responder expects.
    expect(Array.from(bobState.dhRecvPublic)).toEqual(Array.from(aliceState.dhSendPublic));
  });

  it("never advances the DH ratchet during ordinary bidirectional traffic", () => {
    const { aliceState, bobState } = establish();

    let alice = aliceState;
    let bob = bobState;
    const aliceRootAtStart = Uint8Array.from(alice.rootKey);
    const bobRootAtStart = Uint8Array.from(bob.rootKey);
    const aliceDhAtStart = Uint8Array.from(alice.dhSendPublic);
    const bobDhAtStart = Uint8Array.from(bob.dhSendPublic);

    for (let i = 0; i < 5; i++) {
      // Alice sends. Bob receives the key Alice advertises.
      const a2b = nextSendMessageKey(alice);
      alice = a2b.state;
      bob = receive(bob, Uint8Array.from(a2b.state.dhSendPublic), a2b.counter);

      // Bob replies. Alice receives the key Bob advertises.
      const b2a = nextSendMessageKey(bob);
      bob = b2a.state;
      alice = receive(alice, Uint8Array.from(b2a.state.dhSendPublic), b2a.counter);
    }

    expect(Array.from(alice.rootKey)).toEqual(Array.from(aliceRootAtStart));
    expect(Array.from(bob.rootKey)).toEqual(Array.from(bobRootAtStart));
    expect(Array.from(alice.dhSendPublic)).toEqual(Array.from(aliceDhAtStart));
    expect(Array.from(bob.dhSendPublic)).toEqual(Array.from(bobDhAtStart));
  });

  it("keeps the symmetric chains working across many messages", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    for (let i = 0; i < 8; i++) {
      const sent = nextSendMessageKey(alice);
      alice = sent.state;
      const received = nextReceiveMessageKey(bob, sent.counter);
      bob = received.state;
      expect(Array.from(received.messageKey)).toEqual(Array.from(sent.messageKey));

      const reply = nextSendMessageKey(bob);
      bob = reply.state;
      const replyReceived = nextReceiveMessageKey(alice, reply.counter);
      alice = replyReceived.state;
      expect(Array.from(replyReceived.messageKey)).toEqual(Array.from(reply.messageKey));
    }

    expect(aliceState.sendCounter).toBe(0);
    expect(bobState.receiveCounter).toBe(0);
    expect(alice.sendCounter).toBe(8);
    expect(bob.receiveCounter).toBe(8);
    expect(bob.sendCounter).toBe(8);
    expect(alice.receiveCounter).toBe(8);
  });

  // SECURITY: the property a Double Ratchet exists to provide. A snapshot of the
  // state must stop predicting future message keys once the ratchet advances.
  it("post-compromise: a captured state still predicts future keys because the DH ratchet never advances", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    // Two messages, then the attacker captures Alice's full state.
    const first = nextSendMessageKey(alice);
    alice = first.state;
    bob = nextReceiveMessageKey(bob, first.counter).state;
    const second = nextSendMessageKey(alice);
    alice = second.state;
    bob = nextReceiveMessageKey(bob, second.counter).state;

    const captured: PairwiseSessionState = {
      ...alice,
      rootKey: Uint8Array.from(alice.rootKey),
      sendChainKey: Uint8Array.from(alice.sendChainKey),
      receiveChainKey: Uint8Array.from(alice.receiveChainKey),
      dhSendPrivate: Uint8Array.from(alice.dhSendPrivate),
      dhSendPublic: Uint8Array.from(alice.dhSendPublic),
      dhRecvPublic: Uint8Array.from(alice.dhRecvPublic),
      skippedMessageKeys: new Map(alice.skippedMessageKeys),
    };

    // Traffic continues normally: two further messages from Alice, with Bob
    // receiving the first so the model stays faithful to the real exchange.
    const liveThird = nextSendMessageKey(alice);
    alice = liveThird.state;
    expect(Array.from(nextReceiveMessageKey(bob, liveThird.counter).messageKey)).toEqual(
      Array.from(liveThird.messageKey),
    );
    const liveFourth = nextSendMessageKey(alice);
    alice = liveFourth.state;

    // The captured state is advanced the same number of steps. In a working
    // Double Ratchet the two sequences must diverge after a ratchet step.
    const capturedThird = nextSendMessageKey(captured);
    const capturedFourth = nextSendMessageKey(capturedThird.state);

    expect(Array.from(capturedThird.messageKey)).toEqual(Array.from(liveThird.messageKey));
    expect(Array.from(capturedFourth.messageKey)).toEqual(Array.from(liveFourth.messageKey));
  });

  it("post-compromise: a captured state also reproduces the live receive chain", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;

    const sent = nextSendMessageKey(alice);
    alice = sent.state;
    bob = nextReceiveMessageKey(bob, sent.counter).state;

    const captured: PairwiseSessionState = {
      ...alice,
      rootKey: Uint8Array.from(alice.rootKey),
      receiveChainKey: Uint8Array.from(alice.receiveChainKey),
      sendChainKey: Uint8Array.from(alice.sendChainKey),
      skippedMessageKeys: new Map(alice.skippedMessageKeys),
    };

    // Bob sends; the live session and the captured state derive the same key.
    const reply = nextSendMessageKey(bob);
    bob = reply.state;
    const live = nextReceiveMessageKey(alice, reply.counter);
    const fromCaptured = nextReceiveMessageKey(captured, reply.counter);

    expect(Array.from(fromCaptured.messageKey)).toEqual(Array.from(live.messageKey));
  });

  it("a DH turn is reachable when a peer genuinely advertises a different key", () => {
    const { aliceState } = establish();
    const rotated = new Uint8Array(32).fill(0x7f);

    // The receive path does detect and act on a real change.
    const turned = turnForTest(aliceState, rotated);
    expect(Array.from(turned.rootKey)).not.toEqual(Array.from(aliceState.rootKey));
    expect(Array.from(turned.dhSendPublic)).not.toEqual(Array.from(aliceState.dhSendPublic));
    expect(Array.from(turned.dhRecvPublic)).toEqual(Array.from(rotated));
  });

  it("the persisted representation carries no dhSendPrivate, which is why a restart cannot complete a turn", () => {
    const { aliceState } = establish();
    const stored: StoredPairwiseSession = serializePairwiseSession(aliceState);

    expect(Object.prototype.hasOwnProperty.call(stored, "dhSendPrivate")).toBe(false);
    expect(typeof stored.dhSendPublic).toBe("string");

    const restored = restorePairwiseSession(stored);
    if (!restored) throw new Error("restore failed");
    expect(Array.from(restored.dhSendPrivate).every(byte => byte === 0)).toBe(true);
  });
});
