import { ed25519, x25519 } from "@noble/curves/ed25519.js";
import { bytesToBase64, base64ToBytes, randomDeviceID } from "./encoding";
import type {
  RegisterDeviceRequest,
  StoredDeviceKeyBundle,
  StoredOneTimePreKey,
  SerializedKeyPair,
} from "./types";

const DEFAULT_ONE_TIME_PREKEY_COUNT = 20;
const SIGNED_PREKEY_ID = 1;
const SIGNED_PREKEY_CONTEXT = "Zenthril signed prekey v1";

export function createDeviceKeyBundle(
  userId: string,
  deviceName = defaultDeviceName(),
  oneTimePreKeyCount = DEFAULT_ONE_TIME_PREKEY_COUNT,
): StoredDeviceKeyBundle {
  if (!userId) {
    throw new Error("userId is required");
  }
  if (oneTimePreKeyCount < 0 || oneTimePreKeyCount > 100) {
    throw new Error("oneTimePreKeyCount must be between 0 and 100");
  }

  const identitySigningKey = ed25519.keygen();
  const signedPreKey = x25519.keygen();
  const signature = ed25519.sign(
    signedPreKeyMessage(signedPreKey.publicKey),
    identitySigningKey.secretKey,
  );

  return {
    version: 1,
    userId,
    deviceId: randomDeviceID(),
    deviceName,
    identitySigningKey: serializeKeyPair(identitySigningKey),
    signedPreKeyId: SIGNED_PREKEY_ID,
    signedPreKey: serializeKeyPair(signedPreKey),
    signedPreKeySignature: bytesToBase64(signature),
    oneTimePreKeys: createOneTimePreKeys(oneTimePreKeyCount),
    createdAt: new Date().toISOString(),
  };
}

export function toRegisterDeviceRequest(
  bundle: StoredDeviceKeyBundle,
): RegisterDeviceRequest {
  return {
    device_id: bundle.deviceId,
    name: bundle.deviceName,
    identity_public_key: bundle.identitySigningKey.publicKey,
    signed_pre_key_id: bundle.signedPreKeyId,
    signed_pre_key: bundle.signedPreKey.publicKey,
    signed_pre_key_signature: bundle.signedPreKeySignature,
    one_time_prekeys: bundle.oneTimePreKeys.map(preKey => ({
      key_id: preKey.keyId,
      public_key: preKey.publicKey,
    })),
  };
}

export function verifySignedPreKey(bundle: StoredDeviceKeyBundle): boolean {
  return ed25519.verify(
    base64ToBytes(bundle.signedPreKeySignature),
    signedPreKeyMessage(base64ToBytes(bundle.signedPreKey.publicKey)),
    base64ToBytes(bundle.identitySigningKey.publicKey),
  );
}

export function publicBundleContainsNoPrivateKeys(
  request: RegisterDeviceRequest,
): boolean {
  const encoded = JSON.stringify(request).toLowerCase();
  return !encoded.includes("secret") && !encoded.includes("private");
}

function createOneTimePreKeys(count: number): StoredOneTimePreKey[] {
  const out: StoredOneTimePreKey[] = [];
  for (let i = 0; i < count; i++) {
    const keyPair = x25519.keygen();
    out.push({
      keyId: i + 1,
      publicKey: bytesToBase64(keyPair.publicKey),
      secretKey: bytesToBase64(keyPair.secretKey),
    });
  }
  return out;
}

function serializeKeyPair(keyPair: {
  publicKey: Uint8Array;
  secretKey: Uint8Array;
}): SerializedKeyPair {
  return {
    publicKey: bytesToBase64(keyPair.publicKey),
    secretKey: bytesToBase64(keyPair.secretKey),
  };
}

function signedPreKeyMessage(publicKey: Uint8Array): Uint8Array {
  const context = new TextEncoder().encode(SIGNED_PREKEY_CONTEXT);
  const out = new Uint8Array(context.length + publicKey.length);
  out.set(context, 0);
  out.set(publicKey, context.length);
  return out;
}

function defaultDeviceName(): string {
  if (typeof navigator !== "undefined" && navigator.userAgent) {
    if (navigator.userAgent.includes("Windows")) return "Windows device";
    if (navigator.userAgent.includes("Mac")) return "macOS device";
    if (navigator.userAgent.includes("Linux")) return "Linux device";
  }
  return "Zenthril device";
}
