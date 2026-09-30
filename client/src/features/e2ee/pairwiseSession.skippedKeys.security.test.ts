import { describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  importRatchetMessageKey,
  initiatePairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  performDHRatchetTurn,
  restorePairwiseSession,
  serializePairwiseSession,
  type PairwiseSessionState,
  type RatchetedMessageKey,
} from "./pairwiseSession";
import type { KeyBundleAPI } from "./types";

function publicKeyBundle(userId: string, deviceName: string): {
  local: ReturnType<typeof createDeviceKeyBundle>;
  publicBundle: KeyBundleAPI;
} {
  const local = createDeviceKeyBundle(userId, deviceName, 2);
  const request = toRegisterDeviceRequest(local);
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
      one_time_prekey: {
        key_id: firstPreKey.key_id,
        public_key: firstPreKey.public_key,
      },
      fingerprint: "test-fingerprint",
    },
  };
}

function sessionPair(): { alice: PairwiseSessionState; bob: PairwiseSessionState } {
  const alice = publicKeyBundle("alice", "Alice device");
  const bob = publicKeyBundle("bob", "Bob device");
  const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
  const accepted = acceptPairwiseSession(bob.local, initiated.header);
  return { alice: initiated.state, bob: accepted.state };
}

function toBufferSource(value: Uint8Array): ArrayBuffer {
  const out = new ArrayBuffer(value.length);
  new Uint8Array(out).set(value);
  return out;
}

interface SealedMessage {
  counter: number;
  ciphertext: ArrayBuffer;
  nonce: Uint8Array;
}

/**
 * Mirrors the production wrapping step in messageEnvelopes.ts: the ratchet
 * message key becomes an AES-GCM key and the ratchet message nonce becomes
 * the AEAD IV.
 */
async function seal(plaintext: string, step: RatchetedMessageKey): Promise<SealedMessage> {
  const nonce = Uint8Array.from(step.messageNonce);
  const { key } = await importRatchetMessageKey(step.messageKey, step.messageNonce);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toBufferSource(nonce), tagLength: 128 },
    key,
    new TextEncoder().encode(plaintext),
  );
  return { counter: step.counter, ciphertext, nonce };
}

async function open(sealed: SealedMessage, step: RatchetedMessageKey): Promise<string> {
  const { key, nonce } = await importRatchetMessageKey(step.messageKey, step.messageNonce);
  expect(Array.from(nonce)).toEqual(Array.from(sealed.nonce));
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: toBufferSource(nonce), tagLength: 128 },
    key,
    sealed.ciphertext,
  );
  return new TextDecoder().decode(plaintext);
}

function snapshot(state: PairwiseSessionState) {
  return {
    receiveChainKey: Array.from(state.receiveChainKey),
    receiveCounter: state.receiveCounter,
    skipped: Array.from(state.skippedMessageKeys.keys()),
  };
}

describe("TASK #CRYPTO-007 regression: skipped message key handling", () => {
  it("decrypts out-of-order messages that are served from the skipped store", async () => {
    const { alice, bob } = sessionPair();

    const sent: SealedMessage[] = [];
    let sendState = alice;
    for (const text of ["first", "second", "third"]) {
      const step = nextSendMessageKey(sendState);
      sendState = step.state;
      sent.push(await seal(text, step));
    }
    expect(sent.map(m => m.counter)).toEqual([0, 1, 2]);

    // Deliver strictly out of order: 2, then 0, then 1.
    let bobState = bob;
    for (const counter of [2, 0, 1]) {
      const sealed = sent.find(m => m.counter === counter)!;
      const step = nextReceiveMessageKey(bobState, counter);
      bobState = step.state;
      expect(step.counter).toBe(counter);
      await expect(open(sealed, step)).resolves.toBe(
        counter === 0 ? "first" : counter === 1 ? "second" : "third",
      );
    }
    expect(bobState.skippedMessageKeys.size).toBe(0);
  });

  it("derives an identical key and nonce for a counter regardless of arrival order", () => {
    const { alice, bob } = sessionPair();

    let sendState = alice;
    const inOrder: RatchetedMessageKey[] = [];
    for (let i = 0; i < 3; i++) {
      const step = nextSendMessageKey(sendState);
      sendState = step.state;
      inOrder.push(step);
    }

    // Same counters derived by the receiver in order.
    let inOrderState = bob;
    const inOrderRecv: RatchetedMessageKey[] = [];
    for (let i = 0; i < 3; i++) {
      const step = nextReceiveMessageKey(inOrderState, i);
      inOrderState = step.state;
      inOrderRecv.push(step);
    }

    // Same counters derived by the receiver out of order (skipped store path).
    let outOfOrderState = bob;
    const outOfOrderRecv: RatchetedMessageKey[] = [];
    for (const counter of [2, 0, 1]) {
      const step = nextReceiveMessageKey(outOfOrderState, counter);
      outOfOrderState = step.state;
      outOfOrderRecv.push(step);
    }

    for (const sender of inOrder) {
      const inOrderMatch = inOrderRecv.find(r => r.counter === sender.counter)!;
      const outOfOrderMatch = outOfOrderRecv.find(r => r.counter === sender.counter)!;
      expect(Array.from(inOrderMatch.messageKey)).toEqual(Array.from(sender.messageKey));
      expect(Array.from(outOfOrderMatch.messageKey)).toEqual(Array.from(sender.messageKey));
      expect(Array.from(inOrderMatch.messageNonce)).toEqual(Array.from(outOfOrderMatch.messageNonce));
    }
  });

  it("does not mutate the caller's state when a receive key is derived", () => {
    const { alice, bob } = sessionPair();
    const sendState = alice;
    nextSendMessageKey(sendState);
    nextSendMessageKey(sendState);

    const before = snapshot(bob);
    const step = nextReceiveMessageKey(bob, 2);
    expect(snapshot(bob)).toEqual(before);
    expect(step.state).not.toBe(bob);
    expect(step.state.skippedMessageKeys).not.toBe(bob.skippedMessageKeys);
  });

  it("rejects malformed counters without mutating state", () => {
    const { bob } = sessionPair();
    const before = snapshot(bob);

    for (const counter of [Number.NaN, Number.POSITIVE_INFINITY, 1.5, -1, Number.MAX_SAFE_INTEGER + 2]) {
      expect(() => nextReceiveMessageKey(bob, counter)).toThrow();
      expect(snapshot(bob)).toEqual(before);
    }
  });

  it("keeps the skipped-key budget fail-closed and non-destructive", () => {
    const { bob } = sessionPair();
    const before = snapshot(bob);

    expect(() => nextReceiveMessageKey(bob, 2001)).toThrow("Skipped message key limit exceeded");
    expect(snapshot(bob)).toEqual(before);
  });

  it("consumes each skipped key exactly once", () => {
    const { alice, bob } = sessionPair();
    nextSendMessageKey(alice);
    nextSendMessageKey(alice);
    nextSendMessageKey(alice);

    let bobState = bob;
    const third = nextReceiveMessageKey(bobState, 2);
    bobState = third.state;
    const first = nextReceiveMessageKey(bobState, 0);
    bobState = first.state;
    expect(bobState.skippedMessageKeys.has(0)).toBe(false);

    expect(() => nextReceiveMessageKey(bobState, 0)).toThrow("Message key unavailable for skipped counter");
  });

  it("refuses a DH ratchet turn when the private DH key was not recovered", () => {
    const { bob } = sessionPair();
    const restored = restorePairwiseSession(serializePairwiseSession(bob));

    // The persisted format intentionally has no dhSendPrivate field, so a
    // restored session has no usable DH private key. X25519 clamps an all-zero
    // scalar into a publicly known constant, so turning the ratchet here would
    // derive attacker-computable keys. It must fail closed instead.
    expect(Array.from(restored.dhSendPrivate).every(byte => byte === 0)).toBe(true);
    expect(() => performDHRatchetTurn(restored, new Uint8Array(32).fill(0x11))).toThrow();
  });

  it("keeps a usable DH private key on an initiated session", () => {
    // Regression: initiatePairwiseSession zeroed its temporary DH key pair in a
    // cleanup block while the session still aliased that buffer, so the
    // initiator's first ratchet turn ran with an all-zero private key.
    const { alice } = sessionPair();
    expect(alice.dhSendPrivate.some(byte => byte !== 0)).toBe(true);
    expect(() => performDHRatchetTurn(alice, new Uint8Array(32).fill(0x22))).not.toThrow();
  });

  it("keeps a usable DH private key on an accepted session", () => {
    const { bob } = sessionPair();
    expect(bob.dhSendPrivate.some(byte => byte !== 0)).toBe(true);
  });
});
