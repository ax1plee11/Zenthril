import type { CryptoAADContextInput } from "../../crypto";
import { loadDeviceKeyBundle } from "./deviceKeyStore";

export class MissingDeviceKeyBundleError extends Error {
  constructor(userId: string) {
    super(`Missing registered E2EE device key bundle for user ${userId}`);
    this.name = "MissingDeviceKeyBundleError";
  }
}

export function randomClientMessageId(): string {
  return crypto.randomUUID?.() ?? `msg_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

export async function buildMessageAADInput(
  channelId: string,
  currentUserId: string,
): Promise<CryptoAADContextInput> {
  const bundle = await loadDeviceKeyBundle(currentUserId);
  if (!bundle?.deviceId) {
    // WEAKNESS FIXED: protocol-v2 AAD must be bound to a real registered local
    // device. A fake sender device hides setup failures and weakens auditing.
    throw new MissingDeviceKeyBundleError(currentUserId);
  }
  return {
    channelId,
    senderUserId: currentUserId,
    senderDeviceId: bundle.deviceId,
    sessionId: `channel:${channelId}`,
    clientMessageId: randomClientMessageId(),
  };
}
