import { beforeEach, describe, expect, it } from "vitest";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import {
  clearDeviceVerification,
  getDeviceVerificationStatus,
  loadDeviceVerificationRecord,
  verifyDeviceSafetyNumber,
} from "./deviceVerificationStore";
import type { DeviceAPI } from "./types";

describe("device verification store", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("starts devices as unverified", async () => {
    const local = createDeviceKeyBundle("user-a", "desktop", 0);
    const remote = makeRemoteDevice("user-b");

    const status = await getDeviceVerificationStatus(local, remote);

    expect(status.state).toBe("unverified");
    expect(status.record).toBeNull();
    expect(status.safetyNumber.value).toMatch(/^\d{5}( \d{5}){11}$/);
  });

  it("stores a verified safety number for a specific device pair", async () => {
    const local = createDeviceKeyBundle("user-a", "desktop", 0);
    const remote = makeRemoteDevice("user-b");

    const record = await verifyDeviceSafetyNumber(local, remote);
    const status = await getDeviceVerificationStatus(local, remote);

    expect(status.state).toBe("verified");
    expect(status.record).toEqual(record);
    expect(loadDeviceVerificationRecord(local, remote)).toEqual(record);
  });

  it("detects remote identity key changes after verification", async () => {
    const local = createDeviceKeyBundle("user-a", "desktop", 0);
    const remote = makeRemoteDevice("user-b");
    await verifyDeviceSafetyNumber(local, remote);

    const changedRemote = {
      ...remote,
      identity_public_key: "rotated-identity-key",
    };
    const status = await getDeviceVerificationStatus(local, changedRemote);

    expect(status.state).toBe("identity_changed");
    expect(status.record?.remoteIdentityPublicKey).toBe(
      remote.identity_public_key,
    );
  });

  it("clears local verification records", async () => {
    const local = createDeviceKeyBundle("user-a", "desktop", 0);
    const remote = makeRemoteDevice("user-b");
    await verifyDeviceSafetyNumber(local, remote);

    clearDeviceVerification(local, remote);

    await expect(getDeviceVerificationStatus(local, remote)).resolves.toMatchObject({
      state: "unverified",
      record: null,
    });
  });
});

function makeRemoteDevice(userId: string): DeviceAPI {
  const request = toRegisterDeviceRequest(
    createDeviceKeyBundle(userId, "laptop", 0),
  );
  return {
    device_id: request.device_id,
    user_id: userId,
    name: request.name,
    identity_public_key: request.identity_public_key,
    identity_dh_public_key: request.identity_dh_public_key,
    signed_pre_key_id: request.signed_pre_key_id,
    signed_pre_key: request.signed_pre_key,
    signed_pre_key_signature: request.signed_pre_key_signature,
    fingerprint: "fingerprint",
    trust_state: "unverified",
    one_time_prekey_count: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    last_seen_at: new Date().toISOString(),
  };
}
