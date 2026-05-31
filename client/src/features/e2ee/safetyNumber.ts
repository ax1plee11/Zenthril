import type { DeviceAPI, KeyBundleAPI, StoredDeviceKeyBundle } from "./types";

const SAFETY_NUMBER_VERSION = 1;
const SAFETY_NUMBER_BYTES = 20;
const SAFETY_NUMBER_GROUP_SIZE = 5;

interface ParticipantIdentity {
  userId: string;
  deviceId: string;
  identityPublicKey: string;
}

export interface SafetyNumberInput {
  local: ParticipantIdentity;
  remote: ParticipantIdentity;
}

export interface SafetyNumber {
  version: number;
  value: string;
  participants: [ParticipantIdentity, ParticipantIdentity];
}

export interface SafetyNumberQRPayload {
  type: "zenthril.safety-number";
  version: number;
  safetyNumber: string;
  participants: [ParticipantIdentity, ParticipantIdentity];
}

export async function createSafetyNumber(
  input: SafetyNumberInput,
): Promise<SafetyNumber> {
  const participants = normalizeParticipants(input.local, input.remote);
  const payload = JSON.stringify({
    version: SAFETY_NUMBER_VERSION,
    participants,
  });
  const digest = new Uint8Array(
    await crypto.subtle.digest("SHA-256", new TextEncoder().encode(payload)),
  );

  return {
    version: SAFETY_NUMBER_VERSION,
    value: formatSafetyNumber(digest),
    participants,
  };
}

export function createSafetyNumberInput(
  local: StoredDeviceKeyBundle,
  remote: DeviceAPI | KeyBundleAPI,
): SafetyNumberInput {
  return {
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
  };
}

export function formatSafetyNumber(digest: Uint8Array): string {
  const digits = Array.from(digest.slice(0, SAFETY_NUMBER_BYTES), byte =>
    byte.toString(10).padStart(3, "0"),
  ).join("");
  const groups: string[] = [];
  for (let i = 0; i < digits.length; i += SAFETY_NUMBER_GROUP_SIZE) {
    groups.push(digits.slice(i, i + SAFETY_NUMBER_GROUP_SIZE));
  }
  return groups.join(" ");
}

export function createSafetyNumberQRPayload(value: SafetyNumber): string {
  const payload: SafetyNumberQRPayload = {
    type: "zenthril.safety-number",
    version: value.version,
    safetyNumber: value.value,
    participants: value.participants,
  };
  // SECURITY: this payload is for visual verification only; it is not a trust decision by itself.
  return JSON.stringify(payload);
}

export function parseSafetyNumberQRPayload(value: string): SafetyNumberQRPayload {
  const payload = JSON.parse(value) as Partial<SafetyNumberQRPayload>;
  if (
    payload.type !== "zenthril.safety-number" ||
    payload.version !== SAFETY_NUMBER_VERSION ||
    typeof payload.safetyNumber !== "string" ||
    !Array.isArray(payload.participants) ||
    payload.participants.length !== 2
  ) {
    throw new Error("Invalid safety number QR payload");
  }
  return payload as SafetyNumberQRPayload;
}

function normalizeParticipants(
  a: ParticipantIdentity,
  b: ParticipantIdentity,
): [ParticipantIdentity, ParticipantIdentity] {
  const first = normalizeParticipant(a);
  const second = normalizeParticipant(b);
  const sorted = [first, second].sort((left, right) =>
    participantSortKey(left).localeCompare(participantSortKey(right)),
  );
  return [sorted[0]!, sorted[1]!];
}

function normalizeParticipant(value: ParticipantIdentity): ParticipantIdentity {
  return {
    userId: value.userId.trim(),
    deviceId: value.deviceId.trim(),
    identityPublicKey: value.identityPublicKey.trim(),
  };
}

function participantSortKey(value: ParticipantIdentity): string {
  return `${value.userId}:${value.deviceId}:${value.identityPublicKey}`;
}
