import { describe, expect, it } from "vitest";
import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";

describe("hkdf determinism", () => {
  it("produces identical output for identical inputs", () => {
    const ikm = new Uint8Array(32).fill(0x42);
    const salt = new TextEncoder().encode("zenthril-ratchet-v1");
    const info = new TextEncoder().encode("chain");
    const a = hkdf(sha256, ikm, salt, info, 76);
    const b = hkdf(sha256, ikm, salt, info, 76);
    expect(Array.from(a)).toEqual(Array.from(b));
  });
});
