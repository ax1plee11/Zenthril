import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { CIPHER_SUITE_V2, encrypt, decrypt } from "../../crypto";
import type { EncryptedPayload } from "../../types";

const GROUP_SESSION_VERSION = 1;
const MAX_GROUP_DEVICES = 10;
const KEY_BYTES = 32;
const NONCE_BYTES = 12;

// SECURITY: Group Sender Keys foundation. This is NOT full MLS.
// For groups larger than MAX_GROUP_DEVICES, the implementation
// intentionally fails closed to prevent insecure fallback.
// E2EE: per-message keys derived from a shared group chain key.

export interface GroupMember {
  userId: string;
  deviceId: string;
}

export interface GroupSessionState {
  version: typeof GROUP_SESSION_VERSION;
  groupId: string;
  chainKey: Uint8Array;
  messageCounter: number;
  members: GroupMember[];
}

export interface StoredGroupSession {
  version: typeof GROUP_SESSION_VERSION;
  groupId: string;
  chainKey: string;
  messageCounter: number;
  members: GroupMember[];
}

export interface GroupMessageKey {
  key: Uint8Array;
  nonce: Uint8Array;
  counter: number;
  state: GroupSessionState;
}

export interface EncryptedGroupMessage {
  payload: EncryptedPayload;
  state: GroupSessionState;
}

// SECURITY: validates group size before creating a session.
// Groups exceeding MAX_GROUP_DEVICES fail closed.
export function createGroupSession(groupId: string, members: GroupMember[]): GroupSessionState {
  if (members.length > MAX_GROUP_DEVICES) {
    throw new Error(`Group exceeds maximum device limit of ${MAX_GROUP_DEVICES}`);
  }
  if (members.length === 0) {
    throw new Error("Group must have at least one member");
  }
  const seed = new Uint8Array(32);
  crypto.getRandomValues(seed);
  return {
    version: GROUP_SESSION_VERSION,
    groupId,
    chainKey: deriveGroupChainKey(seed, groupId, members),
    messageCounter: 0,
    members: members.map((m) => ({ ...m })),
  };
}

// E2EE: advances the group chain and returns the next message key.
export function nextGroupMessageKey(state: GroupSessionState): GroupMessageKey {
  if (state.version !== GROUP_SESSION_VERSION) {
    throw new Error("Unsupported group session version");
  }
  const step = advanceGroupChain(state.chainKey, state.messageCounter);
  const next: GroupSessionState = {
    ...state,
    chainKey: step.newChainKey,
    messageCounter: state.messageCounter + 1,
    members: state.members.map((m) => ({ ...m })),
  };
  return {
    key: step.messageKey,
    nonce: step.messageNonce,
    counter: state.messageCounter,
    state: next,
  };
}

// SECURITY: encrypts a message for a group. All members can decrypt
// using the same group chain key. The returned state is advanced;
// the caller must persist it for the next message.
export async function encryptGroupMessage(
  plaintext: string,
  state: GroupSessionState,
  aad?: { senderUserId?: string; senderDeviceId?: string },
): Promise<EncryptedGroupMessage> {
  if (state.members.length > MAX_GROUP_DEVICES) {
    throw new Error(`Group exceeds maximum device limit of ${MAX_GROUP_DEVICES}`);
  }
  const step = nextGroupMessageKey(state);
  const payload = await encrypt(plaintext, await importGroupMessageKey(step.key), {
    channelId: state.groupId,
    senderUserId: aad?.senderUserId ?? "unknown",
    senderDeviceId: aad?.senderDeviceId ?? "unknown",
    sessionId: state.groupId,
    clientMessageId: `${state.groupId}:${step.counter}`,
    cipherSuite: CIPHER_SUITE_V2,
  });
  return { payload, state: step.state };
}

// SECURITY: decrypts a group message using the group chain key.
// Returns null if the message cannot be decrypted.
export async function decryptGroupMessage(
  payload: EncryptedPayload,
  state: GroupSessionState,
): Promise<string | null> {
  if (state.version !== GROUP_SESSION_VERSION) {
    throw new Error("Unsupported group session version");
  }
  const counter = payload.clientMessageId?.split(":")[1];
  const counterNum = counter ? parseInt(counter, 10) : state.messageCounter;
  if (Number.isNaN(counterNum) || counterNum < state.messageCounter) {
    return null;
  }
  if (counterNum > state.messageCounter) {
    const gap = counterNum - state.messageCounter;
    if (gap > MAX_GROUP_DEVICES) {
      throw new Error("Skipped group message key limit exceeded");
    }
    for (let i = state.messageCounter; i < counterNum; i++) {
      const step = advanceGroupChain(state.chainKey, i);
      state.chainKey = step.newChainKey;
      state.messageCounter = i + 1;
    }
  }
  const step = advanceGroupChain(state.chainKey, state.messageCounter);
  const key = await importGroupMessageKey(step.messageKey);
  try {
    const text = await decrypt(payload, key);
    state.chainKey = step.newChainKey;
    state.messageCounter++;
    return text;
  } catch {
    return null;
  }
}

// SECURITY: serializes group session for secure storage.
export function serializeGroupSession(state: GroupSessionState): StoredGroupSession {
  return {
    version: GROUP_SESSION_VERSION,
    groupId: state.groupId,
    chainKey: bufferToBase64(state.chainKey),
    messageCounter: state.messageCounter,
    members: state.members.map((m) => ({ ...m })),
  };
}

// SECURITY: restores group session from secure storage.
export function restoreGroupSession(stored: StoredGroupSession): GroupSessionState {
  if (stored.version !== GROUP_SESSION_VERSION) {
    throw new Error("Unsupported group session version");
  }
  return {
    version: stored.version,
    groupId: stored.groupId,
    chainKey: base64ToBuffer(stored.chainKey),
    messageCounter: stored.messageCounter,
    members: stored.members.map((m) => ({ ...m })),
  };
}

// E2EE: derives the initial group chain key from a random seed.
function deriveGroupChainKey(seed: Uint8Array, groupId: string, members: GroupMember[]): Uint8Array {
  const info = new TextEncoder().encode(`zenthril-group:${groupId}:${members.map((m) => `${m.userId}:${m.deviceId}`).sort().join(",")}`);
  const material = hkdf(sha256, seed, undefined, info, KEY_BYTES);
  return material.slice(0, KEY_BYTES);
}

// E2EE: advances the group chain and derives the next message key.
export function advanceGroupChain(chainKey: Uint8Array, counter: number): { newChainKey: Uint8Array; messageKey: Uint8Array; messageNonce: Uint8Array } {
  const info = new TextEncoder().encode(`zenthril-group:msg:${counter}`);
  const output = hkdf(sha256, chainKey, undefined, info, KEY_BYTES * 2 + NONCE_BYTES);
  return {
    newChainKey: output.slice(0, KEY_BYTES),
    messageKey: output.slice(KEY_BYTES, KEY_BYTES * 2),
    messageNonce: output.slice(KEY_BYTES * 2, KEY_BYTES * 2 + NONCE_BYTES),
  };
}

// SECURITY: imports the raw group message key into WebCrypto.
async function importGroupMessageKey(key: Uint8Array): Promise<CryptoKey> {
  if (key.length !== KEY_BYTES) throw new Error("Invalid group message key");
  return crypto.subtle.importKey("raw", key.buffer as ArrayBuffer, { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
}

function bufferToBase64(buffer: Uint8Array): string {
  const bytes = new Uint8Array(buffer.length);
  bytes.set(buffer);
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i] ?? 0);
  }
  return btoa(binary);
}

function base64ToBuffer(base64: string): Uint8Array {
  const binary = atob(base64 ?? "");
  if (!binary) throw new Error("Invalid base64");
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
