import { describe, expect, it, beforeEach } from "vitest";
import {
  createDeviceKeyBundle,
  publicBundleContainsNoPrivateKeys,
  toRegisterDeviceRequest,
  verifySignedPreKey,
} from "./deviceKeys";
import {
  canUseInsecureLocalKeyStorage,
  deleteDeviceKeyBundle,
  getDeviceKeyStorageStatus,
  isTauriRuntime,
  loadDeviceKeyBundle,
  parseStoredDeviceKeyBundle,
  setDeviceKeyStorageAdapterForTests,
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
    expect(bundle.identityDHKey.publicKey).toBeTruthy();
    expect(bundle.identityDHKey.secretKey).toBeTruthy();
    expect(bundle.signedPreKey.publicKey).toBeTruthy();
    expect(bundle.signedPreKey.secretKey).toBeTruthy();
    expect(bundle.oneTimePreKeys).toHaveLength(3);
    expect(verifySignedPreKey(bundle)).toBe(true);
  });

  it("maps to backend registration request without private keys", () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 2);
    const request = toRegisterDeviceRequest(bundle);
    expect(request.identity_public_key).toBe(bundle.identitySigningKey.publicKey);
    expect(request.identity_dh_public_key).toBe(bundle.identityDHKey.publicKey);
    expect(request.signed_pre_key).toBe(bundle.signedPreKey.publicKey);
    expect(request.signed_pre_key_signature).toBe(bundle.signedPreKeySignature);
    expect(request.one_time_prekeys).toHaveLength(2);
    expect(JSON.stringify(request)).not.toContain(bundle.identitySigningKey.secretKey);
    expect(JSON.stringify(request)).not.toContain(bundle.identityDHKey.secretKey);
    expect(JSON.stringify(request)).not.toContain(bundle.signedPreKey.secretKey);
    expect(publicBundleContainsNoPrivateKeys(request)).toBe(true);
  });
});

describe("device key storage fallback", () => {
  beforeEach(() => {
    localStorage.clear();
    delete (globalThis as typeof globalThis & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__;
    setDeviceKeyStorageAdapterForTests(null);
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

  it("rejects stored bundles from another user context", () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    const parsed = parseStoredDeviceKeyBundle(JSON.stringify(bundle), "user-2");
    expect(parsed).toBeNull();
  });

  it("rejects malformed stored bundles", () => {
    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    const malformed = {
      ...bundle,
      identitySigningKey: { ...bundle.identitySigningKey, secretKey: undefined },
    };
    const parsed = parseStoredDeviceKeyBundle(JSON.stringify(malformed), "user-1");
    expect(parsed).toBeNull();
  });

  it("migrates an alpha v1 bundle by generating a new X3DH identity key", () => {
    const current = createDeviceKeyBundle("user-1", "test device", 1);
    const legacy = JSON.parse(JSON.stringify(current)) as Record<string, unknown>;
    legacy.version = 1;
    legacy.registeredAt = "2026-01-01T00:00:00.000Z";
    legacy.backendFingerprint = "old-fingerprint";
    delete legacy.identityDHKey;

    const migrated = parseStoredDeviceKeyBundle(JSON.stringify(legacy), "user-1");
    expect(migrated?.version).toBe(2);
    expect(migrated?.identityDHKey.publicKey).toBeTruthy();
    expect(migrated?.registeredAt).toBeUndefined();
    expect(migrated?.backendFingerprint).toBeUndefined();
  });

  it("documents production web localStorage refusal", () => {
    expect(canUseInsecureLocalKeyStorage({ PROD: true })).toBe(false);
    expect(
      canUseInsecureLocalKeyStorage({
        PROD: true,
        VITE_ALLOW_INSECURE_KEY_STORAGE: "true",
      }),
    ).toBe(true);
    expect(canUseInsecureLocalKeyStorage({ PROD: false })).toBe(true);
  });

  it("reports localStorage fallback as insecure storage", () => {
    const status = getDeviceKeyStorageStatus();
    expect(status.backend).toBe("insecure-localstorage");
    expect(status.available).toBe(true);
    expect(status.productionSafe).toBe(false);
    expect(status.warning).toContain("localStorage");
  });

  it("detects the default Tauri 2 runtime marker", () => {
    Object.defineProperty(globalThis, "__TAURI_INTERNALS__", {
      configurable: true,
      value: {},
    });
    expect(isTauriRuntime()).toBe(true);
  });

  it("uses injected Tauri keychain adapter before localStorage", async () => {
    const stored = new Map<string, string>();
    setDeviceKeyStorageAdapterForTests({
      kind: "tauri-keychain",
      productionSafe: true,
      warning: null,
      async store(bundle) {
        stored.set(bundle.userId, JSON.stringify(bundle));
      },
      async load(userId) {
        return stored.get(userId) ?? null;
      },
      async delete(userId) {
        stored.delete(userId);
      },
    });

    const status = getDeviceKeyStorageStatus();
    expect(status.backend).toBe("tauri-keychain");
    expect(status.productionSafe).toBe(true);

    const bundle = createDeviceKeyBundle("user-1", "test device", 1);
    await storeDeviceKeyBundle(bundle);
    expect(localStorage.length).toBe(0);
    await expect(loadDeviceKeyBundle("user-1")).resolves.toMatchObject({
      userId: "user-1",
      deviceId: bundle.deviceId,
    });
    await deleteDeviceKeyBundle("user-1");
    await expect(loadDeviceKeyBundle("user-1")).resolves.toBeNull();
  });
});
