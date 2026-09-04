import { api } from "../../api";
import type { EncryptedPayloadAPI, RecipientKeyEnvelopeAPI } from "../../api";
import { decrypt, encrypt } from "../../crypto";
import type { EncryptedPayload } from "../../types";
import type { RecipientKeyEnvelope } from "../../types";
import { buildMessageAADInput, MissingDeviceKeyBundleError } from "./messageAAD";
import { loadDeviceKeyBundle, storeDeviceKeyBundle } from "./deviceKeyStore";
import {
  acceptPairwiseSession,
  findPairwiseSessionForPeer,
  importRatchetMessageKey,
  initiatePairwiseSession,
  loadPairwiseSession,
  nextReceiveMessageKey,
  nextSendMessageKey,
  performDHRatchetTurn,
  savePairwiseSession,
  type RatchetedMessageKey,
  type X3DHSessionHeader,
} from "./pairwiseSession";
import { base64ToBytes, bytesToBase64 } from "./encoding";

export interface PreparedChannelMessage {
  payload: EncryptedPayload;
  persist(): Promise<void>;
}

// SECURITY: creates one random content key per message and wraps it separately
// for each recipient device. The server receives only ciphertext and envelopes.
// E2EE: full Double Ratchet with DH ratchet turns and session distribution.
export async function prepareChannelMessage(
  plaintext: string,
  channelId: string,
  userId: string,
): Promise<PreparedChannelMessage> {
  const local = await loadDeviceKeyBundle(userId);
  if (!local) throw new MissingDeviceKeyBundleError(userId);
  const contentKey = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"],
  );
  const contentKeyRaw = new Uint8Array(await crypto.subtle.exportKey("raw", contentKey));
  try {
    const aad = await buildMessageAADInput(channelId, userId);
    const payload = await encrypt(plaintext, contentKey, aad);
    const recipients = await api.messages.recipients(channelId);
    let updated = local;
    const envelopes: RecipientKeyEnvelopeAPI[] = [];

    for (const recipient of recipients.recipients) {
      if (recipient.device_id === local.deviceId) continue;
      let state = findPairwiseSessionForPeer(updated, recipient.user_id, recipient.device_id);
      let bootstrapHeader: X3DHSessionHeader | undefined;
      if (!state) {
        const peer = await api.keyBundles.claim(recipient.user_id, recipient.device_id);
        const initiated = initiatePairwiseSession(updated, peer);
        state = initiated.state;
        bootstrapHeader = initiated.header;
      }
      const step = nextSendMessageKey(state);
      const { key: wrappingKey, nonce } = await importRatchetMessageKey(step.messageKey, step.messageNonce);
      const wrapped = await encrypt(bytesToBase64(contentKeyRaw), wrappingKey, {
        channelId,
        senderUserId: userId,
        senderDeviceId: local.deviceId,
        sessionId: state.sessionId,
        clientMessageId: payload.clientMessageId ?? aad.clientMessageId,
        nonce: Array.from(nonce),
      });
      envelopes.push({
        recipient_user_id: recipient.user_id,
        recipient_device_id: recipient.device_id,
        session_id: state.sessionId,
        ratchet_counter: step.counter,
        dh_public_key: bytesToBase64(state.dhSendPublic),
        ...(bootstrapHeader ? { bootstrap_header: bootstrapHeader } : {}),
        payload: toAPI(wrapped),
      });
      updated = savePairwiseSession(updated, step.state);
    }
    payload.recipientEnvelopes = envelopes.map(fromAPIEnvelope);
    return { payload, persist: () => storeDeviceKeyBundle(updated) };
  } finally {
    contentKeyRaw.fill(0);
  }
}

// SECURITY: only the envelope addressed to the local device is processed.
// Invalid envelopes leave the message encrypted and do not advance local state.
// E2EE: DH ratchet turn on new peer DH public key + skipped message keys.
export async function decryptChannelMessage(
  payload: EncryptedPayload,
  userId: string,
): Promise<string | null> {
  const local = await loadDeviceKeyBundle(userId);
  if (!local) return null;
  const envelope = payload.recipientEnvelopes?.find(item => item.recipientDeviceId === local.deviceId);
  if (!envelope) return null;

  let bundle = local;
  let state = loadPairwiseSession(bundle, envelope.sessionId);
  if (envelope.bootstrapHeader) {
    const accepted = acceptPairwiseSession(bundle, envelope.bootstrapHeader);
    bundle = accepted.updatedLocalBundle;
    state = accepted.state;
  }
  if (!state) return null;

  // E2EE: perform DH ratchet turn if peer rotated DH keys
  if (envelope.dhPublicKey && !bytesEqual(envelope.dhPublicKey, state.dhRecvPublic)) {
    state = performDHRatchetTurn(state, base64ToBytes(envelope.dhPublicKey));
  }

  let step: RatchetedMessageKey;
  try {
    step = nextReceiveMessageKey(state, envelope.ratchetCounter);
  } catch {
    return null;
  }
  const { key: wrappingKey, nonce } = await importRatchetMessageKey(step.messageKey, step.messageNonce);
  const wrappedKeyText = await decrypt(envelope.payload, wrappingKey, undefined, { nonce: Array.from(nonce) });
  const contentKeyBytes = base64ToBytes(wrappedKeyText);
  if (contentKeyBytes.length !== 32) throw new Error("Invalid encrypted content key");
  try {
    const contentKey = await crypto.subtle.importKey(
      "raw", toArrayBuffer(contentKeyBytes), { name: "AES-GCM", length: 256 }, false, ["decrypt"],
    );
    const text = await decrypt(payload, contentKey);
    await storeDeviceKeyBundle(savePairwiseSession(bundle, step.state));
    return text;
  } finally {
    contentKeyBytes.fill(0);
  }
}

function bytesEqual(a: string, b: Uint8Array): boolean {
  const aBytes = base64ToBytes(a);
  if (aBytes.length !== b.length) return false;
  for (let i = 0; i < aBytes.length; i++) {
    if (aBytes[i] !== b[i]) return false;
  }
  aBytes.fill(0);
  return true;
}

function toAPI(payload: EncryptedPayload): EncryptedPayloadAPI {
  const out: EncryptedPayloadAPI = {
    ciphertext: payload.ciphertext, iv: payload.iv, key_id: payload.keyId,
    tag: payload.tag, protocol_version: payload.protocolVersion,
  };
  if (payload.channelId) out.channel_id = payload.channelId;
  if (payload.senderUserId) out.sender_user_id = payload.senderUserId;
  if (payload.senderDeviceId) out.sender_device_id = payload.senderDeviceId;
  if (payload.sessionId) out.session_id = payload.sessionId;
  if (payload.clientMessageId) out.client_message_id = payload.clientMessageId;
  if (payload.cipherSuite) out.cipher_suite = payload.cipherSuite;
  return out;
}

function fromAPIEnvelope(value: RecipientKeyEnvelopeAPI): RecipientKeyEnvelope {
  return {
    recipientUserId: value.recipient_user_id,
    recipientDeviceId: value.recipient_device_id,
    sessionId: value.session_id,
    ratchetCounter: value.ratchet_counter,
    dhPublicKey: value.dh_public_key ?? "",
    ...(value.bootstrap_header ? { bootstrapHeader: value.bootstrap_header as X3DHSessionHeader } : {}),
    payload: fromAPI(value.payload),
  };
}

function fromAPI(payload: EncryptedPayloadAPI): EncryptedPayload {
  const out: EncryptedPayload = {
    ciphertext: payload.ciphertext, iv: payload.iv, keyId: payload.key_id,
    tag: payload.tag, protocolVersion: payload.protocol_version,
  };
  if (payload.channel_id) out.channelId = payload.channel_id;
  if (payload.sender_user_id) out.senderUserId = payload.sender_user_id;
  if (payload.sender_device_id) out.senderDeviceId = payload.sender_device_id;
  if (payload.session_id) out.sessionId = payload.session_id;
  if (payload.client_message_id) out.clientMessageId = payload.client_message_id;
  if (payload.cipher_suite) out.cipherSuite = payload.cipher_suite;
  return out;
}

function toArrayBuffer(value: Uint8Array): ArrayBuffer {
  const out = new Uint8Array(value.length);
  out.set(value);
  return out.buffer;
}
