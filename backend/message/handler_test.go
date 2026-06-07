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
		ChannelID:       "channel-1",
		SenderUserID:    "user-1",
		SenderDeviceID:  "device-1",
		SessionID:       "channel:channel-1",
		ClientMessageID: "client-message-1",
		CipherSuite:     models.CipherSuiteV2,
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
	payload.ProtocolVersion = 99
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("invalid protocol version was accepted")
	}
}

func TestValidateEncryptedPayloadAcceptsLegacyV1(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.ProtocolVersion = models.LegacyCryptoProtocolVersion
	payload.SenderDeviceID = ""
	payload.SessionID = ""
	payload.ClientMessageID = ""
	payload.CipherSuite = ""
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("legacy payload rejected: %v", err)
	}
}

func TestValidateEncryptedPayloadRejectsMissingAADV2Fields(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.SenderDeviceID = ""
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("missing v2 aad field was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsMissingChannelAADV2Field(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.ChannelID = ""
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("missing v2 channel aad field was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsMissingSenderAADV2Field(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.SenderUserID = ""
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("missing v2 sender aad field was accepted")
	}
}

func TestValidateEncryptedPayloadRejectsUnsupportedCipherSuite(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.CipherSuite = "unknown"
	if err := validateEncryptedPayload(payload); err == nil {
		t.Fatal("unsupported cipher suite was accepted")
	}
}

func TestValidateEnvelopeClaimsRejectsMismatchedChannel(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.ChannelID = "channel-2"
	if err := validateEnvelopeClaims(payload, "channel-1", "user-1"); err == nil {
		t.Fatal("mismatched channel claim was accepted")
	}
}

func TestValidateEnvelopeClaimsRejectsMismatchedSender(t *testing.T) {
	t.Parallel()

	payload := validPayload()
	payload.SenderUserID = "user-2"
	if err := validateEnvelopeClaims(payload, "channel-1", "user-1"); err == nil {
		t.Fatal("mismatched sender claim was accepted")
	}
}

func TestDecodeEncryptedPayloadRequestAcceptsWrappedPayload(t *testing.T) {
	t.Parallel()

	body := `{"payload":{"ciphertext":"aGVsbG8=","iv":"MTIzNDU2Nzg5MDEy","key_id":"key-1","tag":"MTIzNDU2Nzg5MDEyMzQ1Ng==","protocol_version":1}}`
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	payload, err := decodeEncryptedPayloadRequest(rec, req)
	if err != nil {
		t.Fatalf("wrapped payload rejected: %v", err)
	}
	if payload.Tag == "" || payload.ProtocolVersion != models.LegacyCryptoProtocolVersion {
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
	rec := httptest.NewRecorder()

	payload, err := decodeEncryptedPayloadRequest(rec, req)
	if err != nil {
		t.Fatalf("bare payload rejected: %v", err)
	}
	if payload.Tag == "" || payload.ProtocolVersion != models.LegacyCryptoProtocolVersion {
		t.Fatalf("decoded payload lost envelope fields: %#v", payload)
	}
	if err := validateEncryptedPayload(payload); err != nil {
		t.Fatalf("decoded bare payload did not validate: %v", err)
	}
}

func TestDecodeEncryptedPayloadRequestRejectsOversizedBodyBeforeValidation(t *testing.T) {
	t.Parallel()

	body := strings.Repeat(" ", maxEncryptedPayloadBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	if _, err := decodeEncryptedPayloadRequest(rec, req); err == nil {
		t.Fatal("oversized request body was accepted")
	}
}
