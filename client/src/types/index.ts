// Zenthril client data types.

import type { X3DHSessionHeader } from "../features/e2ee/pairwiseSession";

export interface RecipientKeyEnvelope {
  recipientUserId: string;
  recipientDeviceId: string;
  sessionId: string;
  ratchetCounter: number;
  dhPublicKey?: string;
  bootstrapHeader?: X3DHSessionHeader;
  payload: EncryptedPayload;
}

export interface EncryptedPayload {
  ciphertext: string;
  iv: string;
  tag: string;
  keyId: string;
  protocolVersion: number;
  channelId?: string;
  senderUserId?: string;
  senderDeviceId?: string;
  sessionId?: string;
  clientMessageId?: string;
  cipherSuite?: string;
  recipientEnvelopes?: RecipientKeyEnvelope[];
}

export interface User {
  id: string;
  username: string;
  publicKey: string;
  createdAt: string;
}

export interface Guild {
  id: string;
  name: string;
  ownerId: string;
  nodeId: string;
  channels: Channel[];
  members: Member[];
}

export interface Channel {
  id: string;
  guildId: string;
  name: string;
  type: "text" | "voice";
  position: number;
}

export interface Message {
  id: string;
  channelId: string;
  authorId: string;
  authorUsername: string;
  payload: EncryptedPayload;
  decryptedContent?: string;
  edited: boolean;
  deleted: boolean;
  createdAt: string;
}

export interface Member {
  userId: string;
  username: string;
  roleId?: string;
  joinedAt: string;
  banned: boolean;
  mutedUntil?: string;
}

export interface Role {
  id: string;
  guildId: string;
  name: string;
  level: number;
  permissions: number;
}
