import { describe, expect, it, beforeEach } from "vitest";
import {
  createDeviceKeyBundle,
  publicBundleContainsNoPrivateKeys,
  toRegisterDeviceRequest,
  verifySignedPreKey,
} from "./deviceKeys";
import {
  deleteDeviceKeyBundle,
  loadDeviceKeyBundle,
  storeDeviceKeyBundle,
} from "./deviceKeyStore";

describe("device key bundle", () => {
  it("generates a signed public device bundle", () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 3);
    expect(bundle.deviceId).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );
    expect(bundle.identitySigningKey.publicKey).toBeTruthy();
    expect(bundle.identitySigningKey.secretKey).toBeTruthy();
    expect(bundle.signedPreKey.publicKey).toBeTruthy();
    expect(bundle.signedPreKey.secretKey).toBeTruthy();
    expect(bundle.oneTimePreKeys).toHaveLength(3);
    expect(verifySignedPreKey(bundle)).toBe(true);
  });

  it("maps to backend registration request without private keys", () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 2);
    const request = toRegisterDeviceRequest(bundle);
    expect(request.identity_public_key).toBe(bundle.identitySigningKey.publicKey);
    expect(request.signed_pre_key).toBe(bundle.signedPreKey.publicKey);
    expect(request.signed_pre_key_signature).toBe(bundle.signedPreKeySignature);
    expect(request.one_time_prekeys).toHaveLength(2);
    expect(JSON.stringify(request)).not.toContain(bundle.identitySigningKey.secretKey);
    expect(JSON.stringify(request)).not.toContain(bundle.signedPreKey.secretKey);
    expect(publicBundleContainsNoPrivateKeys(request)).toBe(true);
  });
});

describe("device key storage fallback", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores and loads a local device bundle", async () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    await storeDeviceKeyBundle(bundle);
    const loaded = await loadDeviceKeyBundle("user-1");
    expect(loaded).not.toBeNull();
    expect(loaded?.deviceId).toBe(bundle.deviceId);
    expect(loaded?.identitySigningKey.secretKey).toBe(
      bundle.identitySigningKey.secretKey,
    );
  });

  it("deletes a local device bundle", async () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    await storeDeviceKeyBundle(bundle);
    await deleteDeviceKeyBundle("user-1");
    await expect(loadDeviceKeyBundle("user-1")).resolves.toBeNull();
  });
});
