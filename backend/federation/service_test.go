package federation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"plain domain", "example.com", "example.com", false},
		{"with https", "https://example.com", "example.com", false},
		{"with http", "http://example.com/", "example.com", false},
		{"with trailing slash", "example.com/", "example.com", false},
		{"uppercase", "EXAMPLE.COM", "example.com", false},
		{"empty", "", "", true},
		{"with path", "example.com/foo", "", true},
		{"with query", "example.com?foo=bar", "", true},
		{"with fragment", "example.com#frag", "", true},
		{"with spaces", "  example.com  ", "example.com", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeDomain(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestValidateMessageEnvelope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     MessageEnvelope
		wantErr bool
	}{
		{"valid envelope", MessageEnvelope{
			MessageID:    "msg-1",
			SourceDomain: "alice.example.com",
			TargetDomain: "bob.example.com",
			SenderUserID: "user-1",
			TargetUserID: "user-2",
			Payload:      map[string]any{"ciphertext": "base64data"},
		}, false},
		{"empty message_id", MessageEnvelope{
			SourceDomain: "alice.example.com",
			TargetDomain: "bob.example.com",
			SenderUserID: "user-1",
			TargetUserID: "user-2",
			Payload:      map[string]any{"ciphertext": "base64data"},
		}, true},
		{"invalid source domain", MessageEnvelope{
			MessageID:    "msg-1",
			SourceDomain: "",
			TargetDomain: "bob.example.com",
			SenderUserID: "user-1",
			TargetUserID: "user-2",
			Payload:      map[string]any{"ciphertext": "base64data"},
		}, true},
		{"missing ciphertext", MessageEnvelope{
			MessageID:    "msg-1",
			SourceDomain: "alice.example.com",
			TargetDomain: "bob.example.com",
			SenderUserID: "user-1",
			TargetUserID: "user-2",
			Payload:      map[string]any{"plaintext": "data"},
		}, true},
		{"empty sender", MessageEnvelope{
			MessageID:    "msg-1",
			SourceDomain: "alice.example.com",
			TargetDomain: "bob.example.com",
			SenderUserID: "",
			TargetUserID: "user-2",
			Payload:      map[string]any{"ciphertext": "base64data"},
		}, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateMessageEnvelope(tc.env)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServiceAnnounceValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)

	// Empty domain should fail validation before touching DB
	_, err := svc.Announce(context.Background(), "", "abc123")
	require.ErrorIs(t, err, ErrInvalidNode)

	// Empty public key should fail validation before touching DB
	_, err = svc.Announce(context.Background(), "node.example.com", "")
	require.ErrorIs(t, err, ErrInvalidNode)
}
