export interface SerializedKeyPair {
  publicKey: string;
  secretKey: string;
}

export interface StoredOneTimePreKey {
  keyId: number;
  publicKey: string;
  secretKey: string;
}

export interface StoredDeviceKeyBundle {
  version: 1;
  userId: string;
  deviceId: string;
  deviceName: string;
  identitySigningKey: SerializedKeyPair;
  signedPreKeyId: number;
  signedPreKey: SerializedKeyPair;
  signedPreKeySignature: string;
  oneTimePreKeys: StoredOneTimePreKey[];
  createdAt: string;
  registeredAt?: string;
  backendFingerprint?: string;
}

export interface RegisterDeviceRequest {
  device_id: string;
  name: string;
  identity_public_key: string;
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
  signed_pre_key_id: number;
  signed_pre_key: string;
  signed_pre_key_signature: string;
  one_time_prekey?: {
    key_id: number;
    public_key: string;
  };
  fingerprint: string;
}
