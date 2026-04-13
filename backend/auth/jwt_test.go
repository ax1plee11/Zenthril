package auth

import (
	"strings"
	"testing"
)

func TestGenerateTokenValidateToken_RoundTrip(t *testing.T) {
	t.Parallel()
	const secret = "test-secret-at-least-32-bytes-long!!"
	token, err := GenerateToken("user-uuid-123", secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	got, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if got != "user-uuid-123" {
		t.Fatalf("user id: got %q want %q", got, "user-uuid-123")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	t.Parallel()
	token, err := GenerateToken("u1", "secret-one")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateToken(token, "secret-two")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidateToken_Garbage(t *testing.T) {
	t.Parallel()
	_, err := ValidateToken("not.a.jwt", "secret")
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestValidateToken_Tampered(t *testing.T) {
	t.Parallel()
	token, err := GenerateToken("u1", "somesecretsomesecretsomesecret")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected jwt shape: %d parts", len(parts))
	}
	parts[1] = "e30" // tampered payload ({} base64)
	tampered := strings.Join(parts, ".")
	_, err = ValidateToken(tampered, "somesecretsomesecretsomesecret")
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}
