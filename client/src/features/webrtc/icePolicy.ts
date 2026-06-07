export const DEFAULT_ICE_SERVERS: RTCIceServer[] = [
  { urls: "stun:stun.l.google.com:19302" },
  { urls: "stun:stun1.l.google.com:19302" },
];

const ICE_SERVERS_ENV = "VITE_WEBRTC_ICE_SERVERS";

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

export function loadConfiguredIceServers(): RTCIceServer[] {
  const raw = import.meta.env[ICE_SERVERS_ENV]?.toString().trim();
  if (!raw) return DEFAULT_ICE_SERVERS;

  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return DEFAULT_ICE_SERVERS;
    const servers = parsed.filter(isValidIceServer);
    return servers.length > 0 ? servers : DEFAULT_ICE_SERVERS;
  } catch {
    return DEFAULT_ICE_SERVERS;
  }
}

export function relayCapableServers(iceServers: RTCIceServer[]): RTCIceServer[] {
  return iceServers.filter(server => serverUrls(server).some(isTurnURL));
}

export function createWebRTCConfig(
  iceServers: RTCIceServer[] = loadConfiguredIceServers(),
): RTCConfiguration {
  const relayOnly = shouldUseRelayOnly();
  const selectedServers = relayOnly ? relayCapableServers(iceServers) : iceServers;
  return {
    // SECURITY-HARDENING: production relay-only mode uses TURN-capable servers
    // only. Public STUN entries are useful in dev, but they are not sufficient
    // for privacy-preserving production voice/P2P.
    iceServers: selectedServers,
    iceTransportPolicy: relayOnly ? "relay" : "all",
  };
}

function isValidIceServer(value: unknown): value is RTCIceServer {
  if (!value || typeof value !== "object") return false;
  const urls = (value as RTCIceServer).urls;
  if (typeof urls === "string") return urls.trim() !== "";
  return Array.isArray(urls) && urls.some(url => typeof url === "string" && url.trim() !== "");
}

function serverUrls(server: RTCIceServer): string[] {
  if (typeof server.urls === "string") return [server.urls];
  return server.urls;
}

function isTurnURL(url: string): boolean {
  return url.toLowerCase().startsWith("turn:") || url.toLowerCase().startsWith("turns:");
}
