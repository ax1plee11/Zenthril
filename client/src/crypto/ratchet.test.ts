import { describe, expect, it } from "vitest";
import { advanceRatchet } from "./ratchet";

describe("advanceRatchet", () => {
  it("derives separate chain and message keys", () => {
    const chainKey = new Uint8Array(32);
    chainKey[31] = 1;

    const step = advanceRatchet(chainKey);

    expect(step.newChainKey).toHaveLength(32);
    expect(step.messageKey).toHaveLength(32);
    expect(Array.from(step.newChainKey)).not.toEqual(Array.from(step.messageKey));
    expect(Array.from(step.messageKey)).not.toEqual(Array.from(chainKey));
  });

  it("is deterministic for the same chain key", () => {
    const chainKey = new Uint8Array(32).fill(7);

    const first = advanceRatchet(chainKey);
    const second = advanceRatchet(chainKey);

    expect(Array.from(first.newChainKey)).toEqual(Array.from(second.newChainKey));
    expect(Array.from(first.messageKey)).toEqual(Array.from(second.messageKey));
  });

  it("changes message keys on every step", () => {
    const first = advanceRatchet(new Uint8Array(32).fill(3));
    const second = advanceRatchet(first.newChainKey);

    expect(Array.from(second.messageKey)).not.toEqual(Array.from(first.messageKey));
  });

  it("rejects invalid chain key length", () => {
    expect(() => advanceRatchet(new Uint8Array(31))).toThrow("chainKey must be 32 bytes");
  });
});
