import type { DeviceAPI, KeyBundleAPI, StoredDeviceKeyBundle } from "./types";
import {
  createSafetyNumber,
  createSafetyNumberInput,
  type SafetyNumber,
} from "./safetyNumber";

const VERIFICATION_RECORD_VERSION = 1;
const STORAGE_PREFIX = "zenthril_e2ee_device_verification:";

export type DeviceVerificationState =
  | "unverified"
  | "verified"
  | "identity_changed";

export interface DeviceVerificationRecord {
  version: 1;
  localUserId: string;
  localDeviceId: string;
  remoteUserId: string;
  remoteDeviceId: string;
  remoteIdentityPublicKey: string;
  safetyNumber: string;
  verifiedAt: string;
}

export interface DeviceVerificationStatus {
  state: DeviceVerificationState;
  safetyNumber: SafetyNumber;
  record: DeviceVerificationRecord | null;
}

export async function getDeviceVerificationStatus(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): Promise<DeviceVerificationStatus> {
  const safetyNumber = await createSafetyNumber(
    createSafetyNumberInput(local, remote),
  );
  const record = loadDeviceVerificationRecord(local, remote);
  if (!record) {
    return { state: "unverified", safetyNumber, record: null };
  }

  // SECURITY: a previously verified device must be treated as changed if its
  // identity key or derived safety number no longer matches the local record.
  const identityChanged =
    record.remoteIdentityPublicKey !== remote.identity_public_key ||
    record.safetyNumber !== safetyNumber.value;

  return {
    state: identityChanged ? "identity_changed" : "verified",
    safetyNumber,
    record,
  };
}

export async function verifyDeviceSafetyNumber(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): Promise<DeviceVerificationRecord> {
  const safetyNumber = await createSafetyNumber(
    createSafetyNumberInput(local, remote),
  );
  const record: DeviceVerificationRecord = {
    version: VERIFICATION_RECORD_VERSION,
    localUserId: local.userId,
    localDeviceId: local.deviceId,
    remoteUserId: remote.user_id,
    remoteDeviceId: remote.device_id,
    remoteIdentityPublicKey: remote.identity_public_key,
    safetyNumber: safetyNumber.value,
    verifiedAt: new Date().toISOString(),
  };
  localStorage.setItem(verificationKey(local, remote), JSON.stringify(record));
  return record;
}

export function clearDeviceVerification(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): void {
  localStorage.removeItem(verificationKey(local, remote));
}

export function loadDeviceVerificationRecord(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): DeviceVerificationRecord | null {
  const raw = localStorage.getItem(verificationKey(local, remote));
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<DeviceVerificationRecord>;
    if (
      parsed.version !== VERIFICATION_RECORD_VERSION ||
      parsed.localUserId !== local.userId ||
      parsed.localDeviceId !== local.deviceId ||
      parsed.remoteUserId !== remote.user_id ||
      parsed.remoteDeviceId !== remote.device_id ||
      typeof parsed.remoteIdentityPublicKey !== "string" ||
      typeof parsed.safetyNumber !== "string" ||
      typeof parsed.verifiedAt !== "string"
    ) {
      return null;
    }
    return parsed as DeviceVerificationRecord;
  } catch {
    return null;
  }
}

function verificationKey(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): string {
  return `${STORAGE_PREFIX}${local.userId}:${local.deviceId}:${remote.user_id}:${remote.device_id}`;
}
