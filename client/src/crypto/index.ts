import { x25519 } from "@noble/curves/ed25519.js";
import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import type { EncryptedPayload } from "../types/index";

export const LEGACY_CRYPTO_PROTOCOL_VERSION = 1;
export const CRYPTO_PROTOCOL_VERSION = 2;
export const CIPHER_SUITE_V2 = "X25519-HKDF-SHA256-AES-256-GCM";

const PRIVATE_KEY_STORAGE_KEY = "zenthril_private_key";
const KEY_BYTES = 32;
const AES_GCM_IV_BYTES = 12;
const AES_GCM_TAG_BYTES = 16;
const ECDH_HKDF_SALT = new TextEncoder().encode("zenthril.e2ee.ecdh.hkdf.v1");
const ECDH_HKDF_INFO = new TextEncoder().encode("zenthril.e2ee.message-key.v1");

const sessionKeys = new Map<string, CryptoKey>();

export interface X25519KeyPair {
  secretKey: Uint8Array;
  publicKey: Uint8Array;
}

export type CryptoAADContext = {
  protocolVersion: number;
  keyId: string;
  channelId: string;
  senderUserId: string;
  senderDeviceId: string;
  sessionId: string;
  clientMessageId: string;
  cipherSuite: typeof CIPHER_SUITE_V2;
};

export type CryptoAADContextInput = Omit<CryptoAADContext, "protocolVersion" | "keyId" | "cipherSuite">;

function bufferToBase64(buf: Uint8Array | ArrayBuffer): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i] ?? 0);
  }
  return btoa(binary);
}

function base64ToUint8Array(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function generateKeyId(): string {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return bufferToBase64(bytes);
}

function toBufferSource(u8: Uint8Array): BufferSource {
  const buf = new ArrayBuffer(u8.byteLength);
  new Uint8Array(buf).set(u8);
  return buf;
}

export function deriveMessageKeyBytes(sharedSecret: Uint8Array): Uint8Array {
  if (sharedSecret.byteLength !== KEY_BYTES) {
    throw new Error("sharedSecret must be 32 bytes");
  }
  return hkdf(sha256, sharedSecret, ECDH_HKDF_SALT, ECDH_HKDF_INFO, KEY_BYTES);
}

export function encryptedPayloadAAD(protocolVersion: number, keyId: string): Uint8Array {
  if (!Number.isInteger(protocolVersion) || protocolVersion <= 0) {
    throw new Error("protocolVersion must be a positive integer");
  }
  if (!keyId) {
    throw new Error("keyId is required");
  }
  return new TextEncoder().encode(
    `zenthril.e2ee.payload|v=${protocolVersion}|key_id=${keyId}`,
  );
}

export function buildAAD(context: CryptoAADContext): Uint8Array {
  if (context.protocolVersion !== CRYPTO_PROTOCOL_VERSION) {
    throw new Error("AAD context protocolVersion must be 2");
  }
  if (context.cipherSuite !== CIPHER_SUITE_V2) {
    throw new Error("Unsupported cipher suite");
  }
  for (const [key, value] of Object.entries(context)) {
    if (typeof value === "string" && value.trim() === "") {
      throw new Error(`AAD context ${key} is required`);
    }
  }

  // SECURITY: deterministic canonical serialization prevents subtle AAD drift
  // between encryption/decryption call sites.
  const canonical = {
    channelId: context.channelId,
    cipherSuite: context.cipherSuite,
    clientMessageId: context.clientMessageId,
    keyId: context.keyId,
    protocolVersion: context.protocolVersion,
    senderDeviceId: context.senderDeviceId,
    senderUserId: context.senderUserId,
    sessionId: context.sessionId,
  };
  return new TextEncoder().encode(`zenthril.e2ee.aad.v2|${JSON.stringify(canonical)}`);
}

export function generateKeyPair(): X25519KeyPair {
  return x25519.keygen();
}

export function exportPublicKey(key: Uint8Array): string {
  return bufferToBase64(key);
}

export function importPublicKey(base64: string): Uint8Array {
  return base64ToUint8Array(base64);
}

export async function deriveSharedSecret(
  myPrivateKey: Uint8Array,
  theirPublicKey: Uint8Array,
): Promise<CryptoKey> {
  const sharedBytes = x25519.getSharedSecret(myPrivateKey, theirPublicKey);
  const keyBytes = deriveMessageKeyBytes(sharedBytes);

  try {
    return await crypto.subtle.importKey(
      "raw",
      toBufferSource(keyBytes),
      { name: "AES-GCM", length: 256 },
      false,
      ["encrypt", "decrypt"],
    );
  } finally {
    sharedBytes.fill(0);
    keyBytes.fill(0);
  }
}

export async function encrypt(
  plaintext: string,
  key: CryptoKey,
  aadContext: CryptoAADContextInput,
): Promise<EncryptedPayload> {
  return encryptWithContext(plaintext, key, aadContext);
}

// SECURITY-HARDENING: new client sends must use protocol-v2 AAD context. This
// helper exists only to decrypt/test alpha protocol-v1 compatibility behavior.
export async function encryptLegacyForAlphaCompatibility(
  plaintext: string,
  key: CryptoKey,
): Promise<EncryptedPayload> {
  return encryptWithContext(plaintext, key);
}

async function encryptWithContext(
  plaintext: string,
  key: CryptoKey,
  aadContext?: CryptoAADContextInput,
): Promise<EncryptedPayload> {
  const iv = crypto.getRandomValues(new Uint8Array(AES_GCM_IV_BYTES));
  const encoded = new TextEncoder().encode(plaintext);
  const keyId = generateKeyId();
  const protocolVersion = aadContext ? CRYPTO_PROTOCOL_VERSION : LEGACY_CRYPTO_PROTOCOL_VERSION;
  const fullAADContext: CryptoAADContext | undefined = aadContext
    ? {
      protocolVersion,
      keyId,
      channelId: aadContext.channelId,
      senderUserId: aadContext.senderUserId,
      senderDeviceId: aadContext.senderDeviceId,
      sessionId: aadContext.sessionId,
      clientMessageId: aadContext.clientMessageId,
      cipherSuite: CIPHER_SUITE_V2,
    }
    : undefined;
  const aad = fullAADContext ? buildAAD(fullAADContext) : encryptedPayloadAAD(protocolVersion, keyId);

  const ciphertextWithTag = await crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: toBufferSource(iv),
      tagLength: AES_GCM_TAG_BYTES * 8,
      additionalData: toBufferSource(aad),
    },
    key,
    encoded,
  );

  const ctBytes = new Uint8Array(ciphertextWithTag);
  const tagOffset = ctBytes.length - AES_GCM_TAG_BYTES;
  const ciphertextBytes = ctBytes.slice(0, tagOffset);
  const tagBytes = ctBytes.slice(tagOffset);

  const payload: EncryptedPayload = {
    ciphertext: bufferToBase64(ciphertextBytes),
    iv: bufferToBase64(iv),
    tag: bufferToBase64(tagBytes),
    keyId,
    protocolVersion,
  };
  if (fullAADContext) {
    payload.channelId = fullAADContext.channelId;
    payload.senderUserId = fullAADContext.senderUserId;
    payload.senderDeviceId = fullAADContext.senderDeviceId;
    payload.sessionId = fullAADContext.sessionId;
    payload.clientMessageId = fullAADContext.clientMessageId;
    payload.cipherSuite = fullAADContext.cipherSuite;
  }
  return payload;
}

export async function decrypt(
  payload: EncryptedPayload,
  key: CryptoKey,
  aadContext?: CryptoAADContext,
): Promise<string> {
  if (payload.protocolVersion !== LEGACY_CRYPTO_PROTOCOL_VERSION && payload.protocolVersion !== CRYPTO_PROTOCOL_VERSION) {
    throw new Error("Unsupported crypto protocol version");
  }
  if (!payload.keyId) {
    throw new Error("Missing keyId");
  }
  if (!payload.tag) {
    throw new Error("Missing authentication tag");
  }

  const ivBytes = base64ToUint8Array(payload.iv);
  if (ivBytes.byteLength !== AES_GCM_IV_BYTES) {
    throw new Error("Invalid IV length");
  }

  const tagBytes = base64ToUint8Array(payload.tag);
  if (tagBytes.byteLength !== AES_GCM_TAG_BYTES) {
    throw new Error("Invalid authentication tag length");
  }

  const ciphertextBytes = base64ToUint8Array(payload.ciphertext);
  const combined = new Uint8Array(ciphertextBytes.length + tagBytes.length);
  combined.set(ciphertextBytes, 0);
  combined.set(tagBytes, ciphertextBytes.length);

  const aad = payload.protocolVersion === CRYPTO_PROTOCOL_VERSION
    ? buildAAD(aadContext ?? {
      protocolVersion: CRYPTO_PROTOCOL_VERSION,
      keyId: payload.keyId,
      channelId: payload.channelId ?? "",
      senderUserId: payload.senderUserId ?? "",
      senderDeviceId: payload.senderDeviceId ?? "",
      sessionId: payload.sessionId ?? "",
      clientMessageId: payload.clientMessageId ?? "",
      cipherSuite: payload.cipherSuite as typeof CIPHER_SUITE_V2,
    })
    : encryptedPayloadAAD(payload.protocolVersion, payload.keyId);
  const plainBuffer = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: toBufferSource(ivBytes),
      tagLength: AES_GCM_TAG_BYTES * 8,
      additionalData: toBufferSource(aad),
    },
    key,
    toBufferSource(combined),
  );

  return new TextDecoder().decode(plainBuffer);
}

export async function rotateSessionKey(channelId: string): Promise<CryptoKey> {
  const newKey = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
  sessionKeys.set(channelId, newKey);
  return newKey;
}

export async function getOrCreateSessionKey(
  channelId: string,
): Promise<CryptoKey> {
  const existing = sessionKeys.get(channelId);
  if (existing) return existing;
  return rotateSessionKey(channelId);
}

export function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

function allowInsecureLocalKeyStorage(): boolean {
  return !import.meta.env.PROD || import.meta.env.VITE_ALLOW_INSECURE_KEY_STORAGE === "true";
}

export async function storePrivateKey(key: Uint8Array): Promise<void> {
  const b64 = bufferToBase64(key);
  if (isTauri()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("store_private_key", { key: b64 });
  } else {
    if (!allowInsecureLocalKeyStorage()) {
      // SECURITY-HARDENING: production web builds must not silently persist private keys in localStorage.
      throw new Error("Secure private key storage is unavailable");
    }
    localStorage.setItem(PRIVATE_KEY_STORAGE_KEY, b64);
  }
}

export async function loadPrivateKey(): Promise<Uint8Array | null> {
  if (isTauri()) {
    const { invoke } = await import("@tauri-apps/api/core");
    const raw = await invoke<string | null>("load_private_key");
    if (!raw) return null;
    try {
      return base64ToUint8Array(raw);
    } catch {
      return null;
    }
  } else {
    if (!allowInsecureLocalKeyStorage()) {
      // SECURITY-HARDENING: avoid reading production private keys from localStorage fallback.
      return null;
    }
    const raw = localStorage.getItem(PRIVATE_KEY_STORAGE_KEY);
    if (!raw) return null;
    try {
      return base64ToUint8Array(raw);
    } catch {
      return null;
    }
  }
}
