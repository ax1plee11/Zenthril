package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

const totpStoragePrefix = "enc:v1:"

// SetMFAEncryptionKey allows deployments to rotate JWT signing keys independently.
// Configure this once at startup, before serving requests.
func (s *Service) SetMFAEncryptionKey(key string) {
	if key != "" {
		s.mfaEncryptionSecret = key
	}
}

func (s *Service) totpCipher() (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, []byte(s.mfaEncryptionSecret))
	mac.Write([]byte("zenthril/mfa-storage/v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Service) encryptTOTPSecret(userID, secret string) (string, error) {
	aead, err := s.totpCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nonce, nonce, []byte(secret), []byte(userID))
	return totpStoragePrefix + base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (s *Service) decryptTOTPSecret(userID, stored string) (string, error) {
	// Legacy rows are migrated on the next successful verification.
	if !strings.HasPrefix(stored, totpStoragePrefix) {
		return stored, nil
	}
	aead, err := s.totpCipher()
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, totpStoragePrefix))
	if err != nil || len(raw) < aead.NonceSize()+aead.Overhead() {
		return "", ErrInvalidTOTPSecret
	}
	secret, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(userID))
	if err != nil {
		return "", fmt.Errorf("decrypt totp secret: %w", ErrInvalidTOTPSecret)
	}
	return string(secret), nil
}
