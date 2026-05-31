# PoC 05: E2EE Envelope Tampering

**Risk:** Critical if accepted.

**Current expectation:** tampering with `tag`, `iv`, `keyId`, `protocolVersion`, ciphertext, or AAD context must fail decryption or backend validation.

## Safe PoC

Run the existing crypto tests:

```bash
cd client
npm test -- crypto
```

Manual tamper cases to keep covered:

```ts
const encrypted = await encrypt("hello", key);

await expect(decrypt({ ...encrypted, tag: "" }, key)).rejects.toThrow();
await expect(decrypt({ ...encrypted, iv: btoa("short") }, key)).rejects.toThrow();
await expect(decrypt({ ...encrypted, protocolVersion: 999 }, key)).rejects.toThrow();
await expect(decrypt({ ...encrypted, keyId: "tampered" }, key)).rejects.toThrow();
await expect(decrypt({ ...encrypted, ciphertext: encrypted.ciphertext.slice(0, -2) + "AA" }, key)).rejects.toThrow();
```

## Recommendation

- Keep HKDF domain separation.
- Keep protocol version in AAD.
- Implement full X3DH + Double Ratchet before making production E2EE claims.
- Add safety-number verification for MITM detection.
