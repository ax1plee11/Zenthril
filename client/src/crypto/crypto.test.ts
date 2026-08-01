import { beforeEach, describe, expect, it } from "vitest";
import {
  CIPHER_SUITE_V2,
  canUseLegacyChannelKeys,
  CRYPTO_PROTOCOL_VERSION,
  decrypt,
  deriveMessageKeyBytes,
  deriveSharedSecret,
  encryptedPayloadAAD,
  encrypt,
  encryptLegacyForAlphaCompatibility,
  exportPublicKey,
  generateKeyPair,
  importPublicKey,
  loadPrivateKey,
  rotateSessionKey,
  storePrivateKey,
} from "./index";
import type { CryptoAADContextInput } from "./index";

function bytesToBase64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes));
}

function base64ToBytes(value: string): Uint8Array {
  return Uint8Array.from(atob(value), c => c.charCodeAt(0));
}

function flipBase64Byte(value: string): string {
  const bytes = base64ToBytes(value);
  bytes[0] = (bytes[0] ?? 0) ^ 1;
  return bytesToBase64(bytes);
}

function testAAD(clientMessageId = "client-message-1"): CryptoAADContextInput {
  return {
    channelId: "channel-1",
    senderUserId: "user-1",
    senderDeviceId: "device-1",
    sessionId: "session-1",
    clientMessageId,
  };
}

describe("generateKeyPair", () => {
  it("returns an X25519 key pair", () => {
    const kp = generateKeyPair();
    expect(kp.secretKey).toBeInstanceOf(Uint8Array);
    expect(kp.publicKey).toBeInstanceOf(Uint8Array);
    expect(kp.secretKey).toHaveLength(32);
    expect(kp.publicKey).toHaveLength(32);
  });

  it("generates unique public keys", () => {
    const kp1 = generateKeyPair();
    const kp2 = generateKeyPair();
    expect(exportPublicKey(kp1.publicKey)).not.toBe(exportPublicKey(kp2.publicKey));
  });
});

describe("exportPublicKey / importPublicKey", () => {
  it("round-trips public keys through base64", () => {
    const kp = generateKeyPair();
    const b64 = exportPublicKey(kp.publicKey);
    const imported = importPublicKey(b64);
    expect(exportPublicKey(imported)).toBe(b64);
  });
});

describe("HKDF and AAD helpers", () => {
  it("deriveMessageKeyBytes is deterministic and separates raw ECDH from AES key bytes", () => {
    const sharedSecret = new Uint8Array(32);
    sharedSecret[31] = 1;

    const first = deriveMessageKeyBytes(sharedSecret);
    const second = deriveMessageKeyBytes(sharedSecret);

    expect(Array.from(first)).toEqual(Array.from(second));
    expect(Array.from(first)).not.toEqual(Array.from(sharedSecret));
    expect(first).toHaveLength(32);
  });

  it("AAD changes when protocol version or key id changes", () => {
    const base = encryptedPayloadAAD(CRYPTO_PROTOCOL_VERSION, "key-1");
    const differentKey = encryptedPayloadAAD(CRYPTO_PROTOCOL_VERSION, "key-2");
    const differentVersion = encryptedPayloadAAD(CRYPTO_PROTOCOL_VERSION + 1, "key-1");

    expect(Array.from(base)).not.toEqual(Array.from(differentKey));
    expect(Array.from(base)).not.toEqual(Array.from(differentVersion));
  });
});

describe("encrypt / decrypt", () => {
  let sharedKey: CryptoKey;
  const aadInput = testAAD();

  beforeEach(async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();
    sharedKey = await deriveSharedSecret(alice.secretKey, bob.publicKey);
  });

  it("decrypt(encrypt(text)) returns the original text", async () => {
    const original = "Hello, Zenthril";
    const payload = await encrypt(original, sharedKey, aadInput);
    const result = await decrypt(payload, sharedKey);
    expect(result).toBe(original);
  });

  it("round-trips empty, long, and unicode text", async () => {
    for (const value of ["", "a".repeat(4000), "secret unicode text"]) {
      const payload = await encrypt(value, sharedKey, {
        ...aadInput,
        clientMessageId: `client-message-${value.length}`,
      });
      await expect(decrypt(payload, sharedKey)).resolves.toBe(value);
    }
  });

  it("returns a complete protocol v2 payload when AAD context is provided", async () => {
    const payload = await encrypt("hello", sharedKey, aadInput);
    expect(payload.ciphertext).toBeTruthy();
    expect(payload.iv).toBeTruthy();
    expect(payload.tag).toBeTruthy();
    expect(payload.keyId).toBeTruthy();
    expect(payload.protocolVersion).toBe(CRYPTO_PROTOCOL_VERSION);
    expect(payload.channelId).toBe(aadInput.channelId);
    expect(payload.senderUserId).toBe(aadInput.senderUserId);
    expect(payload.senderDeviceId).toBe(aadInput.senderDeviceId);
    expect(payload.sessionId).toBe(aadInput.sessionId);
    expect(payload.clientMessageId).toBe(aadInput.clientMessageId);
    expect(payload.cipherSuite).toBe(CIPHER_SUITE_V2);
  });

  it("keeps legacy v1 payloads when no AAD context is provided", async () => {
    const payload = await encryptLegacyForAlphaCompatibility("hello", sharedKey);
    expect(payload.protocolVersion).toBe(1);
    await expect(decrypt(payload, sharedKey)).resolves.toBe("hello");
  });

  it("generates a unique IV for each encryption", async () => {
    const payload1 = await encrypt("test", sharedKey, { ...aadInput, clientMessageId: "client-message-iv-1" });
    const payload2 = await encrypt("test", sharedKey, { ...aadInput, clientMessageId: "client-message-iv-2" });
    expect(payload1.iv).not.toBe(payload2.iv);
  });

  it("rejects the wrong key", async () => {
    const wrongKey = await rotateSessionKey("__test_wrong__");
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(decrypt(payload, wrongKey)).rejects.toThrow();
  });

  it("rejects corrupted ciphertext", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, ciphertext: flipBase64Byte(payload.ciphertext) }, sharedKey),
    ).rejects.toThrow();
  });

  it("rejects invalid IV length", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, iv: bytesToBase64(new Uint8Array([1, 2, 3])) }, sharedKey),
    ).rejects.toThrow("Invalid IV length");
  });

  it("rejects missing or invalid authentication tag", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(decrypt({ ...payload, tag: "" }, sharedKey)).rejects.toThrow(
      "Missing authentication tag",
    );
    await expect(
      decrypt({ ...payload, tag: bytesToBase64(new Uint8Array([1, 2, 3])) }, sharedKey),
    ).rejects.toThrow("Invalid authentication tag length");
  });

  it("rejects AAD key id mismatch", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, keyId: `${payload.keyId}-tampered` }, sharedKey),
    ).rejects.toThrow();
  });

  it("rejects unsupported protocol versions", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, protocolVersion: CRYPTO_PROTOCOL_VERSION + 1 }, sharedKey),
    ).rejects.toThrow("Unsupported crypto protocol version");
  });

  it("rejects v2 ciphertext moved to another channel", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, channelId: "channel-2" }, sharedKey),
    ).rejects.toThrow();
  });

  it("rejects v2 ciphertext moved to another sender user", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, senderUserId: "user-2" }, sharedKey),
    ).rejects.toThrow();
  });

  it("rejects v2 ciphertext moved to another sender device", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, senderDeviceId: "device-2" }, sharedKey),
    ).rejects.toThrow();
  });

  it("rejects v2 ciphertext moved to another session or client message", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(decrypt({ ...payload, sessionId: "session-2" }, sharedKey)).rejects.toThrow();
    await expect(decrypt({ ...payload, clientMessageId: "client-message-2" }, sharedKey)).rejects.toThrow();
  });

  it("rejects v2 ciphertext with wrong cipher suite", async () => {
    const payload = await encrypt("secret", sharedKey, aadInput);
    await expect(
      decrypt({ ...payload, cipherSuite: "WRONG" }, sharedKey),
    ).rejects.toThrow("Unsupported cipher suite");
  });
});

describe("deriveSharedSecret", () => {
  it("Alice and Bob derive compatible HKDF-based AES-GCM keys", async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();

    const aliceShared = await deriveSharedSecret(alice.secretKey, bob.publicKey);
    const bobShared = await deriveSharedSecret(bob.secretKey, alice.publicKey);

    const payload = await encrypt("ECDH works", aliceShared, testAAD("client-message-ecdh"));
    const result = await decrypt(payload, bobShared);
    expect(result).toBe("ECDH works");
  });
});

describe("rotateSessionKey", () => {
  it("returns an AES-GCM CryptoKey", async () => {
    const key = await rotateSessionKey("channel-1");
    expect(key.algorithm.name).toBe("AES-GCM");
  });

  it("replaces an existing channel key", async () => {
    const key1 = await rotateSessionKey("channel-2");
    const key2 = await rotateSessionKey("channel-2");
    expect(key1).not.toBe(key2);
  });

  it("is explicitly limited to development compatibility mode", () => {
    // Production builds must use recipient-device session distribution instead
    // of a random AES key that exists only in one client process.
    expect(canUseLegacyChannelKeys({ PROD: false })).toBe(true);
    expect(canUseLegacyChannelKeys({ PROD: true })).toBe(false);
  });
});

describe("storePrivateKey / loadPrivateKey", () => {
  beforeEach(() => {
    localStorage.removeItem("zenthril_private_key");
  });

  it("loads a stored private key in development fallback mode", async () => {
    const kp = generateKeyPair();
    await storePrivateKey(kp.secretKey);
    const loaded = await loadPrivateKey();
    expect(loaded).not.toBeNull();
    expect(loaded).toBeInstanceOf(Uint8Array);
    expect(loaded).toHaveLength(32);
  });

  it("returns null when the key is missing", async () => {
    await expect(loadPrivateKey()).resolves.toBeNull();
  });

  it("loaded keys remain functional", async () => {
    const alice = generateKeyPair();
    const bob = generateKeyPair();

    await storePrivateKey(alice.secretKey);
    const loadedPrivate = (await loadPrivateKey())!;

    const sharedOriginal = await deriveSharedSecret(alice.secretKey, bob.publicKey);
    const sharedLoaded = await deriveSharedSecret(loadedPrivate, bob.publicKey);

    const payload = await encrypt("persistence test", sharedOriginal, testAAD("client-message-persistence"));
    const result = await decrypt(payload, sharedLoaded);
    expect(result).toBe("persistence test");
  });
});
