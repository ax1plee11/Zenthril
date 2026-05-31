import { describe, expect, it } from "vitest";
import {
  BALANCED_STEALTH_POLICY,
  DEFAULT_TRANSPORT_POLICY,
  loadTransportPolicy,
  policyForStealthMode,
  saveTransportPolicy,
} from "./policy";

describe("transport policy", () => {
  it("maps stealth modes to policy presets", () => {
    expect(policyForStealthMode("off")).toEqual(DEFAULT_TRANSPORT_POLICY);
    expect(policyForStealthMode("balanced")).toEqual(BALANCED_STEALTH_POLICY);
    expect(policyForStealthMode("strict").maxPaddingBytes).toBeGreaterThan(
      BALANCED_STEALTH_POLICY.maxPaddingBytes,
    );
  });

  it("persists policy in local storage", () => {
    saveTransportPolicy(policyForStealthMode("balanced"));

    expect(loadTransportPolicy().stealthMode).toBe("balanced");
    expect(loadTransportPolicy().websocketCamouflage).toBe(true);
  });
});
