import { describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  acceptPairwiseSession,
  initiatePairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
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

describe("chain key tracing", () => {
  it("traces chain keys through send/receive/skip", () => {
    const alice = publicKeyBundle("alice", "Alice device");
    const bob = publicKeyBundle("bob", "Bob device");

    const initiated = initiatePairwiseSession(alice.local, bob.publicBundle);
    const accepted = acceptPairwiseSession(bob.local, initiated.header);

    console.log("Alice initial sendChain:", Array.from(initiated.state.sendChainKey).slice(0, 8));
    console.log("Bob initial receiveChain:", Array.from(accepted.state.receiveChainKey).slice(0, 8));
    expect(Array.from(initiated.state.sendChainKey)).toEqual(Array.from(accepted.state.receiveChainKey));

    const msg0 = nextSendMessageKey(initiated.state);
    initiated.state = msg0.state;
    console.log("After msg0 - Alice sendChain:", Array.from(initiated.state.sendChainKey).slice(0, 8));
    console.log("msg0 key:", Array.from(msg0.messageKey).slice(0, 8));

    const recv0 = nextReceiveMessageKey(accepted.state);
    accepted.state = recv0.state;
    console.log("After recv0 - Bob receiveChain:", Array.from(accepted.state.receiveChainKey).slice(0, 8));
    console.log("recv0 key:", Array.from(recv0.messageKey).slice(0, 8));
    expect(Array.from(msg0.messageKey)).toEqual(Array.from(recv0.messageKey));

    const msg1 = nextSendMessageKey(initiated.state);
    initiated.state = msg1.state;
    console.log("After msg1 - Alice sendChain:", Array.from(initiated.state.sendChainKey).slice(0, 8));
    console.log("msg1 key:", Array.from(msg1.messageKey).slice(0, 8));

    const msg2 = nextSendMessageKey(initiated.state);
    initiated.state = msg2.state;
    console.log("After msg2 - Alice sendChain:", Array.from(initiated.state.sendChainKey).slice(0, 8));
    console.log("msg2 key:", Array.from(msg2.messageKey).slice(0, 8));

    const recv2 = nextReceiveMessageKey(accepted.state, 2);
    accepted.state = recv2.state;
    console.log("After recv2 - Bob receiveChain:", Array.from(accepted.state.receiveChainKey).slice(0, 8));
    console.log("recv2 key:", Array.from(recv2.messageKey).slice(0, 8));
    console.log("skipped keys after recv2:", Array.from(accepted.state.skippedMessageKeys.keys()));
    expect(Array.from(msg2.messageKey)).toEqual(Array.from(recv2.messageKey));

    // msg0 was already consumed normally, so only msg1 is in skipped keys
    expect(accepted.state.skippedMessageKeys.size).toBe(1);
    expect(accepted.state.skippedMessageKeys.has(1)).toBe(true);

    const recv1_skipped = nextReceiveMessageKey(accepted.state, 1);
    console.log("recv1_skipped key:", Array.from(recv1_skipped.messageKey).slice(0, 8));
    expect(Array.from(msg1.messageKey)).toEqual(Array.from(recv1_skipped.messageKey));
    expect(accepted.state.skippedMessageKeys.size).toBe(0);
  });
});
