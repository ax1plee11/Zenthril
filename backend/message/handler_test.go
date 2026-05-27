package message

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zenthril-backend/models"
)

func validPayload() models.EncryptedPayload {
	return models.EncryptedPayload{
		Ciphertext:      base64.StdEncoding.EncodeToString([]byte("hello")),
		IV:              base64.StdEncoding.EncodeToString([]byte("123456789012")),
		KeyID:           "key-1",
		Tag:             base64.StdEncoding.EncodeToString([]byte("1234567890123456")),
		ProtocolVersion: models.CryptoProtocolVersion,
	}
}

func TestValidateEncryptedPayload(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
}

func TestValidateEncryptedPayloadRejectsOversizedCiphertext(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.Ciphertext = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", maxCiphertextBytes+1)))
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("oversized ciphertext was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsInvalidIV(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.IV = base64.StdEncoding.EncodeToString([]byte("short"))
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("invalid iv was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsMissingTag(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.Tag = ""
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("missing tag was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsInvalidTag(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.Tag = base64.StdEncoding.EncodeToString([]byte("short"))
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("invalid tag was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsInvalidProtocolVersion(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.ProtocolVersion = 2
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("invalid protocol version was accepted")
	}
}

func TestDecodeEncryptedPayloadRequestAcceptsWrappedPayload(t *testing.T) {
	t.Parallel()

	body := `{"payload":{"ciphertext":"aGVsbG8=","iv":"MTIzNDU2Nzg5MDEy","key_id":"key-1","tag":"MTIzNDU2Nzg5MDEyMzQ1Ng==","protocol_version":1}}`
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))

	payload, err := decodeEncryptedPayloadRequest(req)
	if err != nil {
		t.Fatalf("wrapped payload rejected: %v", err)
	}
	if payload.Tag == "" || payload.ProtocolVersion != models.CryptoProtocolVersion {
		t.Fatalf("decoded payload lost envelope fields: %#v", payload)
	}
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("decoded wrapped payload did not validate: %v", err)
	}
}

func TestDecodeEncryptedPayloadRequestAcceptsBarePayload(t *testing.T) {
	t.Parallel()

	body := `{"ciphertext":"aGVsbG8=","iv":"MTIzNDU2Nzg5MDEy","key_id":"key-1","tag":"MTIzNDU2Nzg5MDEyMzQ1Ng==","protocol_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))

	payload, err := decodeEncryptedPayloadRequest(req)
	if err != nil {
		t.Fatalf("bare payload rejected: %v", err)
	}
	if payload.Tag == "" || payload.ProtocolVersion != models.CryptoProtocolVersion {
		t.Fatalf("decoded payload lost envelope fields: %#v", payload)
	}
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("decoded bare payload did not validate: %v", err)
	}
}
