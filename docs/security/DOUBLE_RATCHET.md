# Double Ratchet Implementation

## Status: Alpha - Not Audited

This document describes the current Double Ratchet implementation in Zenthril. The implementation provides forward secrecy and post-compromise security through a combination of symmetric and asymmetric ratcheting.

**SECURITY WARNING:** This implementation has not been externally audited and should not be considered production-ready for sensitive communications.

## Components

### 1. X3DH Key Agreement

The Extended Triple Diffie-Hellman (X3DH) protocol establishes an initial shared secret between two parties:

- **DH1** = DH(IKa, SPKb) - Identity key to signed prekey
- **DH2** = DH(EKa, IKb) - Ephemeral to identity
- **DH3** = DH(EKa, SPKb) - Ephemeral to signed prekey
- **DH4** = DH(EKa, OPKb) - Ephemeral to one-time prekey (if available)

The shared secret SK is derived using HKDF-SHA256 from the concatenation of all DH outputs.

### 2. Symmetric Ratchet

The symmetric ratchet (also called the sending/receiving chain) advances with each message:

```
Message Key, Next Chain Key = KDF(Current Chain Key, Counter)
```

Features:
- **Forward Secrecy:** Each message key is immediately deleted after use
- **Out-of-Order Delivery:** Skipped message keys are stored (up to 2000 max)
- **Replay Protection:** Used message keys cannot be reused
- **Counter Exhaustion Protection:** Prevents counter overflow attacks

### 3. DH Ratchet (Asymmetric Ratchet)

The DH ratchet provides post-compromise security by rotating Diffie-Hellman keys:

When receiving a message with a new DH public key:
1. Perform DH with the new peer public key
2. Derive new root key and receive chain key via HKDF
3. Generate a new DH key pair
4. Perform DH with peer's public key using new private key
5. Derive new root key and send chain key via HKDF

This double ratchet step ensures that both sending and receiving chains are refreshed.

### 4. Message Encryption

Each message is encrypted with AES-256-GCM using:
- **Key:** Derived from symmetric ratchet (32 bytes)
- **Nonce:** Derived from symmetric ratchet (12 bytes)
- **AAD:** Additional authenticated data (protocol version, channel ID, user IDs, etc.)

Message structure:
```
Message Header:
  - DH Public Key (32 bytes)
  - Previous Counter (4 bytes)
  - Message Counter (4 bytes)
  
Ciphertext:
  - AES-GCM encrypted plaintext + auth tag
```

## Implementation Files

- `x25519.go` - X3DH key agreement and DH operations
- `ratchet.go` - Symmetric and DH ratchet state management
- `message.go` - Encrypt/Decrypt operations
- `protocol.go` - X3DH service and session initialization

## Security Properties

### Provided

✅ **Forward Secrecy:** Past messages remain secure if current keys are compromised
✅ **Post-Compromise Security:** Future messages become secure after a DH ratchet turn
✅ **Out-of-Order Delivery:** Messages can arrive out of order (bounded gap)
✅ **Replay Protection:** Replayed messages are rejected
✅ **Authentication:** AES-GCM provides authenticated encryption

### Not Yet Implemented

❌ **Header Encryption:** Message headers (DH keys, counters) are not encrypted
❌ **Deniability:** No formal deniable authentication
❌ **Session Healing:** No automatic recovery from persistent desync
❌ **External Audit:** No independent security review
❌ **Production Multi-Device:** Device session distribution is incomplete

## Key Derivation

All key derivation uses HKDF-SHA256 with domain separation:

```
Root Key Derivation: HKDF(DH output, root key, "zenthril-ratchet-v1:root")
Initial State: HKDF(X3DH SK, nil, "zenthril-ratchet-v1:init:" || session info)
Chain Keys: HKDF(chain key, nil, "zenthril-ratchet-v1:chain")
```

## Limitations

1. **Alpha Status:** Not production-ready, no external audit
2. **Bounded Skipped Keys:** Max 2000 skipped messages per ratchet state
3. **Counter Limits:** 2^32 messages per chain before ratchet turn required
4. **No Header Encryption:** Metadata leakage in message headers
5. **Server Sees Metadata:** Server observes message timing and sizes
6. **Group E2EE Incomplete:** Double Ratchet is designed for pairwise sessions

## Test Coverage

Comprehensive tests cover:
- X3DH key agreement and signature verification
- Symmetric ratchet advancement and counter management
- DH ratchet turns and key rotation
- Out-of-order message delivery
- Replay protection
- Skipped message key handling
- AAD validation
- Counter exhaustion protection

## Future Work

1. **Header Encryption:** Encrypt message headers for metadata protection
2. **Session Healing:** Automatic recovery from persistent desync states
3. **Optimized Storage:** More efficient skipped key storage
4. **Group Protocols:** Sender Keys or MLS for multi-party channels
5. **External Audit:** Independent security review
6. **Test Vectors:** Interoperability test vectors against Signal

## References

- [Signal Protocol Specification](https://signal.org/docs/)
- [Double Ratchet Algorithm](https://signal.org/docs/specifications/doubleratchet/)
- [X3DH Key Agreement](https://signal.org/docs/specifications/x3dh/)
- RFC 7539: ChaCha20 and Poly1305 (AES-GCM alternative)
- RFC 5869: HMAC-based Extract-and-Expand Key Derivation Function (HKDF)
