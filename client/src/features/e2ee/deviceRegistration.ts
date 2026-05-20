import { api } from "../../api";
import { createDeviceKeyBundle, toRegisterDeviceRequest } from "./deviceKeys";
import { loadDeviceKeyBundle, storeDeviceKeyBundle } from "./deviceKeyStore";
import type { StoredDeviceKeyBundle } from "./types";

export async function ensureLocalDeviceRegistered(
  userId: string,
  deviceName?: string,
): Promise<StoredDeviceKeyBundle> {
  const existing = await loadDeviceKeyBundle(userId);
  const bundle = existing ?? createDeviceKeyBundle(userId, deviceName);

  if (!existing) {
    await storeDeviceKeyBundle(bundle);
  }

  if (bundle.registeredAt && bundle.backendFingerprint) {
    return bundle;
  }

  const registered = await api.devices.register(toRegisterDeviceRequest(bundle));
  const updated: StoredDeviceKeyBundle = {
    ...bundle,
    registeredAt: new Date().toISOString(),
    backendFingerprint: registered.fingerprint,
  };
  await storeDeviceKeyBundle(updated);
  return updated;
}

export async function prepareLocalDeviceBundle(
  userId: string,
  deviceName?: string,
): Promise<StoredDeviceKeyBundle> {
  const existing = await loadDeviceKeyBundle(userId);
  if (existing) return existing;
  const bundle = createDeviceKeyBundle(userId, deviceName);
  await storeDeviceKeyBundle(bundle);
  return bundle;
}
