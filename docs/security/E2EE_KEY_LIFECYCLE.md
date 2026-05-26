# E2EE key lifecycle

## Key types

| Key | Owner | Stored on server | Rotation | Purpose |
|---|---|---|---|---|
| Identity key | Device | Public only | Rarely | Long-term device identity |
| Signed prekey | Device | Public only | 7–30 days | Async session bootstrap |
| One-time prekey | Device | Public only, consumed once | Replenished on use | Forward secrecy at session start |
| Ephemeral key | Sender | No | Per session | X3DH handshake |
| Root key | Client only | No | Every DH ratchet step | Double Ratchet root chain |
| Message key | Client only | No | Per message | AES-256-GCM encryption |

## Session establishment (X3DH)

```
DH1 = DH(IK_sender,  SPK_receiver)
DH2 = DH(EK_sender,  IK_receiver)
DH3 = DH(EK_sender,  SPK_receiver)
DH4 = DH(EK_sender,  OPK_receiver)  // optional
root_key = HKDF(DH1 || DH2 || DH3 || DH4)
```

## Message encryption (Double Ratchet)

```
chain_key, message_key = KDF(chain_key)
ciphertext = AES-256-GCM(message_key, plaintext)
delete(message_key)  // immediately after use
```

## Key deletion policy
- Message keys: deleted immediately after encryption/decryption.
- One-time prekeys: deleted on server after consumption.
- Ephemeral keys: deleted after session root key is derived.
- Root/chain keys: never leave the device.
