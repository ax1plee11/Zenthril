import { describe, expect, it, vi } from "vitest";
import { DirectMessagingPeer } from "./directMessaging";

describe("DirectMessagingPeer", () => {
  it("throws when sending before data channel is open", () => {
    const peer = new DirectMessagingPeer({ localDeviceId: "device-a" });

    expect(() => peer.send("ciphertext")).toThrow("P2P direct channel is not open");
  });

  it("emits closed state on close", () => {
    const onStateChange = vi.fn();
    const peer = new DirectMessagingPeer({ localDeviceId: "device-a", onStateChange });

    peer.close();

    expect(onStateChange).toHaveBeenCalledWith("closed");
  });
});
