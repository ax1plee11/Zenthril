import { paddingEnabledByPolicy, paddingBounds } from "./connectivityPolicy";

export interface PaddedEnvelope {
  type: "transport.frame";
  version: 1;
  mode: "json-padding-v1";
  payload: string;
  padding: string;
  cover: "json";
}

const FRAME_VERSION = 1;

export function paddingEnabled(): boolean {
  return paddingEnabledByPolicy();
}

export function encodePaddedFrame(event: Record<string, unknown>): string {
  const payload = btoa(JSON.stringify(event));
  const paddingBytes = new Uint8Array(randomPaddingLength());
  crypto.getRandomValues(paddingBytes);
  const envelope: PaddedEnvelope = {
    type: "transport.frame",
    version: FRAME_VERSION,
    mode: "json-padding-v1",
    payload,
    padding: btoa(String.fromCharCode(...paddingBytes)),
    cover: "json",
  };
  // CONNECTIVITY: optional JSON padding is an outage-testing transport feature.
  // It is not cryptographic secrecy and does not make traffic invisible.
  return JSON.stringify(envelope);
}

export function decodePaddedFrame(value: string): Record<string, unknown> {
  const parsed = JSON.parse(value) as Partial<PaddedEnvelope> | Record<string, unknown>;
  if (parsed.type !== "transport.frame") {
    return parsed as Record<string, unknown>;
  }

  const envelope = parsed as Partial<PaddedEnvelope>;
  if (envelope.version !== FRAME_VERSION || envelope.mode !== "json-padding-v1" || !envelope.payload) {
    throw new Error("Unsupported padded frame");
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
