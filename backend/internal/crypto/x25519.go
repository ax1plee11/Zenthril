package crypto

import (
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

const x25519KeySize = 32

// generateEphemeralKeyPair creates a fresh X25519 key pair for X3DH initiation.
// SECURITY: ephemeral keys must be single-use per session bootstrap.
func generateEphemeralKeyPair() (privateKey, publicKey []byte, err error) {
	privateKey = make([]byte, x25519KeySize)
	if _, err := io.ReadFull(rand.Reader, privateKey); err != nil {
		return nil, nil, fmt.Errorf("generate ephemeral private key: %w", err)
	}
	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("derive ephemeral public key: %w", err)
	}
	return privateKey, publicKey, nil
}

// x25519SharedSecret performs a raw X25519 Diffie-Hellman exchange.
// WEAKNESS FIXED: session bootstrap now uses real ECDH instead of public-key hashing.
func x25519SharedSecret(privateKey, publicKey []byte) ([]byte, error) {
	if len(privateKey) != x25519KeySize || len(publicKey) != x25519KeySize {
		return nil, fmt.Errorf("X25519 keys must be %d bytes", x25519KeySize)
	}
	shared, err := curve25519.X25519(privateKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("X25519 DH: %w", err)
	}
	return shared, nil
}

// deriveX3DHSharedSecret computes the Signal-style X3DH input key material for
// an initiator with identity key IKa and ephemeral key EKa against a peer bundle.
//
// SECURITY: DH1=DH(IKa,SPKb), DH2=DH(EKa,IKb), DH3=DH(EKa,SPKb), DH4=DH(EKa,OPKb).
// Output is 96 bytes: rootKey[0:32], sendChainKey[32:64], recvChainKey[64:96].
// Matches client/src/features/e2ee/pairwiseSession.ts deriveSessionState.
func deriveX3DHSharedSecret(
	identityPrivateKey []byte,
	ephemeralPrivateKey []byte,
	peer DeviceKeyBundle,
	oneTimePreKeyPublic []byte,
	info []byte,
) ([]byte, error) {
	if len(identityPrivateKey) != x25519KeySize {
		return nil, fmt.Errorf("identity private key must be %d bytes", x25519KeySize)
	}
	if len(ephemeralPrivateKey) != x25519KeySize {
		return nil, fmt.Errorf("ephemeral private key must be %d bytes", x25519KeySize)
	}

	dh1, err := x25519SharedSecret(identityPrivateKey, peer.SignedPreKey)
	if err != nil {
		return nil, fmt.Errorf("X3DH DH1: %w", err)
	}
	dh2, err := x25519SharedSecret(ephemeralPrivateKey, peer.IdentityDHPublicKey)
	if err != nil {
		return nil, fmt.Errorf("X3DH DH2: %w", err)
	}
	dh3, err := x25519SharedSecret(ephemeralPrivateKey, peer.SignedPreKey)
	if err != nil {
		return nil, fmt.Errorf("X3DH DH3: %w", err)
	}

	parts := [][]byte{dh1, dh2, dh3}
	if len(oneTimePreKeyPublic) == x25519KeySize {
		dh4, err := x25519SharedSecret(ephemeralPrivateKey, oneTimePreKeyPublic)
		if err != nil {
			return nil, fmt.Errorf("X3DH DH4: %w", err)
		}
		parts = append(parts, dh4)
	}

	ikm := concatBytes(parts...)
	// SECURITY: salt matches client-side "zenthril.e2ee.x3dh.root.v1" for
	// cross-language X3DH interoperability.
	salt := []byte("zenthril.e2ee.x3dh.root.v1")
	return hkdfBytes(ikm, salt, info, ratchetKeySize*3)
}
