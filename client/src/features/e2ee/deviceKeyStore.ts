import { x25519 } from "@noble/curves/ed25519.js";
import { bytesToBase64 } from "./encoding";
import type { StoredDeviceKeyBundle } from "./types";

const STORAGE_PREFIX = "zenthril_e2ee_device_bundle:";

type EnvLike = {
  PROD?: boolean;
  VITE_ALLOW_INSECURE_KEY_STORAGE?: string;
};

export type DeviceKeyStorageBackend =
  | "tauri-keychain"
  | "insecure-localstorage"
  | "unavailable";

export interface DeviceKeyStorageStatus {
  backend: DeviceKeyStorageBackend;
  available: boolean;
  productionSafe: boolean;
  warning: string | null;
}

export interface DeviceKeyStorageAdapter {
  kind: DeviceKeyStorageBackend;
  productionSafe: boolean;
  warning: string | null;
  store(bundle: StoredDeviceKeyBundle): Promise<void>;
  load(userId: string): Promise<string | null>;
  delete(userId: string): Promise<void>;
}

type InvokeFn = <T = unknown>(command: string, args?: Record<string, unknown>) => Promise<T>;
type ParsedStoredBundle = Omit<Partial<StoredDeviceKeyBundle>, "version"> & {
  version?: number;
};

let testAdapter: DeviceKeyStorageAdapter | null = null;

export function setDeviceKeyStorageAdapterForTests(
  adapter: DeviceKeyStorageAdapter | null,
): void {
  testAdapter = adapter;
}

export function isTauriRuntime(): boolean {
  // SECURITY: Tauri 2 exposes __TAURI_INTERNALS__ by default. __TAURI__ is
  // optional and relying on it can silently select an insecure web fallback.
  return typeof globalThis !== "undefined" && "__TAURI_INTERNALS__" in globalThis;
}

export function canUseInsecureLocalKeyStorage(env: EnvLike = import.meta.env): boolean {
  return !env.PROD || env.VITE_ALLOW_INSECURE_KEY_STORAGE === "true";
}

export function getDeviceKeyStorageStatus(): DeviceKeyStorageStatus {
  const adapter = resolveStorageAdapter();
  if (adapter) return adapterStatus(adapter);

  if (canUseInsecureLocalKeyStorage()) {
    return adapterStatus(localStorageAdapter());
  }

  return {
    backend: "unavailable",
    available: false,
    productionSafe: false,
    warning:
      "Secure device key storage is unavailable in this production web build.",
  };
}

export async function storeDeviceKeyBundle(
  bundle: StoredDeviceKeyBundle,
): Promise<void> {
  validateBundleForStorage(bundle);
  const adapter = resolveStorageAdapter();
  if (adapter) {
    await adapter.store(bundle);
    return;
  }
  if (!canUseInsecureLocalKeyStorage()) {
    // SECURITY-HARDENING: production web builds must not persist E2EE private material in localStorage.
    throw new Error("Secure device key storage is unavailable");
  }
  await localStorageAdapter().store(bundle);
}

export async function loadDeviceKeyBundle(
  userId: string,
): Promise<StoredDeviceKeyBundle | null> {
  const adapter = resolveStorageAdapter();
  const raw = adapter
    ? await adapter.load(userId)
    : canUseInsecureLocalKeyStorage()
      ? await localStorageAdapter().load(userId)
      : null;
  if (!raw) return null;
  return parseStoredDeviceKeyBundle(raw, userId);
}

export async function deleteDeviceKeyBundle(userId: string): Promise<void> {
  const adapter = resolveStorageAdapter();
  if (adapter) {
    await adapter.delete(userId);
    return;
  }
  if (!canUseInsecureLocalKeyStorage()) {
    return;
  }
  await localStorageAdapter().delete(userId);
}

export function parseStoredDeviceKeyBundle(
  raw: string,
  expectedUserId: string,
): StoredDeviceKeyBundle | null {
  try {
    const parsed = JSON.parse(raw) as ParsedStoredBundle;
    // SECURITY-HARDENING: never load private device keys for a different user context.
    if (parsed.version === 1) {
      return migrateLegacyBundle(parsed, expectedUserId);
    }
    if (
      parsed.version !== 2 ||
      parsed.userId !== expectedUserId ||
      typeof parsed.deviceId !== "string" ||
      typeof parsed.identitySigningKey?.secretKey !== "string" ||
      typeof parsed.identityDHKey?.secretKey !== "string" ||
      typeof parsed.signedPreKey?.secretKey !== "string"
    ) {
      return null;
    }
    return parsed as StoredDeviceKeyBundle;
  } catch {
    return null;
  }
}

function adapterStatus(adapter: DeviceKeyStorageAdapter): DeviceKeyStorageStatus {
  return {
    backend: adapter.kind,
    available: true,
    productionSafe: adapter.productionSafe,
    warning: adapter.warning,
  };
}

function resolveStorageAdapter(): DeviceKeyStorageAdapter | null {
  if (testAdapter) return testAdapter;
  if (isTauriRuntime()) return tauriKeychainAdapter();
  return null;
}

function tauriKeychainAdapter(): DeviceKeyStorageAdapter {
  return {
    kind: "tauri-keychain",
    productionSafe: true,
    warning: null,
    async store(bundle) {
      await invokeTauri("store_device_key_bundle", {
        userId: bundle.userId,
        bundleJson: JSON.stringify(bundle),
      });
    },
    async load(userId) {
      return invokeTauri<string | null>("load_device_key_bundle", { userId });
    },
    async delete(userId) {
      await invokeTauri("delete_device_key_bundle", { userId });
    },
  };
}

function localStorageAdapter(): DeviceKeyStorageAdapter {
  return {
    kind: "insecure-localstorage",
    productionSafe: false,
    warning:
      "localStorage may expose private E2EE keys to XSS or local profile compromise.",
    async store(bundle) {
      localStorage.setItem(storageKey(bundle.userId), JSON.stringify(bundle));
    },
    async load(userId) {
      return localStorage.getItem(storageKey(userId));
    },
    async delete(userId) {
      localStorage.removeItem(storageKey(userId));
    },
  };
}

async function invokeTauri<T = unknown>(
  command: string,
  args?: Record<string, unknown>,
): Promise<T> {
  const { invoke } = await import("@tauri-apps/api/core");
  return (invoke as InvokeFn)<T>(command, args);
}

function validateBundleForStorage(bundle: StoredDeviceKeyBundle): void {
  // SECURITY-HARDENING: reject malformed local private-key bundles before persistence.
  if (
    bundle.version !== 2 ||
    !bundle.userId ||
    !bundle.deviceId ||
    !bundle.identitySigningKey?.secretKey ||
    !bundle.identityDHKey?.secretKey ||
    !bundle.signedPreKey?.secretKey
  ) {
    throw new Error("Invalid device key bundle");
  }
}

function migrateLegacyBundle(
  parsed: ParsedStoredBundle,
  expectedUserId: string,
): StoredDeviceKeyBundle | null {
  if (
    parsed.userId !== expectedUserId ||
    typeof parsed.deviceId !== "string" ||
    typeof parsed.deviceName !== "string" ||
    typeof parsed.identitySigningKey?.publicKey !== "string" ||
    typeof parsed.identitySigningKey.secretKey !== "string" ||
    typeof parsed.signedPreKey?.publicKey !== "string" ||
    typeof parsed.signedPreKey.secretKey !== "string" ||
    typeof parsed.signedPreKeyId !== "number" ||
    typeof parsed.signedPreKeySignature !== "string" ||
    !Array.isArray(parsed.oneTimePreKeys) ||
    typeof parsed.createdAt !== "string"
  ) {
    return null;
  }

  // SECURITY: a v1 bundle has no valid X25519 identity-DH key. Generate one
  // locally and clear registration markers so the public bundle is published
  // again before it can be used for X3DH.
  const identityDHKey = x25519.keygen();
  const migrated = {
    ...(parsed as Omit<StoredDeviceKeyBundle, "version" | "identityDHKey">),
    version: 2,
    identityDHKey: {
      publicKey: bytesToBase64(identityDHKey.publicKey),
      secretKey: bytesToBase64(identityDHKey.secretKey),
    },
  } as StoredDeviceKeyBundle;
  delete (migrated as Partial<StoredDeviceKeyBundle>).registeredAt;
  delete (migrated as Partial<StoredDeviceKeyBundle>).backendFingerprint;
  return migrated;
}

function storageKey(userId: string): string {
  return `${STORAGE_PREFIX}${userId}`;
}
