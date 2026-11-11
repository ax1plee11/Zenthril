import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { x25519 } from "@noble/curves/ed25519.js";
import { describe, expect, it } from "vitest";
import { dhRatchetTurn } from "../../crypto/ratchet";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  type PairwiseSessionState,
} from "./pairwiseSession";
import type { KeyBundleAPI, StoredDeviceKeyBundle } from "./types";

/**
 * Design validation for the sender-side DH ratchet.
 *
 * Production never performs a sender-side step, so this file composes the
 * shipped primitives to test whether the formulas agree on both sides before any
 * production change is written.
 *
 * Formulas under test, per the documented Double Ratchet:
 *
 *   sender   : DH(new own private, peer CURRENT public)
 *   receiver : DH(own CURRENT private, new peer public)
 *
 * Both sides evaluate the DH over the same unordered key pair, so they agree.
 * dhRatchetTurn implements the receiver line. The sender line does not exist in
 * production and is expressed here only to validate the design.
 */

function peerBundle(userId: string): { local: StoredDeviceKeyBundle; api: KeyBundleAPI } {
  const local = createDeviceKeyBundle(userId, `${userId} device`, 2) as StoredDeviceKeyBundle;
  const request = toRegisterDeviceRequest(local as never);
  const firstPreKey = request.one_time_prekeys[0];
  if (!firstPreKey) throw new Error("no one-time prekey");
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

/** Receiver-side step, byte-for-byte what production performs. */
function receiverTurn(state: PairwiseSessionState, newPeerPublic: Uint8Array): PairwiseSessionState {
  const turn = dhRatchetTurn(state.rootKey, state.dhSendPrivate, state.dhSendPublic, newPeerPublic);
  return {
    ...state,
    rootKey: turn.newRootKey,
    receiveChainKey: turn.newRecvChainKey,
    sendChainKey: turn.newSendChainKey,
    dhSendPrivate: turn.newDHPrivate,
    dhSendPublic: turn.newDHPublic,
    dhRecvPublic: Uint8Array.from(newPeerPublic),
    previousCounter: state.sendCounter,
    sendCounter: 0,
    receiveCounter: 0,
    skippedMessageKeys: new Map(),
  };
}

/**
 * Sender-side step that production lacks.
 *
 * DESIGN FINDING: this cannot reuse dhRatchetTurn. That primitive returns the
 * results of its SECOND root-ratchet step, whereas the sender must mirror the
 * receiver's FIRST step in order to derive the same chain and the same root.
 * The formula is therefore expressed explicitly here using the same HKDF inputs
 * that crypto/ratchet.ts uses for rootRatchet.
 *
 * sender: DH(new own private, peer CURRENT public) -> rootKdf -> new root + new sending chain
 * receiver: DH(own CURRENT private, new peer public) -> rootKdf -> new root + new receiving chain
 */
function senderTurn(
  state: PairwiseSessionState,
  newPair: { publicKey: Uint8Array; secretKey: Uint8Array },
): PairwiseSessionState {
  const dhOutput = x25519.getSharedSecret(newPair.secretKey, state.dhRecvPublic);
  const material = hkdf(
    sha256,
    dhOutput,
    state.rootKey,
    new TextEncoder().encode("zenthril-ratchet-v1:root"),
    64,
  );
  return {
    ...state,
    rootKey: material.slice(0, 32),
    sendChainKey: material.slice(32, 64),
    dhSendPrivate: newPair.secretKey.slice(),
    dhSendPublic: newPair.publicKey.slice(),
    sendCounter: 0,
  };
}

function newDhPair(): { publicKey: Uint8Array; secretKey: Uint8Array } {
  return x25519.keygen();
}

describe("Double Ratchet design validation", () => {
  it("the shipped receiver step consumes the peer public and the own current private", () => {
    const { bobState } = establish();
    const newPeer = new Uint8Array(32).fill(0x31);
    const turned = receiverTurn(bobState, newPeer);

    expect(Array.from(turned.dhRecvPublic)).toEqual(Array.from(newPeer));
    expect(Array.from(turned.rootKey)).not.toEqual(Array.from(bobState.rootKey));
  });

  it("reusing the receiver primitive on the sending side does not interoperate", () => {
    // SF-1 expressed executably: dhRatchetTurn cannot be reused as the sender
    // step, because the two sides then evaluate different DH outputs.
    const { aliceState, bobState } = establish();

    const aliceMisused = dhRatchetTurn(
      aliceState.rootKey,
      aliceState.dhSendPrivate,
      aliceState.dhSendPublic,
      aliceState.dhRecvPublic,
    );
    const bobTurned = receiverTurn(bobState, aliceMisused.newDHPublic);

    const aliceSends = nextSendMessageKey({ ...aliceState, sendChainKey: aliceMisused.newSendChainKey });
    const bobReads = nextReceiveMessageKey(bobTurned, aliceSends.counter);

    expect(Array.from(aliceSends.messageKey)).not.toEqual(Array.from(bobReads.messageKey));
  });

  it("shipped bootstrap leaves both sides with dhSendPublic equal to the peer's dhRecvPublic", () => {
    const { aliceState, bobState } = establish();
    expect(Array.from(aliceState.dhSendPublic)).toEqual(Array.from(bobState.dhRecvPublic));
    expect(Array.from(bobState.dhSendPublic)).toEqual(Array.from(aliceState.dhRecvPublic));
  });

  it("design: a sender step and a receiver step agree on the same chain", () => {
    const { aliceState, bobState } = establish();

    // Bob, the receiver of this exchange, turns because he observes Alice's
    // advertised key. Alice performs the sender step for the same new key pair.
    const aliceNew = newDhPair();
    const aliceSent = senderTurn(aliceState, aliceNew);
    const bobTurned = receiverTurn(bobState, aliceNew.publicKey);

    // Alice's sending chain after the sender step must be the chain Bob derives
    // as his receiving chain.
    const aliceSends = nextSendMessageKey(aliceSent);
    const bobReads = nextReceiveMessageKey(bobTurned, aliceSends.counter);

    // The chain-derived message keys must agree. This is the property that makes
    // the two sides interoperable.
    expect(Array.from(aliceSends.messageKey)).toEqual(Array.from(bobReads.messageKey));

    // CORRECTION to the earlier design note: the root keys do NOT have to match
    // after a turn. dhRatchetTurn returns the root of its second step, which is
    // the receiver's position, while the sender mirrors only the first step. In
    // the documented model each side advances independently and therefore holds
    // a different root once a ratchet exchange completes. Requiring equal roots
    // was an error in the previous design document.
    expect(Array.from(aliceSent.rootKey)).not.toEqual(Array.from(bobTurned.rootKey));
  });

  it("design: two consecutive ratchets keep both sides in agreement", () => {
    const { aliceState, bobState } = establish();
    let alice = aliceState;
    let bob = bobState;
    let aliceAdvertised: Uint8Array<ArrayBufferLike> = new Uint8Array(32);

    for (let i = 0; i < 3; i++) {
      // The sender step is needed ONLY when a side has not just received a
      // ratchet. A receiver turn already returns a fresh DH pair and a ready
      // sending chain, so applying a sender step on top of it would rotate twice
      // and break agreement. After the initial step the cascade self-sustains.
      if (i === 0) {
        const aliceNew = newDhPair();
        alice = senderTurn(alice, aliceNew);
        aliceAdvertised = aliceNew.publicKey;
      } else {
        // Alice keeps using the sending chain produced by her own receiver turn.
        aliceAdvertised = alice.dhSendPublic;
      }

      const a2b = nextSendMessageKey(alice);
      alice = a2b.state;

      // Bob observes the advertised key and performs the shipped receiver step.
      bob = receiverTurn(bob, aliceAdvertised);
      const b2aFromAlice = nextReceiveMessageKey(bob, a2b.counter);
      expect(Array.from(b2aFromAlice.messageKey), `iteration ${i}`).toEqual(
        Array.from(a2b.messageKey),
      );
      bob = b2aFromAlice.state;

      // Bob replies with the sending chain his own turn produced.
      const b2a = nextSendMessageKey(bob);
      bob = b2a.state;

      // Alice observes Bob's new key, performs the receiver step, and afterwards
      // already holds a usable sending chain.
      alice = receiverTurn(alice, bob.dhSendPublic);
      const a2bFromBob = nextReceiveMessageKey(alice, b2a.counter);
      expect(Array.from(a2bFromBob.messageKey), `iteration ${i} reverse`).toEqual(
        Array.from(b2a.messageKey),
      );
      alice = a2bFromBob.state;
    }

    // Each rotation changed the root key, so no stagnation.
    expect(Array.from(alice.rootKey)).not.toEqual(Array.from(aliceState.rootKey));
    expect(Array.from(bob.rootKey)).not.toEqual(Array.from(bobState.rootKey));
  });

  it("design: a snapshot taken before a real ratchet stops predicting keys", () => {
    const { aliceState } = establish();

    const before = nextSendMessageKey(aliceState);
    const captured: PairwiseSessionState = {
      ...aliceState,
      rootKey: Uint8Array.from(aliceState.rootKey),
      sendChainKey: Uint8Array.from(aliceState.sendChainKey),
      receiveChainKey: Uint8Array.from(aliceState.receiveChainKey),
      dhSendPrivate: Uint8Array.from(aliceState.dhSendPrivate),
      dhSendPublic: Uint8Array.from(aliceState.dhSendPublic),
      dhRecvPublic: Uint8Array.from(aliceState.dhRecvPublic),
      skippedMessageKeys: new Map(),
    };

    // A real sender-side ratchet happens.
    const aliceAfter = senderTurn(before.state, newDhPair());
    const live = nextSendMessageKey(aliceAfter);
    const fromCaptured = nextSendMessageKey(captured);

    expect(Array.from(fromCaptured.messageKey)).not.toEqual(Array.from(live.messageKey));
  });
});