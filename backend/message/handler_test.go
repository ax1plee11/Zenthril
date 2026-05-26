package message

import (
	"encoding/base64"
	"strings"
	"testing"

	"zenthril-backend/models"
)

func TestValidateEncryptedPayload(t *testing.T) {
	t.Parallel()

	payload := models.EncryptedPayload{
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("hello")),
		IV:         base64.StdEncoding.EncodeToString([]byte("123456789012")),
		KeyID:      "key-1",
	}
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
}

func TestValidateEncryptedPayloadRejectsOversizedCiphertext(t *testing.T) {
	t.Parallel()

	payload := models.EncryptedPayload{
		Ciphertext: base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", maxCiphertextBytes+1))),
		IV:         base64.StdEncoding.EncodeToString([]byte("123456789012")),
		KeyID:      "key-1",
	}
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("oversized ciphertext was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsInvalidIV(t *testing.T) {
	t.Parallel()

	payload := models.EncryptedPayload{
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("hello")),
		IV:         base64.StdEncoding.EncodeToString([]byte("short")),
		KeyID:      "key-1",
	}
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("invalid iv was accepted")
	}
}
