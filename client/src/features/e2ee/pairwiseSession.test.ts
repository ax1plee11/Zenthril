import { describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  loadPairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
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
