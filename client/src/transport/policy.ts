export type StealthMode = "off" | "balanced" | "strict";
export type TransportPreference = "auto" | "websocket" | "webrtc" | "tor" | "bridge";

export interface TransportPolicy {
  stealthMode: StealthMode;
  transportPreference: TransportPreference;
  websocketCamouflage: boolean;
  minPaddingBytes: number;
  maxPaddingBytes: number;
  jitterMs: number;
  allowDomainFronting: boolean;
  frontingHost?: string;
}

const POLICY_KEY = "zenthril_transport_policy_v1";

export const DEFAULT_TRANSPORT_POLICY: TransportPolicy = {
  stealthMode: "off",
  transportPreference: "auto",
  websocketCamouflage: false,
  minPaddingBytes: 0,
  maxPaddingBytes: 96,
  jitterMs: 0,
  allowDomainFronting: false,
};

export const BALANCED_STEALTH_POLICY: TransportPolicy = {
  stealthMode: "balanced",
  transportPreference: "auto",
  websocketCamouflage: true,
  minPaddingBytes: 24,
  maxPaddingBytes: 256,
  jitterMs: 250,
  allowDomainFronting: false,
};

export const STRICT_STEALTH_POLICY: TransportPolicy = {
  stealthMode: "strict",
  transportPreference: "auto",
  websocketCamouflage: true,
  minPaddingBytes: 96,
  maxPaddingBytes: 768,
  jitterMs: 1200,
  allowDomainFronting: false,
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

export function policyForStealthMode(mode: StealthMode): TransportPolicy {
  if (mode === "strict") return STRICT_STEALTH_POLICY;
  if (mode === "balanced") return BALANCED_STEALTH_POLICY;
  return DEFAULT_TRANSPORT_POLICY;
}

export function camouflageEnabledByPolicy(): boolean {
  return loadTransportPolicy().websocketCamouflage;
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
  const envMode = import.meta.env.VITE_STEALTH_MODE as StealthMode | undefined;
  const base = policyForStealthMode(envMode === "balanced" || envMode === "strict" ? envMode : "off");
  const envCamouflage = import.meta.env.VITE_WS_CAMOUFLAGE === "json-padding-v1";
  return normalizePolicy({
    ...base,
    websocketCamouflage: base.websocketCamouflage || envCamouflage,
    allowDomainFronting: import.meta.env.VITE_ALLOW_DOMAIN_FRONTING === "true",
    frontingHost: import.meta.env.VITE_FRONTING_HOST?.trim() || undefined,
  });
}

function normalizePolicy(policy: TransportPolicy): TransportPolicy {
  const minPaddingBytes = clamp(Math.floor(policy.minPaddingBytes), 0, 4096);
  const maxPaddingBytes = clamp(Math.floor(policy.maxPaddingBytes), minPaddingBytes, 8192);
  return {
    ...policy,
    minPaddingBytes,
    maxPaddingBytes,
    jitterMs: clamp(Math.floor(policy.jitterMs), 0, 5000),
    allowDomainFronting: Boolean(policy.allowDomainFronting && policy.frontingHost),
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
