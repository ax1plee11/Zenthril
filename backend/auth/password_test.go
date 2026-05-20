package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordVerifyPassword_RoundTrip(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		n := 20
		if len(hash) < n {
			n = len(hash)
		}
		t.Fatalf("unexpected hash prefix: %s", hash[:n])
	}
	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}
	ok, err = VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("VerifyPassword wrong pass: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	t.Parallel()
	_, err := VerifyPassword("x", "not-a-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash format")
	}
}

func TestValidatePasswordComplexity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{name: "strong", pass: "StrongPass1!", wantErr: false},
		{name: "short", pass: "Aa1!", wantErr: true},
		{name: "no uppercase", pass: "strongpass1!", wantErr: true},
		{name: "no lowercase", pass: "STRONGPASS1!", wantErr: true},
		{name: "no number", pass: "StrongPass!", wantErr: true},
		{name: "no special", pass: "StrongPass1", wantErr: true},
	}
	for _, tt := range tests {
		err := ValidatePasswordComplexity(tt.pass)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%s: err = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
