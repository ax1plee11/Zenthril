import { describe, expect, it } from "vitest";
import {
  createSafetyNumber,
  createSafetyNumberInput,
  createSafetyNumberQRPayload,
  formatSafetyNumber,
  parseSafetyNumberQRPayload,
} from "./safetyNumber";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import type { DeviceAPI } from "./types";

describe("safety number", () => {
  it("is stable when local and remote participants are swapped", async () => {
    const local = {
      userId: "user-a",
      deviceId: "device-a",
      identityPublicKey: "identity-a",
    };
    const remote = {
      userId: "user-b",
      deviceId: "device-b",
      identityPublicKey: "identity-b",
    };

    const first = await createSafetyNumber({ local, remote });
    const second = await createSafetyNumber({ local: remote, remote: local });

    expect(first.value).toBe(second.value);
    expect(first.value).toMatch(/^\d{5}( \d{5}){11}$/);
  });

  it("changes when a participant identity key changes", async () => {
    const local = {
      userId: "user-a",
      deviceId: "device-a",
      identityPublicKey: "identity-a",
    };
    const remote = {
      userId: "user-b",
      deviceId: "device-b",
      identityPublicKey: "identity-b",
    };

    const first = await createSafetyNumber({ local, remote });
    const second = await createSafetyNumber({
      local,
      remote: { ...remote, identityPublicKey: "identity-b-rotated" },
    });

    expect(first.value).not.toBe(second.value);
  });

  it("formats digest bytes into 12 readable groups", () => {
    const digest = new Uint8Array(Array.from({ length: 32 }, (_, i) => i));

    expect(formatSafetyNumber(digest)).toBe(
      "00000 10020 03004 00500 60070 08009 01001 10120 13014 01501 60170 18019",
    );
  });

  it("builds input from local and remote device records", () => {
    const local = createDeviceKeyBundle("user-a", "desktop", 0);
    const request = toRegisterDeviceRequest(
      createDeviceKeyBundle("user-b", "laptop", 0),
    );
    const remote: DeviceAPI = {
      device_id: request.device_id,
      user_id: "user-b",
      name: request.name,
      identity_public_key: request.identity_public_key,
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

    expect(createSafetyNumberInput(local, remote)).toEqual({
      local: {
        userId: local.userId,
        deviceId: local.deviceId,
        identityPublicKey: local.identitySigningKey.publicKey,
      },
      remote: {
        userId: remote.user_id,
        deviceId: remote.device_id,
        identityPublicKey: remote.identity_public_key,
      },
    });
  });

  it("round-trips QR verification payloads", async () => {
    const safetyNumber = await createSafetyNumber({
      local: {
        userId: "user-a",
        deviceId: "device-a",
        identityPublicKey: "identity-a",
      },
      remote: {
        userId: "user-b",
        deviceId: "device-b",
        identityPublicKey: "identity-b",
      },
    });

    const payload = createSafetyNumberQRPayload(safetyNumber);

    expect(parseSafetyNumberQRPayload(payload)).toEqual({
      type: "zenthril.safety-number",
      version: 1,
      safetyNumber: safetyNumber.value,
      participants: safetyNumber.participants,
    });
  });
});
