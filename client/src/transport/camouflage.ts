export interface CamouflageEnvelope {
  type: "transport.frame";
  version: 1;
  mode: "json-padding-v1";
  payload: string;
  padding: string;
  cover: "browser-json";
}

import { camouflageEnabledByPolicy, paddingBounds } from "./policy";

const FRAME_VERSION = 1;

export function camouflageEnabled(): boolean {
  return camouflageEnabledByPolicy();
}

export function encodeCamouflageFrame(event: Record<string, unknown>): string {
  const payload = btoa(JSON.stringify(event));
  const paddingBytes = new Uint8Array(randomPaddingLength());
  crypto.getRandomValues(paddingBytes);
  const envelope: CamouflageEnvelope = {
    type: "transport.frame",
    version: FRAME_VERSION,
    mode: "json-padding-v1",
    payload,
    padding: btoa(String.fromCharCode(...paddingBytes)),
    cover: "browser-json",
  };
  // ANTI-DETECTION: this pads application frames to reduce simple payload-size fingerprints.
  // It is traffic camouflage only; it is not cryptographic secrecy or DPI invisibility.
  return JSON.stringify(envelope);
}

export function decodeCamouflageFrame(value: string): Record<string, unknown> {
  const parsed = JSON.parse(value) as Partial<CamouflageEnvelope> | Record<string, unknown>;
  if (parsed.type !== "transport.frame") {
    return parsed as Record<string, unknown>;
  }

  const envelope = parsed as Partial<CamouflageEnvelope>;
  if (envelope.version !== FRAME_VERSION || envelope.mode !== "json-padding-v1" || !envelope.payload) {
    throw new Error("Unsupported camouflage frame");
  }
  return JSON.parse(atob(envelope.payload)) as Record<string, unknown>;
}

function randomPaddingLength(): number {
  const { min, max } = paddingBounds();
  const range = Math.max(1, max - min + 1);
  const bytes = new Uint32Array(1);
  crypto.getRandomValues(bytes);
  return min + ((bytes[0] ?? 0) % range);
}
