package auth

import "testing"

func TestTOTPStorageBindsCiphertextToAccountAndKey(t *testing.T) {
	svc := NewService(nil, nil, "test-signing-secret")
	stored, err := svc.encryptTOTPSecret("alice", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	value, err := svc.decryptTOTPSecret("alice", stored)
	if err != nil || value != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("round trip failed: %v", err)
	}
	if _, err := svc.decryptTOTPSecret("bob", stored); err == nil {
		t.Fatal("ciphertext copied between accounts")
	}
	svc.SetMFAEncryptionKey("different-encryption-key")
	if _, err := svc.decryptTOTPSecret("alice", stored); err == nil {
		t.Fatal("wrong key accepted")
	}
}
