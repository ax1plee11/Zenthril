import { afterEach, describe, expect, it, vi } from "vitest";
import { createWebRTCConfig } from "./icePolicy";

describe("WebRTC ICE privacy policy", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("uses relay-only ICE when explicitly enabled", () => {
    vi.stubEnv("VITE_WEBRTC_RELAY_ONLY", "true");

    expect(createWebRTCConfig().iceTransportPolicy).toBe("relay");
  });

  it("allows direct ICE only when explicitly configured off", () => {
    vi.stubEnv("VITE_WEBRTC_RELAY_ONLY", "false");

    expect(createWebRTCConfig().iceTransportPolicy).toBe("all");
  });
});
