export interface SerializedKeyPair {
  publicKey: string;
  secretKey: string;
}

export interface StoredOneTimePreKey {
  keyId: number;
  publicKey: string;
  secretKey: string;
}

// SECURITY: this state contains ratchet secrets and is persisted only through
// the existing secure device-bundle storage adapter (OS keychain in Tauri).
export interface StoredPairwiseSession {
  version: 1;
  sessionId: string;
  peerUserId: string;
  peerDeviceId: string;
  rootKey: string;
  sendChainKey: string;
  receiveChainKey: string;
  sendCounter: number;
  receiveCounter: number;
  dhSendPublic: string;
  dhRecvPublic: string;
  previousCounter: number;
  skippedMessageKeys: Record<number, { key: string; nonce: string; counter: number }>;
}

export interface StoredDeviceKeyBundle {
  version: 2;
  userId: string;
  deviceId: string;
  deviceName: string;
  identitySigningKey: SerializedKeyPair;
  // SECURITY: X25519 identity key used only for X3DH. It is deliberately
  // separate from the Ed25519 signing identity above.
  identityDHKey: SerializedKeyPair;
  signedPreKeyId: number;
  signedPreKey: SerializedKeyPair;
  signedPreKeySignature: string;
  oneTimePreKeys: StoredOneTimePreKey[];
  createdAt: string;
  registeredAt?: string;
	backendFingerprint?: string;
	pairwiseSessions?: Record<string, StoredPairwiseSession>;
}

export interface RegisterDeviceRequest {
  device_id: string;
  name: string;
  identity_public_key: string;
  identity_dh_public_key: string;
  signed_pre_key_id: number;
  signed_pre_key: string;
  signed_pre_key_signature: string;
  one_time_prekeys: Array<{
    key_id: number;
    public_key: string;
  }>;
}

export interface DeviceAPI {
  device_id: string;
  user_id: string;
  name: string;
  identity_public_key: string;
  identity_dh_public_key: string;
  signed_pre_key_id: number;
  signed_pre_key: string;
  signed_pre_key_signature: string;
  fingerprint: string;
  trust_state: "unverified" | "verified" | "revoked";
  one_time_prekey_count: number;
  created_at: string;
  updated_at: string;
  last_seen_at: string;
  revoked_at?: string | null;
}

export interface KeyBundleAPI {
  user_id: string;
  device_id: string;
  identity_public_key: string;
  identity_dh_public_key: string;
  signed_pre_key_id: number;
  signed_pre_key: string;
  signed_pre_key_signature: string;
  one_time_prekey?: {
    key_id: number;
    public_key: string;
  };
  fingerprint: string;
}
