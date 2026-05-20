package device

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestValidateRegisterDeviceRequest(t *testing.T) {
	t.Parallel()
	req := RegisterDeviceRequest{
		Name:                  "desktop",
		IdentityPublicKey:     testKey(32),
		SignedPreKeyID:        1,
		SignedPreKey:          testKey(32),
		SignedPreKeySignature: testKey(64),
		OneTimePreKeys: []OneTimePreKey{
			{KeyID: 1, PublicKey: testKey(32)},
			{KeyID: 2, PublicKey: testKey(32)},
		},
	}
	if err := validateRegisterDeviceRequest(req); err != nil {
		t.Fatalf("validateRegisterDeviceRequest: %v", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsBadKey(t *testing.T) {
	t.Parallel()
	req := RegisterDeviceRequest{
		IdentityPublicKey:     "not-base64",
		SignedPreKeyID:        1,
		SignedPreKey:          testKey(32),
		SignedPreKeySignature: testKey(64),
	}
	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestRejectsDuplicateOneTimePreKeys(t *testing.T) {
	t.Parallel()
	req := RegisterDeviceRequest{
		IdentityPublicKey:     testKey(32),
		SignedPreKeyID:        1,
		SignedPreKey:          testKey(32),
		SignedPreKeySignature: testKey(64),
		OneTimePreKeys: []OneTimePreKey{
			{KeyID: 1, PublicKey: testKey(32)},
			{KeyID: 1, PublicKey: testKey(32)},
		},
	}
	err := validateRegisterDeviceRequest(req)
	if !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("err = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestValidateRegisterDeviceRequestAcceptsBase64URLKeys(t *testing.T) {
	t.Parallel()
	req := RegisterDeviceRequest{
		IdentityPublicKey:     testURLKey(32),
		SignedPreKeyID:        1,
		SignedPreKey:          testURLKey(32),
		SignedPreKeySignature: testURLKey(64),
		OneTimePreKeys: []OneTimePreKey{
			{KeyID: 1, PublicKey: testURLKey(32)},
		},
	}
	if err := validateRegisterDeviceRequest(req); err != nil {
		t.Fatalf("validateRegisterDeviceRequest: %v", err)
	}
}

func TestDeviceFingerprintStable(t *testing.T) {
	t.Parallel()
	first := DeviceFingerprint("user-1", "device-1", testKey(32))
	second := DeviceFingerprint("user-1", "device-1", testKey(32))
	if first != second {
		t.Fatal("fingerprint is not stable")
	}
	if len(first) != 64 || strings.Contains(first, " ") {
		t.Fatalf("unexpected fingerprint shape: %q", first)
	}
}

func testKey(size int) string {
	return base64.StdEncoding.EncodeToString(make([]byte, size))
}

func testURLKey(size int) string {
	return base64.RawURLEncoding.EncodeToString(make([]byte, size))
}
