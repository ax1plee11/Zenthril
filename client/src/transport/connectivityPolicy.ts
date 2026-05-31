export type ConnectivityMode = "off" | "balanced" | "strict";
export type TransportPreference = "auto" | "websocket" | "webrtc" | "custom";

export interface TransportPolicy {
  connectivityMode: ConnectivityMode;
  transportPreference: TransportPreference;
  websocketPadding: boolean;
  minPaddingBytes: number;
  maxPaddingBytes: number;
  jitterMs: number;
}

const POLICY_KEY = "zenthril_transport_policy_v1";

export const DEFAULT_TRANSPORT_POLICY: TransportPolicy = {
  connectivityMode: "off",
  transportPreference: "auto",
  websocketPadding: false,
  minPaddingBytes: 0,
  maxPaddingBytes: 96,
  jitterMs: 0,
};

export const BALANCED_CONNECTIVITY_POLICY: TransportPolicy = {
  connectivityMode: "balanced",
  transportPreference: "auto",
  websocketPadding: true,
  minPaddingBytes: 24,
  maxPaddingBytes: 256,
  jitterMs: 250,
};

export const STRICT_CONNECTIVITY_POLICY: TransportPolicy = {
  connectivityMode: "strict",
  transportPreference: "auto",
  websocketPadding: true,
  minPaddingBytes: 96,
  maxPaddingBytes: 768,
  jitterMs: 1200,
};

export function loadTransportPolicy(): TransportPolicy {
  try {
    const raw = localStorage.getItem(POLICY_KEY);
    if (!raw) return envPolicy();
    return normalizePolicy({ ...envPolicy(), ...(JSON.parse(raw) as Partial<TransportPolicy>) });
  } catch {
    return envPolicy();
  }
}

export function saveTransportPolicy(policy: TransportPolicy): void {
  const normalized = normalizePolicy(policy);
  localStorage.setItem(POLICY_KEY, JSON.stringify(normalized));
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent("zenthril:transport-policy-changed", { detail: normalized }));
  }
}

export function policyForConnectivityMode(mode: ConnectivityMode): TransportPolicy {
  if (mode === "strict") return STRICT_CONNECTIVITY_POLICY;
  if (mode === "balanced") return BALANCED_CONNECTIVITY_POLICY;
  return DEFAULT_TRANSPORT_POLICY;
}

export function paddingEnabledByPolicy(): boolean {
  return loadTransportPolicy().websocketPadding;
}

export async function applyTransportJitter(): Promise<void> {
  const { jitterMs } = loadTransportPolicy();
  if (jitterMs <= 0) return;
  const delay = randomInt(0, jitterMs);
  await new Promise(resolve => setTimeout(resolve, delay));
}

export function paddingBounds(): { min: number; max: number } {
  const policy = loadTransportPolicy();
  return {
    min: policy.minPaddingBytes,
    max: Math.max(policy.minPaddingBytes, policy.maxPaddingBytes),
  };
}

function envPolicy(): TransportPolicy {
  const envMode = import.meta.env.VITE_CONNECTIVITY_MODE as ConnectivityMode | undefined;
  const base = policyForConnectivityMode(envMode === "balanced" || envMode === "strict" ? envMode : "off");
  const envPadding = import.meta.env.VITE_WS_PADDING === "json-padding-v1";
  return normalizePolicy({
    ...base,
    websocketPadding: base.websocketPadding || envPadding,
  });
}

function normalizePolicy(policy: Partial<TransportPolicy>): TransportPolicy {
  const mode = policy.connectivityMode === "balanced" || policy.connectivityMode === "strict"
    ? policy.connectivityMode
    : "off";
  const minPaddingBytes = clamp(Math.floor(policy.minPaddingBytes ?? 0), 0, 4096);
  const maxPaddingBytes = clamp(Math.floor(policy.maxPaddingBytes ?? 96), minPaddingBytes, 8192);
  return {
    connectivityMode: mode,
    transportPreference: policy.transportPreference ?? "auto",
    websocketPadding: Boolean(policy.websocketPadding),
    minPaddingBytes,
    maxPaddingBytes,
    jitterMs: clamp(Math.floor(policy.jitterMs ?? 0), 0, 5000),
  };
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, Number.isFinite(value) ? value : min));
}

function randomInt(min: number, max: number): number {
  const range = Math.max(1, max - min + 1);
  const bytes = new Uint32Array(1);
  crypto.getRandomValues(bytes);
  return min + ((bytes[0] ?? 0) % range);
}
