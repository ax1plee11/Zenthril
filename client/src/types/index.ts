// Типы данных Veltrix (клиент)

export interface EncryptedPayload {
  ciphertext: string; // base64
  iv: string;         // base64, 12 байт для GCM
  tag: string;        // base64, 16 байт auth tag
  keyId: string;      // идентификатор версии ключа
}

export interface User {
  id: string;
  username: string;
  publicKey: string;
  createdAt: string; // ISO 8601
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
  decryptedContent?: string; // заполняется после дешифрования на клиенте
  edited: boolean;
  deleted: boolean;
  createdAt: string; // ISO 8601
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
