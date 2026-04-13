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
