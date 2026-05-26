import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";

const RATCHET_SALT = new TextEncoder().encode("zenthril.double-ratchet.v1");
const RATCHET_INFO = new TextEncoder().encode("chain+message");
const KEY_BYTES = 32;

export interface RatchetStep {
  newChainKey: Uint8Array;
  messageKey: Uint8Array;
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

  const output = hkdf(sha256, chainKey, RATCHET_SALT, RATCHET_INFO, KEY_BYTES * 2);
  return {
    newChainKey: output.slice(0, KEY_BYTES),
    messageKey: output.slice(KEY_BYTES, KEY_BYTES * 2),
  };
}
