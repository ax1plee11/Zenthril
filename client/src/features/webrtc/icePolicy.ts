export const DEFAULT_ICE_SERVERS: RTCIceServer[] = [
  { urls: "stun:stun.l.google.com:19302" },
  { urls: "stun:stun1.l.google.com:19302" },
];

function envFlag(name: string): boolean | null {
  const value = import.meta.env[name]?.toString().trim().toLowerCase();
  if (!value) return null;
  if (["1", "true", "yes", "on"].includes(value)) return true;
  if (["0", "false", "no", "off"].includes(value)) return false;
  return null;
}

export function shouldUseRelayOnly(): boolean {
  const configured = envFlag("VITE_WEBRTC_RELAY_ONLY");
  if (configured !== null) return configured;
  return import.meta.env.PROD;
}

export function createWebRTCConfig(
  iceServers: RTCIceServer[] = DEFAULT_ICE_SERVERS,
): RTCConfiguration {
  const relayOnly = shouldUseRelayOnly();
  return {
    iceServers,
    // SECURITY-HARDENING: production builds default to relay-only ICE to reduce
    // local/public IP exposure. Operators must configure TURN for reliable calls.
    iceTransportPolicy: relayOnly ? "relay" : "all",
  };
}
