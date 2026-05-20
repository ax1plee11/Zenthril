import type { StoredDeviceKeyBundle } from "./types";

const STORAGE_PREFIX = "zenthril_e2ee_device_bundle:";

export function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

export async function storeDeviceKeyBundle(
  bundle: StoredDeviceKeyBundle,
): Promise<void> {
  const serialized = JSON.stringify(bundle);
  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("store_device_key_bundle", {
      userId: bundle.userId,
      bundleJson: serialized,
    });
    return;
  }
  localStorage.setItem(storageKey(bundle.userId), serialized);
}

export async function loadDeviceKeyBundle(
  userId: string,
): Promise<StoredDeviceKeyBundle | null> {
  const raw = isTauriRuntime()
    ? await loadFromTauri(userId)
    : localStorage.getItem(storageKey(userId));
  if (!raw) return null;
  try {
    return JSON.parse(raw) as StoredDeviceKeyBundle;
  } catch {
    return null;
  }
}

export async function deleteDeviceKeyBundle(userId: string): Promise<void> {
  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("delete_device_key_bundle", { userId });
    return;
  }
  localStorage.removeItem(storageKey(userId));
}

async function loadFromTauri(userId: string): Promise<string | null> {
  const { invoke } = await import("@tauri-apps/api/core");
  return invoke<string | null>("load_device_key_bundle", { userId });
}

function storageKey(userId: string): string {
  return `${STORAGE_PREFIX}${userId}`;
}
