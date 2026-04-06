/**
 * CryptoModule — E2EE для Veltrix
 *
 * X25519 ECDH: @noble/curves (работает в браузере, Tauri и Node.js)
 * AES-256-GCM:  WebCrypto API (window.crypto.subtle)
 */

import { x25519 } from "@noble/curves/ed25519.js";
import type { EncryptedPayload } from "../types/index";

// ─── Типы ────────────────────────────────────────────────────────────────────

/** Ключевая пара X25519: приватный и публичный ключи в виде Uint8Array */
export interface X25519KeyPair {
  secretKey: Uint8Array;
  publicKey: Uint8Array;
}

// ─── Константы ───────────────────────────────────────────────────────────────

const PRIVATE_KEY_STORAGE_KEY = "veltrix_private_key";

/** Сессионные ключи: channelId → CryptoKey (AES-256-GCM) */
const sessionKeys = new Map<string, CryptoKey>();

// ─── Вспомогательные функции ─────────────────────────────────────────────────

function bufferToBase64(buf: Uint8Array | ArrayBuffer): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
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

/** Копия в BufferSource, совместимый с типами WebCrypto (TS 5.6+). */
function toBufferSource(u8: Uint8Array): BufferSource {
  const buf = new ArrayBuffer(u8.byteLength);
  new Uint8Array(buf).set(u8);
  return buf;
}

// ─── Генерация ключевой пары ─────────────────────────────────────────────────

/**
 * Генерирует X25519 ключевую пару.
 * Использует @noble/curves для совместимости с браузером, Tauri и Node.js.
 */
export function generateKeyPair(): X25519KeyPair {
  return x25519.keygen();
}

// ─── Экспорт / импорт публичного ключа ───────────────────────────────────────

/**
 * Экспортирует публичный ключ в base64.
 */
export function exportPublicKey(key: Uint8Array): string {
  return bufferToBase64(key);
}

/**
 * Импортирует публичный ключ из base64.
 */
export function importPublicKey(base64: string): Uint8Array {
  return base64ToUint8Array(base64);
}

// ─── ECDH: общий секрет → AES-256-GCM ключ ───────────────────────────────────

/**
 * Выполняет X25519 ECDH и возвращает AES-256-GCM CryptoKey из общего секрета.
 * Общий секрет (32 байта) используется напрямую как ключ AES-256.
 */
export async function deriveSharedSecret(
  myPrivateKey: Uint8Array,
  theirPublicKey: Uint8Array,
): Promise<CryptoKey> {
  const sharedBytes = toBufferSource(
    x25519.getSharedSecret(myPrivateKey, theirPublicKey),
  );
  return crypto.subtle.importKey(
    "raw",
    sharedBytes,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

// ─── AES-256-GCM: шифрование / дешифрование ──────────────────────────────────

/**
 * Шифрует plaintext с помощью AES-256-GCM.
 * WebCrypto возвращает ciphertext + 16-байтовый auth tag слитно.
 * Мы разделяем их для явного хранения в EncryptedPayload.
 */
export async function encrypt(
  plaintext: string,
  key: CryptoKey,
): Promise<EncryptedPayload> {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encoded = new TextEncoder().encode(plaintext);

  const ciphertextWithTag = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv, tagLength: 128 },
    key,
    encoded,
  );

  const ctBytes = new Uint8Array(ciphertextWithTag);
  const tagOffset = ctBytes.length - 16;
  const ciphertextBytes = ctBytes.slice(0, tagOffset);
  const tagBytes = ctBytes.slice(tagOffset);

  return {
    ciphertext: bufferToBase64(ciphertextBytes),
    iv: bufferToBase64(iv),
    tag: bufferToBase64(tagBytes),
    keyId: generateKeyId(),
  };
}

/**
 * Дешифрует EncryptedPayload с помощью AES-256-GCM.
 */
export async function decrypt(
  payload: EncryptedPayload,
  key: CryptoKey,
): Promise<string> {
  const ivBytes = base64ToUint8Array(payload.iv);
  const ciphertextBytes = base64ToUint8Array(payload.ciphertext);
  const tagBytes = base64ToUint8Array(payload.tag);

  // Собираем обратно: ciphertext || tag
  const combined = new Uint8Array(ciphertextBytes.length + tagBytes.length);
  combined.set(ciphertextBytes, 0);
  combined.set(tagBytes, ciphertextBytes.length);

  const plainBuffer = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: toBufferSource(ivBytes), tagLength: 128 },
    key,
    toBufferSource(combined),
  );

  return new TextDecoder().decode(plainBuffer);
}

// ─── Ротация сессионного ключа ────────────────────────────────────────────────

/**
 * Генерирует новый AES-256-GCM ключ для канала (forward secrecy).
 */
export async function rotateSessionKey(channelId: string): Promise<CryptoKey> {
  const newKey = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
  sessionKeys.set(channelId, newKey);
  return newKey;
}

/**
 * Возвращает текущий сессионный ключ для канала или создаёт новый.
 */
export async function getOrCreateSessionKey(
  channelId: string,
): Promise<CryptoKey> {
  const existing = sessionKeys.get(channelId);
  if (existing) return existing;
  return rotateSessionKey(channelId);
}

// ─── Tauri detection ─────────────────────────────────────────────────────────

/**
 * Возвращает true если код выполняется внутри Tauri-приложения.
 */
export function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

// ─── Хранение приватного ключа ────────────────────────────────────────────────

/**
 * Сохраняет приватный ключ X25519.
 * В Tauri-окружении использует invoke("store_private_key") для безопасного хранилища,
 * иначе fallback на localStorage.
 */
export async function storePrivateKey(key: Uint8Array): Promise<void> {
  const b64 = bufferToBase64(key);
  if (isTauri()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("store_private_key", { key: b64 });
  } else {
    localStorage.setItem(PRIVATE_KEY_STORAGE_KEY, b64);
  }
}

/**
 * Загружает приватный ключ.
 * В Tauri-окружении использует invoke("load_private_key"),
 * иначе fallback на localStorage.
 * Возвращает null, если ключ не найден.
 */
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
    const raw = localStorage.getItem(PRIVATE_KEY_STORAGE_KEY);
    if (!raw) return null;
    try {
      return base64ToUint8Array(raw);
    } catch {
      return null;
    }
  }
}
