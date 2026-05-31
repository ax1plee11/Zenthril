import { describe, expect, it, beforeEach } from "vitest";
import {
  BALANCED_CONNECTIVITY_POLICY,
  DEFAULT_TRANSPORT_POLICY,
  loadTransportPolicy,
  policyForConnectivityMode,
  saveTransportPolicy,
} from "./connectivityPolicy";

describe("transport connectivity policy", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("maps connectivity modes to policy presets", () => {
    expect(policyForConnectivityMode("off")).toEqual(DEFAULT_TRANSPORT_POLICY);
    expect(policyForConnectivityMode("balanced")).toEqual(BALANCED_CONNECTIVITY_POLICY);
    expect(policyForConnectivityMode("strict").maxPaddingBytes).toBeGreaterThan(
      BALANCED_CONNECTIVITY_POLICY.maxPaddingBytes,
    );
  });

  it("persists selected connectivity policy", () => {
    saveTransportPolicy(policyForConnectivityMode("balanced"));

    expect(loadTransportPolicy().connectivityMode).toBe("balanced");
    expect(loadTransportPolicy().websocketPadding).toBe(true);
  });
});
