package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMFARequired        = errors.New("mfa_required")
	ErrInvalidTOTPSecret  = errors.New("invalid_totp_secret")
	ErrTOTPNotEnabled     = errors.New("totp_not_enabled")
	ErrTOTPAlreadyEnabled = errors.New("totp_already_enabled")
	ErrInvalidTOTPCode    = errors.New("invalid_totp_code")
)

const (
	totpTimeStep   = 30
	totpCodeDigits = 6
	totpSecretBits = 160
	totpLookAhead  = 1
	totpIssuer     = "Zenthril"
)

// generateTOTPSecret creates a random base32-encoded TOTP secret.
func generateTOTPSecret() (string, error) {
	buf := make([]byte, totpSecretBits/8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(buf), nil
}

// totpURI builds an otpauth:// URI for authenticator apps.
func totpURI(username, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		totpIssuer, username, secret, totpIssuer)
}

// computeTOTPCode computes the TOTP code for a given time step using HMAC-SHA1.
func computeTOTPCode(secret []byte, counter uint64) (string, error) {
	msg := make([]byte, 8)
	for i := uint64(0); i < 8; i++ {
		msg[7-i] = byte(counter >> (8 * i))
	}

	mac := hmac.New(sha1.New, secret)
	mac.Write(msg)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binary := uint32(sum[offset])&0x7f<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	code := int(binary % 1000000)
	return fmt.Sprintf("%06d", code), nil
}

// verifyTOTPCode checks a user-provided 6-digit code against current and
// adjacent time steps to allow for clock skew.
func verifyTOTPCode(secret []byte, code string) error {
	if len(code) != totpCodeDigits {
		return ErrInvalidTOTPCode
	}

	now := time.Now().Unix()
	for offset := -totpLookAhead; offset <= totpLookAhead; offset++ {
		counter := uint64(now+int64(offset)*totpTimeStep) / totpTimeStep
		expected, err := computeTOTPCode(secret, counter)
		if err != nil {
			continue
		}
		if hmac.Equal([]byte(expected), []byte(code)) {
			return nil
		}
	}
	return ErrInvalidTOTPCode
}

// base32Decode decodes a base32 secret, normalizing whitespace and case.
func base32Decode(secret string) ([]byte, error) {
	secret = strings.TrimSpace(secret)
	secret = strings.ToUpper(secret)
	secret = strings.ReplaceAll(secret, " ", "")
	decoded, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, ErrInvalidTOTPSecret
	}
	return decoded, nil
}
