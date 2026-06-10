import { beforeEach, describe, expect, it } from "vitest";
import { createDeviceKeyBundle } from "./deviceKeys";
import {
  buildMessageAADInput,
  MissingDeviceKeyBundleError,
} from "./messageAAD";
import {
  deleteDeviceKeyBundle,
  setDeviceKeyStorageAdapterForTests,
  storeDeviceKeyBundle,
} from "./deviceKeyStore";

describe("message AAD context", () => {
  beforeEach(() => {
    localStorage.clear();
    setDeviceKeyStorageAdapterForTests(null);
  });

  it("binds new message AAD to the registered local device id", async () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    await storeDeviceKeyBundle(bundle);

    const aad = await buildMessageAADInput("channel-1", "user-1");

    expect(aad).toMatchObject({
      channelId: "channel-1",
      senderUserId: "user-1",
      senderDeviceId: bundle.deviceId,
      sessionId: "channel:channel-1",
    });
    expect(aad.clientMessageId).toBeTruthy();
    expect(aad.senderDeviceId).not.toBe("unregistered-device");
  });

  it("fails closed when no local device bundle is available", async () => {
    await deleteDeviceKeyBundle("user-1");

    await expect(buildMessageAADInput("channel-1", "user-1")).rejects.toBeInstanceOf(
      MissingDeviceKeyBundleError,
    );
  });
});
