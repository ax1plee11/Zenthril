package device

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateRegisterDeviceRequestVerifiesSignedPreKeySignature(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	if err := validateRegisterDeviceRequest(req); err != nil {
		t.Fatalf("valid signed prekey rejected: %v", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsInvalidSignedPreKeySignature(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	req.SignedPreKeySignature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))

	if err := validateRegisterDeviceRequest(req); err == nil {
		t.Fatal("invalid signed prekey signature was accepted")
	}
}

func TestValidateRegisterDeviceRequestRejectsBadKey(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	req.IdentityPublicKey = "not-base64"

	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsMissingIdentityDHKey(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	req.IdentityDHPublicKey = ""

	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsLowOrderX25519Key(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	req.IdentityDHPublicKey = base64.StdEncoding.EncodeToString(make([]byte, 32))

	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsDuplicateOneTimePreKeys(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequest(t)
	req.OneTimePreKeys = append(req.OneTimePreKeys, OneTimePreKey{
		KeyID:     req.OneTimePreKeys[0].KeyID,
		PublicKey: base64.StdEncoding.EncodeToString(randomBytes(t, 32)),
	})

	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestAcceptsBase64URLKeys(t *testing.T) {
	t.Parallel()

	req := validRegisterDeviceRequestWithEncoder(t, base64.RawURLEncoding)
	if err := validateRegisterDeviceRequest(req); err != nil {
		t.Fatalf("validateRegisterDeviceRequest: %v", err)
	}
}

func TestDeviceFingerprintStable(t *testing.T) {
	t.Parallel()

	key := base64.StdEncoding.EncodeToString(randomBytes(t, 32))
	first := DeviceFingerprint("user-1", "device-1", key, key)
	second := DeviceFingerprint("user-1", "device-1", key, key)
	if first != second {
		t.Fatal("fingerprint is not stable")
	}
	if len(first) != 64 || strings.Contains(first, " ") {
		t.Fatalf("unexpected fingerprint shape: %q", first)
	}
}

func TestDeviceFingerprintBindsBothIdentityKeys(t *testing.T) {
	t.Parallel()

	signingKey := base64.StdEncoding.EncodeToString(randomBytes(t, 32))
	firstDHKey := base64.StdEncoding.EncodeToString(randomBytes(t, 32))
	secondDHKey := base64.StdEncoding.EncodeToString(randomBytes(t, 32))
	first := DeviceFingerprint("user-1", "device-1", signingKey, firstDHKey)
	second := DeviceFingerprint("user-1", "device-1", signingKey, secondDHKey)
	if first == second {
		t.Fatal("fingerprint must change when the X3DH identity changes")
	}
}

func TestShouldConsumeOneTimePreKeyOnlyForCrossUserClaims(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	otherID := uuid.New()

	if shouldConsumeOneTimePreKey(ownerID, ownerID) {
		t.Fatal("same-user key bundle claim should not consume one-time prekeys")
	}
	if !shouldConsumeOneTimePreKey(otherID, ownerID) {
		t.Fatal("cross-user key bundle claim should consume one-time prekeys")
	}
}

func validRegisterDeviceRequest(t *testing.T) RegisterDeviceRequest {
	t.Helper()

	return validRegisterDeviceRequestWithEncoder(t, base64.StdEncoding)
}

func validRegisterDeviceRequestWithEncoder(t *testing.T, encoding *base64.Encoding) RegisterDeviceRequest {
	t.Helper()

	identityPublic, identityPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate identity key: %v", err)
	}
	signedPreKey := randomBytes(t, 32)
	signature := ed25519.Sign(identityPrivate, signedPreKeyMessage(signedPreKey))

	return RegisterDeviceRequest{
		DeviceID:              "11111111-1111-4111-8111-111111111111",
		Name:                  "test device",
		IdentityPublicKey:     encoding.EncodeToString(identityPublic),
		SignedPreKeyID:        1,
		SignedPreKey:          encoding.EncodeToString(signedPreKey),
		SignedPreKeySignature: encoding.EncodeToString(signature),
		IdentityDHPublicKey:   encoding.EncodeToString(randomBytes(t, 32)),
		OneTimePreKeys: []OneTimePreKey{
			{
				KeyID:     1,
				PublicKey: encoding.EncodeToString(randomBytes(t, 32)),
			},
		},
	}
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	out := make([]byte, n)
	if _, err := rand.Read(out); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return out
}
