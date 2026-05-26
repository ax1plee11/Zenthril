package auth

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
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

func TestValidateRefreshTokenWithID(t *testing.T) {
	t.Parallel()
	const secret = "test-secret-at-least-32-bytes-long!!"
	token, err := generateTypedToken("user-uuid-123", "refresh", DefaultRefreshTokenTTL, secret)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	userID, tokenID, err := ValidateRefreshTokenWithID(token, secret)
	if err != nil {
		t.Fatalf("ValidateRefreshTokenWithID: %v", err)
	}
	if userID != "user-uuid-123" {
		t.Fatalf("user id: got %q want %q", userID, "user-uuid-123")
	}
	if tokenID == "" {
		t.Fatal("expected token id")
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

func TestValidateTokenRejectsNonHS256Algorithms(t *testing.T) {
	t.Parallel()
	claims := claims{
		UserID:    "user-uuid-123",
		TokenType: "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte("test-secret-at-least-32-bytes-long!!"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(signed, "test-secret-at-least-32-bytes-long!!"); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}
	noneToken := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) +
		"." + base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"user-uuid-123","token_type":"access"}`)) + "."
	if _, err := ValidateToken(noneToken, "test-secret-at-least-32-bytes-long!!"); err == nil {
		t.Fatal("expected alg none token to be rejected")
	}
}

func TestValidateTokenRequiresExpiration(t *testing.T) {
	t.Parallel()
	const secret = "test-secret-at-least-32-bytes-long!!"
	c := claims{
		UserID:    "user-uuid-123",
		TokenType: "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(signed, secret); err == nil {
		t.Fatal("expected token without exp to be rejected")
	}
}
