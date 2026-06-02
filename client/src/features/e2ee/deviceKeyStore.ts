import type { StoredDeviceKeyBundle } from "./types";

const STORAGE_PREFIX = "zenthril_e2ee_device_bundle:";

type EnvLike = {
  PROD?: boolean;
  VITE_ALLOW_INSECURE_KEY_STORAGE?: string;
};

export type DeviceKeyStorageBackend =
  | "tauri-store"
  | "insecure-localstorage"
  | "unavailable";

export interface DeviceKeyStorageStatus {
  backend: DeviceKeyStorageBackend;
  available: boolean;
  productionSafe: boolean;
  warning: string | null;
}

export function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

export function canUseInsecureLocalKeyStorage(env: EnvLike = import.meta.env): boolean {
  return !env.PROD || env.VITE_ALLOW_INSECURE_KEY_STORAGE === "true";
}

export function getDeviceKeyStorageStatus(): DeviceKeyStorageStatus {
  if (isTauriRuntime()) {
    return {
      backend: "tauri-store",
      available: true,
      productionSafe: false,
      warning:
        "Tauri Store is temporary desktop storage, not OS keychain or Stronghold.",
    };
  }

  if (canUseInsecureLocalKeyStorage()) {
    return {
      backend: "insecure-localstorage",
      available: true,
      productionSafe: false,
      warning:
        "localStorage may expose private E2EE keys to XSS or local profile compromise.",
    };
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
  const serialized = JSON.stringify(bundle);
  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("store_device_key_bundle", {
      userId: bundle.userId,
      bundleJson: serialized,
    });
    return;
  }
  if (!canUseInsecureLocalKeyStorage()) {
    // SECURITY-HARDENING: production web builds must not persist E2EE private material in localStorage.
    throw new Error("Secure device key storage is unavailable");
  }
  localStorage.setItem(storageKey(bundle.userId), serialized);
}

export async function loadDeviceKeyBundle(
  userId: string,
): Promise<StoredDeviceKeyBundle | null> {
  const raw = isTauriRuntime()
    ? await loadFromTauri(userId)
    : canUseInsecureLocalKeyStorage()
      ? localStorage.getItem(storageKey(userId))
      : null;
  if (!raw) return null;
  return parseStoredDeviceKeyBundle(raw, userId);
}

export async function deleteDeviceKeyBundle(userId: string): Promise<void> {
  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("delete_device_key_bundle", { userId });
    return;
  }
  if (!canUseInsecureLocalKeyStorage()) {
    return;
  }
  localStorage.removeItem(storageKey(userId));
}

export function parseStoredDeviceKeyBundle(
  raw: string,
  expectedUserId: string,
): StoredDeviceKeyBundle | null {
  try {
    const parsed = JSON.parse(raw) as Partial<StoredDeviceKeyBundle>;
    // SECURITY-HARDENING: never load private device keys for a different user context.
    if (
      parsed.version !== 1 ||
      parsed.userId !== expectedUserId ||
      typeof parsed.deviceId !== "string" ||
      typeof parsed.identitySigningKey?.secretKey !== "string" ||
      typeof parsed.signedPreKey?.secretKey !== "string"
    ) {
      return null;
    }
    return parsed as StoredDeviceKeyBundle;
  } catch {
    return null;
  }
}

async function loadFromTauri(userId: string): Promise<string | null> {
  const { invoke } = await import("@tauri-apps/api/core");
  return invoke<string | null>("load_device_key_bundle", { userId });
}

function validateBundleForStorage(bundle: StoredDeviceKeyBundle): void {
  // SECURITY-HARDENING: reject malformed local private-key bundles before persistence.
  if (
    bundle.version !== 1 ||
    !bundle.userId ||
    !bundle.deviceId ||
    !bundle.identitySigningKey?.secretKey ||
    !bundle.signedPreKey?.secretKey
  ) {
    throw new Error("Invalid device key bundle");
  }
}

function storageKey(userId: string): string {
  return `${STORAGE_PREFIX}${userId}`;
}
