import { afterEach, describe, expect, it, vi } from "vitest";
import { createWebRTCConfig, loadConfiguredIceServers, relayCapableServers } from "./icePolicy";

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

  it("loads TURN servers from environment JSON", () => {
    vi.stubEnv("VITE_WEBRTC_ICE_SERVERS", JSON.stringify([
      { urls: "turn:turn.example.com:3478", username: "u", credential: "p" },
    ]));

    expect(loadConfiguredIceServers()).toEqual([
      { urls: "turn:turn.example.com:3478", username: "u", credential: "p" },
    ]);
  });

  it("filters STUN-only servers out of relay-only configs", () => {
    vi.stubEnv("VITE_WEBRTC_RELAY_ONLY", "true");
    const config = createWebRTCConfig([
      { urls: "stun:stun.l.google.com:19302" },
      { urls: "turns:turn.example.com:5349", username: "u", credential: "p" },
    ]);

    expect(config.iceTransportPolicy).toBe("relay");
    expect(config.iceServers).toEqual([
      { urls: "turns:turn.example.com:5349", username: "u", credential: "p" },
    ]);
  });

  it("detects TURN-capable servers with string or array urls", () => {
    expect(relayCapableServers([
      { urls: ["stun:stun.example.com:19302", "turn:turn.example.com:3478"] },
      { urls: "stun:stun2.example.com:19302" },
    ])).toEqual([
      { urls: ["stun:stun.example.com:19302", "turn:turn.example.com:3478"] },
    ]);
  });
});
