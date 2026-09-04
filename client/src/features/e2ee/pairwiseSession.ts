import { ed25519, x25519 } from "@noble/curves/ed25519.js";
import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { advanceRatchet, dhRatchetTurn } from "../../crypto/ratchet";
import { base64ToBytes, bytesToBase64 } from "./encoding";
import { signedPreKeyMessage } from "./deviceKeys";
import type { KeyBundleAPI, StoredDeviceKeyBundle, StoredPairwiseSession } from "./types";

export const PAIRWISE_SESSION_PROTOCOL_VERSION = 1;
export const PAIRWISE_SESSION_CIPHER_SUITE = "X3DH-HKDF-SHA256-DR-v1";

const ROOT_SALT = new TextEncoder().encode("zenthril.e2ee.x3dh.root.v1");
const ROOT_INFO_PREFIX = "zenthril.e2ee.x3dh.session.v1|";
const KEY_BYTES = 32;
const MAX_SKIPPED_KEYS = 2000;

export interface X3DHSessionHeader {
  version: typeof PAIRWISE_SESSION_PROTOCOL_VERSION;
  cipherSuite: typeof PAIRWISE_SESSION_CIPHER_SUITE;
  sessionId: string;
  senderUserId: string;
  senderDeviceId: string;
  senderIdentityDHPublicKey: string;
  senderEphemeralPublicKey: string;
  senderDHPublicKey: string;
  recipientUserId: string;
  recipientDeviceId: string;
  recipientSignedPreKeyId: number;
  recipientOneTimePreKeyId?: number;
}

export interface PairwiseSessionState {
  version: typeof PAIRWISE_SESSION_PROTOCOL_VERSION;
  sessionId: string;
  peerUserId: string;
  peerDeviceId: string;
  rootKey: Uint8Array;
  sendChainKey: Uint8Array;
  receiveChainKey: Uint8Array;
  sendCounter: number;
  receiveCounter: number;
  dhSendPrivate: Uint8Array;
  dhSendPublic: Uint8Array;
  dhRecvPublic: Uint8Array;
  previousCounter: number;
  skippedMessageKeys: Map<number, { key: Uint8Array; nonce: Uint8Array; counter: number }>;
}

export interface InitiatedPairwiseSession {
  header: X3DHSessionHeader;
  state: PairwiseSessionState;
}

export interface AcceptedPairwiseSession {
  state: PairwiseSessionState;
  updatedLocalBundle: StoredDeviceKeyBundle;
}

export interface RatchetedMessageKey {
  messageKey: Uint8Array;
  messageNonce: Uint8Array;
  counter: number;
  state: PairwiseSessionState;
}

export interface SkippedMessageKey {
  key: Uint8Array;
  nonce: Uint8Array;
  counter: number;
}

export interface PairwiseSessionState {
  version: typeof PAIRWISE_SESSION_PROTOCOL_VERSION;
  sessionId: string;
  peerUserId: string;
  peerDeviceId: string;
  rootKey: Uint8Array;
  sendChainKey: Uint8Array;
  receiveChainKey: Uint8Array;
  sendCounter: number;
  receiveCounter: number;
  dhSendPrivate: Uint8Array;
  dhSendPublic: Uint8Array;
  dhRecvPublic: Uint8Array;
  previousCounter: number;
  skippedMessageKeys: Map<number, { key: Uint8Array; nonce: Uint8Array; counter: number }>;
}

export interface InitiatedPairwiseSession {
  header: X3DHSessionHeader;
  state: PairwiseSessionState;
}

export interface AcceptedPairwiseSession {
  state: PairwiseSessionState;
  updatedLocalBundle: StoredDeviceKeyBundle;
}

export interface RatchetedMessageKey {
  messageKey: Uint8Array;
  counter: number;
  state: PairwiseSessionState;
}

export async function importRatchetMessageKey(messageKey: Uint8Array, messageNonce: Uint8Array): Promise<{ key: CryptoKey; nonce: Uint8Array }> {
  if (messageKey.length !== KEY_BYTES) throw new Error("Invalid ratchet message key");
  if (messageNonce.length !== 12) throw new Error("Invalid ratchet nonce");
  try {
    const keyRaw = new Uint8Array(messageKey.length);
    keyRaw.set(messageKey);
    const key = await crypto.subtle.importKey(
      "raw", keyRaw.buffer,
      { name: "AES-GCM", length: 256 },
      false, ["encrypt", "decrypt"],
    );
    return { key, nonce: new Uint8Array(messageNonce) };
  } finally {
    messageKey.fill(0);
    messageNonce.fill(0);
  }
}

export function serializePairwiseSession(state: PairwiseSessionState): StoredPairwiseSession {
  validateState(state);
  const skipped: Record<number, { key: string; nonce: string; counter: number }> = {};
  for (const [counter, entry] of state.skippedMessageKeys) {
    skipped[counter] = { key: bytesToBase64(entry.key), nonce: bytesToBase64(entry.nonce), counter: entry.counter };
  }
  return {
    version: PAIRWISE_SESSION_PROTOCOL_VERSION,
    sessionId: state.sessionId,
    peerUserId: state.peerUserId,
    peerDeviceId: state.peerDeviceId,
    rootKey: bytesToBase64(state.rootKey),
    sendChainKey: bytesToBase64(state.sendChainKey),
    receiveChainKey: bytesToBase64(state.receiveChainKey),
    sendCounter: state.sendCounter,
    receiveCounter: state.receiveCounter,
    dhSendPublic: bytesToBase64(state.dhSendPublic),
    dhRecvPublic: state.dhRecvPublic.length > 0 ? bytesToBase64(state.dhRecvPublic) : "",
    previousCounter: state.previousCounter,
    skippedMessageKeys: skipped,
  };
}

export function restorePairwiseSession(stored: StoredPairwiseSession): PairwiseSessionState {
  const skipped = new Map<number, { key: Uint8Array; nonce: Uint8Array; counter: number }>();
  if (stored.skippedMessageKeys) {
    for (const [counterStr, entry] of Object.entries(stored.skippedMessageKeys)) {
      skipped.set(Number(counterStr), { key: base64ToBytes(entry.key), nonce: base64ToBytes(entry.nonce), counter: entry.counter });
    }
  }
  const state: PairwiseSessionState = {
    version: stored.version,
    sessionId: stored.sessionId,
    peerUserId: stored.peerUserId,
    peerDeviceId: stored.peerDeviceId,
    rootKey: base64ToBytes(stored.rootKey),
    sendChainKey: base64ToBytes(stored.sendChainKey),
    receiveChainKey: base64ToBytes(stored.receiveChainKey),
    sendCounter: stored.sendCounter,
    receiveCounter: stored.receiveCounter,
    dhSendPrivate: new Uint8Array(32),
    dhSendPublic: base64ToBytes(stored.dhSendPublic),
    dhRecvPublic: stored.dhRecvPublic ? base64ToBytes(stored.dhRecvPublic) : new Uint8Array(32),
    previousCounter: stored.previousCounter ?? 0,
    skippedMessageKeys: skipped,
  };
  validateState(state);
  return state;
}

export function savePairwiseSession(
  bundle: StoredDeviceKeyBundle,
  state: PairwiseSessionState,
): StoredDeviceKeyBundle {
  const stored = serializePairwiseSession(state);
  return {
    ...bundle,
    pairwiseSessions: { ...bundle.pairwiseSessions, [state.sessionId]: stored },
  };
}

export function loadPairwiseSession(
  bundle: StoredDeviceKeyBundle,
  sessionId: string,
): PairwiseSessionState | null {
  const stored = bundle.pairwiseSessions?.[sessionId];
  if (!stored) return null;
  try {
    return restorePairwiseSession(stored);
  } catch {
    return null;
  }
}

export function findPairwiseSessionForPeer(
  bundle: StoredDeviceKeyBundle,
  peerUserId: string,
  peerDeviceId: string,
): PairwiseSessionState | null {
  for (const stored of Object.values(bundle.pairwiseSessions ?? {})) {
    if (stored.peerUserId === peerUserId && stored.peerDeviceId === peerDeviceId) {
      const restored = loadPairwiseSession(bundle, stored.sessionId);
      if (restored) return restored;
    }
  }
  return null;
}

// SECURITY: creates a pairwise session only after authenticating the peer's
// signed prekey. This is an X3DH foundation, not a complete Signal protocol.
// E2EE: full Double Ratchet initialization with DH ratchet keys.
export function initiatePairwiseSession(
  local: StoredDeviceKeyBundle,
  peer: KeyBundleAPI,
): InitiatedPairwiseSession {
  validateLocalBundle(local);
  validatePeerBundle(peer);
  verifyPeerSignedPreKey(peer);

  const ephemeral = x25519.keygen();
  const dhSend = x25519.keygen();
  try {
    const header: X3DHSessionHeader = {
      version: PAIRWISE_SESSION_PROTOCOL_VERSION,
      cipherSuite: PAIRWISE_SESSION_CIPHER_SUITE,
      sessionId: randomSessionID(),
      senderUserId: local.userId,
      senderDeviceId: local.deviceId,
      senderIdentityDHPublicKey: local.identityDHKey.publicKey,
      senderEphemeralPublicKey: bytesToBase64(ephemeral.publicKey),
      senderDHPublicKey: bytesToBase64(dhSend.publicKey),
      recipientUserId: peer.user_id,
      recipientDeviceId: peer.device_id,
      recipientSignedPreKeyId: peer.signed_pre_key_id,
      ...(peer.one_time_prekey ? { recipientOneTimePreKeyId: peer.one_time_prekey.key_id } : {}),
    };

    const dhValues = [
      x25519.getSharedSecret(base64ToBytes(local.identityDHKey.secretKey), base64ToBytes(peer.signed_pre_key)),
      x25519.getSharedSecret(ephemeral.secretKey, base64ToBytes(peer.identity_dh_public_key)),
      x25519.getSharedSecret(ephemeral.secretKey, base64ToBytes(peer.signed_pre_key)),
    ];
    if (peer.one_time_prekey) {
      dhValues.push(x25519.getSharedSecret(ephemeral.secretKey, base64ToBytes(peer.one_time_prekey.public_key)));
    }
    let state = deriveSessionState(header, dhValues, "initiator");
    // E2EE: initialize DH ratchet keys for full Double Ratchet
    const initTurn = initializeDHRatchet(state, base64ToBytes(peer.signed_pre_key), dhSend);
    state = { ...initTurn.state, sendChainKey: initTurn.newChainKey };
    return { header, state };
  } finally {
    ephemeral.secretKey.fill(0);
    dhSend.secretKey.fill(0);
  }
}

// SECURITY: the receiver consumes the selected one-time prekey only after a
// valid bootstrap is accepted, preventing accidental reuse by this device.
// E2EE: full Double Ratchet initialization with DH ratchet keys on accept.
export function acceptPairwiseSession(
  local: StoredDeviceKeyBundle,
  header: X3DHSessionHeader,
): AcceptedPairwiseSession {
  validateLocalBundle(local);
  validateHeader(header);
  if (header.recipientUserId !== local.userId || header.recipientDeviceId !== local.deviceId) {
    throw new Error("X3DH session header is not addressed to this device");
  }
  if (header.recipientSignedPreKeyId !== local.signedPreKeyId) {
    throw new Error("Unknown signed prekey");
  }

  const oneTimePreKey = header.recipientOneTimePreKeyId === undefined
    ? undefined
    : local.oneTimePreKeys.find(key => key.keyId === header.recipientOneTimePreKeyId);
  if (header.recipientOneTimePreKeyId !== undefined && !oneTimePreKey) {
    throw new Error("Unknown or already consumed one-time prekey");
  }

  const senderIdentity = base64ToBytes(header.senderIdentityDHPublicKey);
  const senderEphemeral = base64ToBytes(header.senderEphemeralPublicKey);
  const dhValues = [
    x25519.getSharedSecret(base64ToBytes(local.signedPreKey.secretKey), senderIdentity),
    x25519.getSharedSecret(base64ToBytes(local.identityDHKey.secretKey), senderEphemeral),
    x25519.getSharedSecret(base64ToBytes(local.signedPreKey.secretKey), senderEphemeral),
  ];
  if (oneTimePreKey) {
    dhValues.push(x25519.getSharedSecret(base64ToBytes(oneTimePreKey.secretKey), senderEphemeral));
  }

  let state = deriveSessionState(header, dhValues, "receiver");
  const senderDHPublic = base64ToBytes(header.senderDHPublicKey);
  // E2EE: initial DH ratchet step for receiver uses signed prekey private key,
  // matching the initiator's DH with peer's signed prekey public key.
  const acceptTurn = initializeDHRatchet(state, senderDHPublic, {
    publicKey: base64ToBytes(local.signedPreKey.publicKey),
    secretKey: base64ToBytes(local.signedPreKey.secretKey),
  });
  state = { ...acceptTurn.state, receiveChainKey: acceptTurn.newChainKey };
  const updatedLocalBundle = oneTimePreKey
    ? { ...local, oneTimePreKeys: local.oneTimePreKeys.filter(key => key.keyId !== oneTimePreKey.keyId) }
    : local;
  return { state, updatedLocalBundle };
}

// SECURITY: callers must persist the returned state through secure device
// storage and zero messageKey after importing it into WebCrypto.
export function nextSendMessageKey(state: PairwiseSessionState): RatchetedMessageKey {
  return nextMessageKey(state, "send");
}

// SECURITY: skipped-message-key handling supports bounded out-of-order delivery.
export function nextReceiveMessageKey(state: PairwiseSessionState, counter?: number): RatchetedMessageKey {
  if (counter !== undefined && counter < state.receiveCounter) {
    const skipped = state.skippedMessageKeys.get(counter);
    if (!skipped) throw new Error("Message key unavailable for skipped counter");
    state.skippedMessageKeys.delete(counter);
    return { messageKey: skipped.key, messageNonce: skipped.nonce, counter: skipped.counter, state };
  }
  if (counter !== undefined && counter > state.receiveCounter) {
    const gap = counter - state.receiveCounter;
    if (state.skippedMessageKeys.size + gap > MAX_SKIPPED_KEYS) {
      throw new Error("Skipped message key limit exceeded");
    }
    for (let i = state.receiveCounter; i < counter; i++) {
      const step = advanceRatchet(state.receiveChainKey);
      state.skippedMessageKeys.set(i, { key: step.messageKey, nonce: step.newChainKey.slice(0, 12), counter: i });
      state.receiveChainKey = step.newChainKey;
    }
  }
  return nextMessageKey(state, "receive");
}

function nextMessageKey(state: PairwiseSessionState, direction: "send" | "receive"): RatchetedMessageKey {
  validateState(state);
  const chainKey = direction === "send" ? state.sendChainKey : state.receiveChainKey;
  const step = advanceRatchet(chainKey);
  const next: PairwiseSessionState = {
    ...state,
    rootKey: state.rootKey.slice(),
    sendChainKey: direction === "send" ? step.newChainKey : state.sendChainKey.slice(),
    receiveChainKey: direction === "receive" ? step.newChainKey : state.receiveChainKey.slice(),
    sendCounter: direction === "send" ? state.sendCounter + 1 : state.sendCounter,
    receiveCounter: direction === "receive" ? state.receiveCounter + 1 : state.receiveCounter,
  };
  return {
    messageKey: step.messageKey,
    messageNonce: step.newChainKey.slice(0, 12),
    counter: direction === "send" ? state.sendCounter : state.receiveCounter,
    state: next,
  };
}

function deriveSessionState(
  header: X3DHSessionHeader,
  values: Uint8Array[],
  role: "initiator" | "receiver",
): PairwiseSessionState {
  try {
    const output = hkdf(
      sha256,
      concatBytes(values),
      ROOT_SALT,
      new TextEncoder().encode(`${ROOT_INFO_PREFIX}${canonicalHeader(header)}`),
      KEY_BYTES * 3,
    );
    const initiatorChain = output.slice(KEY_BYTES, KEY_BYTES * 2);
    const receiverChain = output.slice(KEY_BYTES * 2, KEY_BYTES * 3);
    const rootKey = output.slice(0, KEY_BYTES);
    return {
      version: PAIRWISE_SESSION_PROTOCOL_VERSION,
      sessionId: header.sessionId,
      peerUserId: role === "initiator" ? header.recipientUserId : header.senderUserId,
      peerDeviceId: role === "initiator" ? header.recipientDeviceId : header.senderDeviceId,
      rootKey,
      sendChainKey: role === "initiator" ? initiatorChain : receiverChain,
      receiveChainKey: role === "initiator" ? receiverChain : initiatorChain,
      sendCounter: 0,
      receiveCounter: 0,
      dhSendPrivate: new Uint8Array(32),
      dhSendPublic: new Uint8Array(32),
      dhRecvPublic: new Uint8Array(32),
      previousCounter: 0,
      skippedMessageKeys: new Map(),
    };
  } finally {
    for (const value of values) value.fill(0);
  }
}

// E2EE: initializes DH ratchet keys for a newly derived session state.
function initializeDHRatchet(
  state: PairwiseSessionState,
  peerDHPublic: Uint8Array,
  localDH: { publicKey: Uint8Array; secretKey: Uint8Array },
): { state: PairwiseSessionState; newChainKey: Uint8Array } {
  const dhOutput = x25519.getSharedSecret(localDH.secretKey, peerDHPublic);
  const material = hkdf(sha256, dhOutput, state.rootKey, new TextEncoder().encode("zenthril-ratchet-v1:init-dh"), KEY_BYTES * 2);
  const newRootKey = material.slice(0, KEY_BYTES);
  const newChainKey = material.slice(KEY_BYTES, KEY_BYTES * 2);
  return {
    state: {
      ...state,
      rootKey: newRootKey,
      sendChainKey: state.sendChainKey,
      receiveChainKey: state.receiveChainKey,
      dhSendPrivate: localDH.secretKey,
      dhSendPublic: localDH.publicKey,
      dhRecvPublic: peerDHPublic,
    },
    newChainKey,
  };
}

// E2EE: performs a DH ratchet turn when receiving a message with a new DH public key.
export function performDHRatchetTurn(state: PairwiseSessionState, newPeerDHPublic: Uint8Array): PairwiseSessionState {
  const turn = dhRatchetTurn(state.rootKey, state.dhSendPrivate, state.dhSendPublic, newPeerDHPublic);
  return {
    ...state,
    rootKey: turn.newRootKey,
    receiveChainKey: turn.newRecvChainKey,
    sendChainKey: turn.newSendChainKey,
    dhSendPrivate: turn.newDHPrivate,
    dhSendPublic: turn.newDHPublic,
    dhRecvPublic: newPeerDHPublic,
    previousCounter: state.sendCounter,
    sendCounter: 0,
    receiveCounter: 0,
    skippedMessageKeys: new Map(),
  };
}

function verifyPeerSignedPreKey(peer: KeyBundleAPI): void {
  try {
    const valid = ed25519.verify(
      base64ToBytes(peer.signed_pre_key_signature),
      signedPreKeyMessage(base64ToBytes(peer.signed_pre_key)),
      base64ToBytes(peer.identity_public_key),
    );
    if (!valid) throw new Error("Peer signed prekey signature is invalid");
  } catch {
    // SECURITY: malformed public material must not bubble decoder details into
    // the UI or logs; it is indistinguishable from an invalid authentication.
    throw new Error("Peer signed prekey signature is invalid");
  }
}

function validatePeerBundle(peer: KeyBundleAPI): void {
  if (!peer.user_id || !peer.device_id || !peer.identity_public_key || !peer.identity_dh_public_key || !peer.signed_pre_key || !peer.signed_pre_key_signature) {
    throw new Error("Incomplete peer key bundle");
  }
  if (!Number.isInteger(peer.signed_pre_key_id) || peer.signed_pre_key_id < 1) {
    throw new Error("Invalid peer signed prekey id");
  }
}

function validateLocalBundle(bundle: StoredDeviceKeyBundle): void {
  if (bundle.version !== 2 || !bundle.userId || !bundle.deviceId) throw new Error("Invalid local device key bundle");
}

function validateHeader(header: X3DHSessionHeader): void {
  if (header.version !== PAIRWISE_SESSION_PROTOCOL_VERSION || header.cipherSuite !== PAIRWISE_SESSION_CIPHER_SUITE) {
    throw new Error("Unsupported X3DH session protocol");
  }
  if (!header.sessionId || !header.senderUserId || !header.senderDeviceId || !header.senderIdentityDHPublicKey || !header.senderEphemeralPublicKey || !header.recipientUserId || !header.recipientDeviceId) {
    throw new Error("Incomplete X3DH session header");
  }
}

function validateState(state: PairwiseSessionState): void {
  if (state.version !== PAIRWISE_SESSION_PROTOCOL_VERSION || state.rootKey.length !== KEY_BYTES || state.sendChainKey.length !== KEY_BYTES || state.receiveChainKey.length !== KEY_BYTES) {
    throw new Error("Invalid pairwise session state");
  }
  if (state.dhSendPrivate.length !== KEY_BYTES || state.dhSendPublic.length !== KEY_BYTES || state.dhRecvPublic.length !== KEY_BYTES) {
    throw new Error("Invalid DH ratchet keys");
  }
  if (state.skippedMessageKeys.size > MAX_SKIPPED_KEYS) {
    throw new Error("Skipped message key limit exceeded");
  }
}

function canonicalHeader(header: X3DHSessionHeader): string {
  return JSON.stringify({
    cipherSuite: header.cipherSuite,
    recipientDeviceId: header.recipientDeviceId,
    recipientOneTimePreKeyId: header.recipientOneTimePreKeyId ?? null,
    recipientSignedPreKeyId: header.recipientSignedPreKeyId,
    recipientUserId: header.recipientUserId,
    senderDeviceId: header.senderDeviceId,
    senderDHPublicKey: header.senderDHPublicKey,
    senderEphemeralPublicKey: header.senderEphemeralPublicKey,
    senderIdentityDHPublicKey: header.senderIdentityDHPublicKey,
    senderUserId: header.senderUserId,
    sessionId: header.sessionId,
    version: header.version,
  });
}

function concatBytes(values: Uint8Array[]): Uint8Array {
  const out = new Uint8Array(values.reduce((total, value) => total + value.length, 0));
  let offset = 0;
  for (const value of values) {
    out.set(value, offset);
    offset += value.length;
  }
  return out;
}

function randomSessionID(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return bytesToBase64(bytes);
}
