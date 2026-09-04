import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { x25519 } from "@noble/curves/ed25519.js";

const RATCHET_SALT = new TextEncoder().encode("zenthril-ratchet-v1");
const RATCHET_INFO = new TextEncoder().encode("chain");
const KEY_BYTES = 32;
const NONCE_BYTES = 12;

export interface RatchetStep {
  newChainKey: Uint8Array;
  messageKey: Uint8Array;
  messageNonce: Uint8Array;
}

/**
 * Advances a symmetric Double Ratchet chain using HKDF-SHA256.
 *
 * @security caller must zero messageKey after use.
 */
export function advanceRatchet(chainKey: Uint8Array): RatchetStep {
  if (chainKey.byteLength !== KEY_BYTES) {
    throw new Error("chainKey must be 32 bytes");
  }

  const output = hkdf(sha256, chainKey, RATCHET_SALT, RATCHET_INFO, KEY_BYTES * 2 + NONCE_BYTES);
  return {
    newChainKey: output.slice(0, KEY_BYTES),
    messageKey: output.slice(KEY_BYTES, KEY_BYTES * 2),
    messageNonce: output.slice(KEY_BYTES * 2, KEY_BYTES * 2 + NONCE_BYTES),
  };
}

/**
 * E2EE: Performs a full DH ratchet turn for forward secrecy and
 * post-compromise security.
 */
export function dhRatchetTurn(
  rootKey: Uint8Array,
  dhSendPrivate: Uint8Array,
  dhSendPublic: Uint8Array,
  newPeerDHPublic: Uint8Array,
): {
  newRootKey: Uint8Array;
  newRecvChainKey: Uint8Array;
  newSendChainKey: Uint8Array;
  newDHPrivate: Uint8Array;
  newDHPublic: Uint8Array;
} {
  if (rootKey.byteLength !== KEY_BYTES || dhSendPrivate.byteLength !== KEY_BYTES || dhSendPublic.byteLength !== KEY_BYTES || newPeerDHPublic.byteLength !== KEY_BYTES) {
    throw new Error("All DH ratchet inputs must be 32 bytes");
  }

  const dhOutput1 = x25519.getSharedSecret(dhSendPrivate, newPeerDHPublic);
  const rootStep1 = rootRatchet(rootKey, dhOutput1);
  let newRootKey = rootStep1.newRootKey;
  let newRecvChainKey = rootStep1.newChainKey;

  const newDH = x25519.keygen();
  const dhOutput2 = x25519.getSharedSecret(newDH.secretKey, newPeerDHPublic);
  const rootStep2 = rootRatchet(newRootKey, dhOutput2);
  newRootKey = rootStep2.newRootKey;
  const newSendChainKey = rootStep2.newChainKey;

  return {
    newRootKey,
    newRecvChainKey,
    newSendChainKey,
    newDHPrivate: newDH.secretKey,
    newDHPublic: newDH.publicKey,
  };
}

function rootRatchet(rootKey: Uint8Array, dhOutput: Uint8Array): { newRootKey: Uint8Array; newChainKey: Uint8Array } {
  const material = hkdf(sha256, dhOutput, rootKey, new TextEncoder().encode("zenthril-ratchet-v1:root"), KEY_BYTES * 2);
  return { newRootKey: material.slice(0, KEY_BYTES), newChainKey: material.slice(KEY_BYTES, KEY_BYTES * 2) };
}
