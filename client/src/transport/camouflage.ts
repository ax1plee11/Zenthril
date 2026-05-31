export interface CamouflageEnvelope {
  type: "transport.frame";
  version: 1;
  mode: "json-padding-v1";
  payload: string;
  padding: string;
}

const FRAME_VERSION = 1;
const MAX_PADDING_BYTES = 96;

export function camouflageEnabled(): boolean {
  return import.meta.env.VITE_WS_CAMOUFLAGE === "json-padding-v1";
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
  };
  // ANTI-BLOCKING: this is traffic camouflage only; it is not cryptographic secrecy.
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
  const byte = new Uint8Array(1);
  crypto.getRandomValues(byte);
  return (byte[0] ?? 0) % (MAX_PADDING_BYTES + 1);
}
