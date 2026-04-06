import { describe, it, expect, beforeEach } from "vitest";
import {
  generateKeyPair,
  exportPublicKey,
  importPublicKey,
  deriveSharedSecret,
  encrypt,
  decrypt,
  rotateSessionKey,
  storePrivateKey,
  loadPrivateKey,
} from "./index";

// ─── Генерация ключевой пары ──────────────────────────────────────────────────

describe("generateKeyPair", () => {
  it("возвращает пару ключей с secretKey и publicKey", () => {
    const kp = generateKeyPair();
    expect(kp.secretKey).toBeInstanceOf(Uint8Array);
    expect(kp.publicKey).toBeInstanceOf(Uint8Array);
  });

  it("приватный ключ — 32 байта, публичный ключ — 32 байта (X25519)", () => {
    const kp = generateKeyPair();
    expect(kp.secretKey.length).toBe(32);
    expect(kp.publicKey.length).toBe(32);
  });

  it("каждый вызов генерирует уникальную пару", () => {
    const kp1 = generateKeyPair();
    const kp2 = generateKeyPair();
    expect(exportPublicKey(kp1.publicKey)).not.toBe(
      exportPublicKey(kp2.publicKey),
    );
  });
});

// ─── Экспорт / импорт публичного ключа ───────────────────────────────────────

describe("exportPublicKey / importPublicKey", () => {
  it("round-trip: экспорт → импорт возвращает эквивалентный ключ", () => {
    const kp = generateKeyPair();
    const b64 = exportPublicKey(kp.publicKey);
    const imported = importPublicKey(b64);
    expect(exportPublicKey(imported)).toBe(b64);
  });

  it("экспортированный ключ — непустая base64-строка", () => {
    const kp = generateKeyPair();
    const b64 = exportPublicKey(kp.publicKey);
    expect(typeof b64).toBe("string");
    expect(b64.length).toBeGreaterThan(0);
  });
});

// ─── Round-trip шифрования ────────────────────────────────────────────────────

describe("encrypt / decrypt — round-trip", () => {
  let sharedKey: CryptoKey;

  beforeEach(async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();
    sharedKey = await deriveSharedSecret(alice.secretKey, bob.publicKey);
  });

  it("decrypt(encrypt(text)) === исходный текст", async () => {
    const original = "Привет, Veltrix!";
    const payload = await encrypt(original, sharedKey);
    const result = await decrypt(payload, sharedKey);
    expect(result).toBe(original);
  });

  it("round-trip для пустой строки", async () => {
    const payload = await encrypt("", sharedKey);
    const result = await decrypt(payload, sharedKey);
    expect(result).toBe("");
  });

  it("round-trip для длинного текста (4000 символов)", async () => {
    const long = "a".repeat(4000);
    const payload = await encrypt(long, sharedKey);
    const result = await decrypt(payload, sharedKey);
    expect(result).toBe(long);
  });

  it("round-trip для Unicode / emoji", async () => {
    const text = "🔐 Секрет: Alice";
    const payload = await encrypt(text, sharedKey);
    const result = await decrypt(payload, sharedKey);
    expect(result).toBe(text);
  });

  it("зашифрованный payload не содержит исходный текст в открытом виде", async () => {
    const original = "super-secret-message";
    const payload = await encrypt(original, sharedKey);
    expect(payload.ciphertext).not.toContain(original);
    expect(payload.iv).not.toContain(original);
  });

  it("каждый вызов encrypt генерирует уникальный IV", async () => {
    const payload1 = await encrypt("test", sharedKey);
    const payload2 = await encrypt("test", sharedKey);
    expect(payload1.iv).not.toBe(payload2.iv);
  });

  it("payload содержит ciphertext, iv, tag, keyId", async () => {
    const payload = await encrypt("hello", sharedKey);
    expect(payload.ciphertext).toBeTruthy();
    expect(payload.iv).toBeTruthy();
    expect(payload.tag).toBeTruthy();
    expect(payload.keyId).toBeTruthy();
  });

  it("дешифрование с неверным ключом выбрасывает ошибку", async () => {
    const wrongKey = await rotateSessionKey("__test_wrong__");
    const payload = await encrypt("secret", sharedKey);
    await expect(decrypt(payload, wrongKey)).rejects.toThrow();
  });
});

// ─── deriveSharedSecret ───────────────────────────────────────────────────────

describe("deriveSharedSecret", () => {
  it("Alice и Bob получают одинаковый общий секрет (симметрия ECDH)", async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();

    const aliceShared = await deriveSharedSecret(
      alice.secretKey,
      bob.publicKey,
    );
    const bobShared = await deriveSharedSecret(bob.secretKey, alice.publicKey);

    // Шифруем ключом Alice, дешифруем ключом Bob
    const payload = await encrypt("ECDH works", aliceShared);
    const result = await decrypt(payload, bobShared);
    expect(result).toBe("ECDH works");
  });
});

// ─── rotateSessionKey ─────────────────────────────────────────────────────────

describe("rotateSessionKey", () => {
  it("возвращает CryptoKey для AES-GCM", async () => {
    const key = await rotateSessionKey("channel-1");
    expect(key.algorithm.name).toBe("AES-GCM");
  });

  it("повторный вызов заменяет ключ (новый объект)", async () => {
    const key1 = await rotateSessionKey("channel-2");
    const key2 = await rotateSessionKey("channel-2");
    expect(key1).not.toBe(key2);
  });
});

// ─── storePrivateKey / loadPrivateKey ─────────────────────────────────────────

describe("storePrivateKey / loadPrivateKey", () => {
  beforeEach(() => {
    localStorage.removeItem("veltrix_private_key");
  });

  it("сохранённый ключ можно загрузить обратно", async () => {
    const kp = generateKeyPair();
    await storePrivateKey(kp.secretKey);
    const loaded = await loadPrivateKey();
    expect(loaded).not.toBeNull();
    expect(loaded).toBeInstanceOf(Uint8Array);
    expect(loaded!.length).toBe(32);
  });

  it("loadPrivateKey возвращает null, если ключ не сохранён", async () => {
    const result = await loadPrivateKey();
    expect(result).toBeNull();
  });

  it("загруженный ключ функционально эквивалентен оригиналу", async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();

    await storePrivateKey(alice.secretKey);
    const loadedPrivate = (await loadPrivateKey())!;

    const sharedOriginal = await deriveSharedSecret(
      alice.secretKey,
      bob.publicKey,
    );
    const sharedLoaded = await deriveSharedSecret(loadedPrivate, bob.publicKey);

    const payload = await encrypt("persistence test", sharedOriginal);
    const result = await decrypt(payload, sharedLoaded);
    expect(result).toBe("persistence test");
  });
});
