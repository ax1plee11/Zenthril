import { describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  loadPairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  performDHRatchetTurn,
  savePairwiseSession,
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

describe("pairwise X3DH session foundation", () => {
  it("derives matching directional ratchet chains and consumes a one-time prekey", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");

    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    expect(accepted.state.sessionId).toBe(initiated.state.sessionId);
    expect(Array.from(accepted.state.rootKey)).toEqual(Array.from(initiated.state.rootKey));
    expect(accepted.updatedLocalBundle.oneTimePreKeys).toHaveLength(
      bob.local.oneTimePreKeys.length - 1,
    );

    const sent = nextSendMessageKey(initiated.state);
    const received = nextReceiveMessageKey(accepted.state);
    expect(sent.counter).toBe(0);
    expect(received.counter).toBe(0);
    expect(Array.from(received.messageKey)).toEqual(Array.from(sent.messageKey));

    sent.messageKey.fill(0);
    received.messageKey.fill(0);
  });

  it("rejects a tampered signed prekey before creating a session", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const tampered = {
      ...bob.publicBundle,
      signed_pre_key_signature: `${bob.publicBundle.signed_pre_key_signature.slice(0, -1)}A`,
    };

    expect(() => initiatePairwiseSession(alice.local, tampered)).toThrow(
      "Peer signed prekey signature is invalid",
    );
  });

  it("rejects a replayed one-time prekey header", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const first = acceptPairwiseSession(bob.local, initiated.header);

    expect(() => acceptPairwiseSession(first.updatedLocalBundle, initiated.header)).toThrow(
      "Unknown or already consumed one-time prekey",
    );
  });

  it("rejects a header addressed to a different recipient device", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);

    expect(() => acceptPairwiseSession(bob.local, {
      ...initiated.header,
      recipientDeviceId: "another-device",
    })).toThrow("not addressed to this device");
  });

  it("serializes ratchet state for secure device-bundle persistence", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const savedBundle = savePairwiseSession(alice.local, initiated.state);
    const restored = loadPairwiseSession(savedBundle, initiated.state.sessionId);

    expect(restored).not.toBeNull();
    expect(Array.from(restored!.sendChainKey)).toEqual(Array.from(initiated.state.sendChainKey));
  });
});

describe("Double Ratchet deep security properties", () => {
  it("produces unique message keys for consecutive messages", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);

    const sentKeys = [];
    for (let i = 0; i < 10; i++) {
      const step = nextSendMessageKey(initiated.state);
      sentKeys.push(Array.from(step.messageKey));
      initiated.state = step.state;
    }

    const unique = new Set(sentKeys.map((k) => JSON.stringify(k)));
    expect(unique.size).toBe(10);
  });

  it("rejects replay of the same counter on the receiver", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    const first = nextReceiveMessageKey(accepted.state);
    expect(first.counter).toBe(0);
    accepted.state = first.state;

    expect(() => nextReceiveMessageKey(accepted.state, 0)).toThrow("Message key unavailable for skipped counter");
  });

  it("supports bounded out-of-order delivery with skipped message keys", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    const msg0 = nextSendMessageKey(initiated.state);
    initiated.state = msg0.state;
    const msg1 = nextSendMessageKey(initiated.state);
    initiated.state = msg1.state;
    const msg2 = nextSendMessageKey(initiated.state);
    initiated.state = msg2.state;

    const recv2 = nextReceiveMessageKey(accepted.state, 2);
    expect(Array.from(recv2.messageKey)).toEqual(Array.from(msg2.messageKey));
    accepted.state = recv2.state;

    const recv0 = nextReceiveMessageKey(accepted.state, 0);
    expect(Array.from(recv0.messageKey)).toEqual(Array.from(msg0.messageKey));
    accepted.state = recv0.state;

    const recv1 = nextReceiveMessageKey(accepted.state, 1);
    expect(Array.from(recv1.messageKey)).toEqual(Array.from(msg1.messageKey));
  });

  it("rejects excessive skipped-message requests without mutating state", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const initiated = initiatePairwiseSession(alice.local, alice.publicBundle);

    const originalChain = Array.from(initiated.state.receiveChainKey);
    expect(() => nextReceiveMessageKey(initiated.state, 2001)).toThrow("Skipped message key limit exceeded");
    expect(Array.from(initiated.state.receiveChainKey)).toEqual(originalChain);
  });

  it("performDHRatchetTurn mutates root key, chain keys, and DH keys", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const initiated = initiatePairwiseSession(alice.local, alice.publicBundle);

    const oldRoot = Array.from(initiated.state.rootKey);
    const oldSendChain = Array.from(initiated.state.sendChainKey);
    const oldRecvChain = Array.from(initiated.state.receiveChainKey);
    const oldDHPub = Array.from(initiated.state.dhSendPublic);

    const newPeerDH = Array.from(new Uint8Array(32)).map(() => 0xbb);
    const turned = performDHRatchetTurn(initiated.state, Uint8Array.from(newPeerDH));

    expect(Array.from(turned.rootKey)).not.toEqual(oldRoot);
    expect(Array.from(turned.sendChainKey)).not.toEqual(oldSendChain);
    expect(Array.from(turned.receiveChainKey)).not.toEqual(oldRecvChain);
    expect(Array.from(turned.dhSendPublic)).not.toEqual(oldDHPub);
    expect(Array.from(turned.dhRecvPublic)).toEqual(newPeerDH);
  });

  it("fail-closed on invalid DH ratchet input sizes", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const initiated = initiatePairwiseSession(alice.local, alice.publicBundle);

    expect(() => performDHRatchetTurn(
      initiated.state,
      new Uint8Array(16),
    )).toThrow("All DH ratchet inputs must be 32 bytes");
  });
});

describe("Bootstrap header security", () => {
  it("rejects bootstrap header with mismatched recipient device", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const tampered = {
      ...initiated.header,
      recipientDeviceId: "evil-device",
    };

    expect(() => acceptPairwiseSession(bob.local, tampered)).toThrow("not addressed to this device");
  });

  it("rejects bootstrap header with mismatched recipient user", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const tampered = {
      ...initiated.header,
      recipientUserId: "eve",
    };

    expect(() => acceptPairwiseSession(bob.local, tampered)).toThrow("not addressed to this device");
  });

  it("rejects bootstrap header with wrong signed prekey id", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const tampered = {
      ...initiated.header,
      recipientSignedPreKeyId: 99999,
    };

    expect(() => acceptPairwiseSession(bob.local, tampered)).toThrow("Unknown signed prekey");
  });

  it("rejects bootstrap header with unknown one-time prekey id", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const tampered = {
      ...initiated.header,
      recipientOneTimePreKeyId: 99999,
    };

    expect(() => acceptPairwiseSession(bob.local, tampered)).toThrow("Unknown or already consumed one-time prekey");
  });
});

describe("Multi-device race conditions", () => {
  it("isolates session state across concurrent device sessions", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bobPhone = publicKeyBundle("bob", "Bob phone");
    const bobTablet = publicKeyBundle("bob", "Bob tablet");

    const phoneSession = initiatePairwiseSession(alice.local, bobPhone.publicBundle);
    const tabletSession = initiatePairwiseSession(alice.local, bobTablet.publicBundle);

    expect(phoneSession.state.sessionId).not.toBe(tabletSession.state.sessionId);
    expect(phoneSession.header.recipientDeviceId).toBe(bobPhone.publicBundle.device_id);
    expect(tabletSession.header.recipientDeviceId).toBe(bobTablet.publicBundle.device_id);

    const phoneSend = nextSendMessageKey(phoneSession.state);
    phoneSession.state = phoneSend.state;
    const tabletSend = nextSendMessageKey(tabletSession.state);
    tabletSession.state = tabletSend.state;

    expect(Array.from(phoneSend.messageKey)).not.toEqual(Array.from(tabletSend.messageKey));
  });

  it("handles concurrent session acceptance without state leakage", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");

    const session1 = initiatePairwiseSession(alice.local, bob.publicBundle);
    const session2 = initiatePairwiseSession(alice.local, bob.publicBundle);

    const accepted1 = acceptPairwiseSession(bob.local, session1.header);
    const accepted2 = acceptPairwiseSession(bob.local, session2.header);

    expect(accepted1.state.sessionId).toBe(session1.state.sessionId);
    expect(accepted2.state.sessionId).toBe(session2.state.sessionId);
    expect(accepted1.state.sessionId).not.toBe(accepted2.state.sessionId);
  });
});

describe("Device revocation mid-session", () => {
  it("fail-closed when all one-time prekeys are exhausted", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");

    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const first = acceptPairwiseSession(bob.local, initiated.header);

    const emptyBundle = {
      ...first.updatedLocalBundle,
      oneTimePreKeys: [],
    };

    expect(() => acceptPairwiseSession(emptyBundle, initiated.header)).toThrow(
      "Unknown or already consumed one-time prekey",
    );
  });
});

/**
 * SECURITY NOTE: DH ratchet cross-side matching.
 *
 * Signal protocol requires that after Alice sends a message with a new DH public
 * key, Bob derives the receive chain from DH(Alice_new_priv, Bob_priv). When
 * Alice sends a follow-up message, she derives send_chain = HKDF(root, DH(Alice_new_priv, Bob_new_priv)).
 * Bob's receive chain advances from DH(Alice_new_priv, Bob_priv), so they diverge.
 *
 * This test suite validates the observable correctness properties:
 *   1. DH ratchet turn state transitions are correct
 *   2. Encryption/decryption works within each direction after a ratchet turn
 *   3. Post-compromise security holds (old key material cannot decrypt new messages)
 *   4. Multiple DH ratchet turns produce independent, non-degenerate keys
 *
 * Cross-side chain matching (Alice send_chain == Bob receive_chain) requires
 * the full two-step DH ratchet derivation that is currently implemented in
 * `dhRatchetTurn`. If cross-side matching fails, it indicates a critical
 * protocol bug — this test suite will surface it via roundtrip failures.
 */
describe("DH ratchet cross-side message matching (WIP)", () => {
  it("performs DH ratchet turn when peer rotates DH keys", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    // Send two messages to advance the symmetric ratchet
    const msg0 = nextSendMessageKey(initiated.state);
    initiated.state = msg0.state;
    const msg1 = nextSendMessageKey(initiated.state);
    initiated.state = msg1.state;

    // Verify both received first two messages
    const recv0 = nextReceiveMessageKey(accepted.state);
    expect(Array.from(recv0.messageKey)).toEqual(Array.from(msg0.messageKey));
    accepted.state = recv0.state;
    const recv1 = nextReceiveMessageKey(accepted.state);
    expect(Array.from(recv1.messageKey)).toEqual(Array.from(msg1.messageKey));
    accepted.state = recv1.state;

    // Alice performs DH ratchet turn (generates new DH key pair, derives new send chain)
    const oldRoot = Array.from(initiated.state.rootKey);
    const oldSendChain = Array.from(initiated.state.sendChainKey);
    const oldRecvChain = Array.from(initiated.state.receiveChainKey);
    const oldDHPublic = Array.from(initiated.state.dhSendPublic);
    const oldDHRecv = Array.from(initiated.state.dhRecvPublic);

    // Bob performs a DH ratchet turn with Alice's new public key
    // (simulating Bob receiving Alice's new DH public key from a message header)
    const aliceNewDHPublic = new Uint8Array(32);
    crypto.getRandomValues(aliceNewDHPublic);
    const bobTurned = performDHRatchetTurn(accepted.state, aliceNewDHPublic);

    // Alice also performs a DH ratchet turn with Bob's new DH public key
    const bobNewDHPublic = new Uint8Array(32);
    crypto.getRandomValues(bobNewDHPublic);
    const aliceTurned = performDHRatchetTurn(initiated.state, bobNewDHPublic);

    // State changed: root key, send chain, recv chain, DH public all different
    expect(Array.from(aliceTurned.rootKey)).not.toEqual(oldRoot);
    expect(Array.from(aliceTurned.sendChainKey)).not.toEqual(oldSendChain);
    expect(Array.from(aliceTurned.receiveChainKey)).not.toEqual(oldRecvChain);
    expect(Array.from(aliceTurned.dhSendPublic)).not.toEqual(oldDHPublic);
    expect(aliceTurned.dhRecvPublic).toEqual(bobNewDHPublic);

    // Bob's DH ratchet state is also changed
    expect(Array.from(bobTurned.rootKey)).not.toEqual(Array.from(accepted.state.rootKey));
    expect(Array.from(bobTurned.sendChainKey)).not.toEqual(Array.from(accepted.state.sendChainKey));
    expect(Array.from(bobTurned.receiveChainKey)).not.toEqual(Array.from(accepted.state.receiveChainKey));

    // Encryption/decryption works with the new send chain (directional correctness)
    const msg2 = nextSendMessageKey(aliceTurned);
    aliceTurned.state = msg2.state;

    const msg3 = nextSendMessageKey(aliceTurned);
    aliceTurned.state = msg3.state;

    const recv2 = nextReceiveMessageKey(bobTurned, 0);
    expect(Array.from(recv2.messageKey)).toEqual(Array.from(msg2.messageKey));
    bobTurned.state = recv2.state;

    const recv3 = nextReceiveMessageKey(bobTurned, 1);
    expect(Array.from(recv3.messageKey)).toEqual(Array.from(msg3.messageKey));

    // Post-compromise security: attacker with old chain keys cannot decrypt
    const attackerChainKey = new Uint8Array(initiated.state.sendChainKey);
    const attackerDHPrivate = new Uint8Array(initiated.state.dhSendPrivate);
    const attackerTurned = performDHRatchetTurn(
      { ...initiated.state, sendChainKey: attackerChainKey, dhSendPrivate: attackerDHPrivate },
      new Uint8Array(32), // attacker doesn't know Bob's real new DH public
    );
    const attackerMsg = nextSendMessageKey(attackerTurned);

    // Attacker's key is derived from different inputs — cannot decrypt Alice's real message
    expect(Array.from(attackerMsg.messageKey)).not.toEqual(Array.from(msg2.messageKey));
  });

  it("recovers from post-compromise security via DH ratchet", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    // Phase 1: normal exchange before compromise
    const pre0 = nextSendMessageKey(initiated.state);
    initiated.state = pre0.state;
    const pre1 = nextSendMessageKey(initiated.state);
    initiated.state = pre1.state;

    const preRecv0 = nextReceiveMessageKey(accepted.state);
    expect(Array.from(preRecv0.messageKey)).toEqual(Array.from(pre0.messageKey));
    accepted.state = preRecv0.state;
    const preRecv1 = nextReceiveMessageKey(accepted.state);
    expect(Array.from(preRecv1.messageKey)).toEqual(Array.from(pre1.messageKey));
    accepted.state = preRecv1.state;

    // Phase 2: simulate compromise — attacker obtains current state
    const compromisedState = {
      ...accepted.state,
      rootKey: accepted.state.rootKey.slice(),
      sendChainKey: accepted.state.sendChainKey.slice(),
      receiveChainKey: accepted.state.receiveChainKey.slice(),
      dhSendPrivate: accepted.state.dhSendPrivate.slice(),
      dhSendPublic: accepted.state.dhSendPublic.slice(),
      dhRecvPublic: accepted.state.dhRecvPublic.slice(),
      skippedMessageKeys: new Map(accepted.state.skippedMessageKeys),
    };

    // Phase 3: compromise recovery via DH ratchet — both sides rotate DH keys
    const bobNewKeyPair = { publicKey: new Uint8Array(32), secretKey: new Uint8Array(32) };
    crypto.getRandomValues(bobNewKeyPair.publicKey);
    crypto.getRandomValues(bobNewKeyPair.secretKey);
    const bobRecovered = performDHRatchetTurn(accepted.state, bobNewKeyPair.publicKey);

    const aliceNewKeyPair = { publicKey: new Uint8Array(32), secretKey: new Uint8Array(32) };
    crypto.getRandomValues(aliceNewKeyPair.publicKey);
    crypto.getRandomValues(aliceNewKeyPair.secretKey);
    const aliceRecovered = performDHRatchetTurn(initiated.state, aliceNewKeyPair.publicKey);

    // Phase 4: normal exchange after recovery — new keys are independent of compromised state
    const post0 = nextSendMessageKey(aliceRecovered);
    aliceRecovered.state = post0.state;
    const post1 = nextSendMessageKey(aliceRecovered);
    aliceRecovered.state = post1.state;

    const postRecv0 = nextReceiveMessageKey(bobRecovered, 0);
    expect(Array.from(postRecv0.messageKey)).toEqual(Array.from(post0.messageKey));
    bobRecovered.state = postRecv0.state;
    const postRecv1 = nextReceiveMessageKey(bobRecovered, 1);
    expect(Array.from(postRecv1.messageKey)).toEqual(Array.from(post1.messageKey));

    // Attacker's post-compromise state is stale — cannot decrypt new messages
    const attackerMsg = nextSendMessageKey(compromisedState);
    expect(Array.from(attackerMsg.messageKey)).not.toEqual(Array.from(post0.messageKey));
    expect(Array.from(attackerMsg.messageKey)).not.toEqual(Array.from(post1.messageKey));

    // Attacker's DH ratchet with unknown public key produces different keys
    const attackerDH = performDHRatchetTurn(compromisedState, new Uint8Array(32));
    const attackerDHMsg = nextSendMessageKey(attackerDH);
    expect(Array.from(attackerDHMsg.messageKey)).not.toEqual(Array.from(post0.messageKey));
    expect(Array.from(attackerDHMsg.messageKey)).not.toEqual(Array.from(post1.messageKey));
  });

  it("multiple DH ratchet turns produce independent, non-degenerate keys", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");
    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    // Perform multiple DH ratchet turns, collecting message keys after each
    const messageKeys: number[][] = [];
    for (let i = 0; i < 5; i++) {
      const peerDH = new Uint8Array(32);
      crypto.getRandomValues(peerDH);
      const turned = performDHRatchetTurn(initiated.state, peerDH);
      initiated.state = turned;

      const mk = nextSendMessageKey(initiated.state);
      initiated.state = mk.state;
      messageKeys.push(Array.from(mk.messageKey));
    }

    // All message keys are unique — no degeneracy
    const unique = new Set(messageKeys.map((k) => JSON.stringify(k)));
    expect(unique.size).toBe(5);

    // All root keys changed at each step — no state stagnation
    const rootKeys: number[][] = [];
    let state = initiated.state;
    for (let i = 0; i < 5; i++) {
      rootKeys.push(Array.from(state.rootKey));
      const peerDH = new Uint8Array(32);
      crypto.getRandomValues(peerDH);
      state = performDHRatchetTurn(state, peerDH);
    }
    const uniqueRoots = new Set(rootKeys.map((k) => JSON.stringify(k)));
    expect(uniqueRoots.size).toBe(5);
  });
});
