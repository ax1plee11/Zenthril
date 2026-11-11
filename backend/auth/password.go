package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 4
	argon2KeyLen      = 32
	argon2SaltLen     = 16
)

var (
	ErrPasswordTooShort      = errors.New("password_too_short")
	ErrPasswordNoNumber      = errors.New("password_no_number")
	ErrPasswordNoUppercase   = errors.New("password_no_uppercase")
	ErrPasswordNoLowercase   = errors.New("password_no_lowercase")
	ErrPasswordNoSpecialChar = errors.New("password_no_special_char")
)

// ValidatePasswordComplexity checks if password meets security requirements
// Minimum 8 characters, at least one uppercase, one lowercase, one number, one special character
func ValidatePasswordComplexity(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ErrPasswordNoUppercase
	}
	if !hasLower {
		return ErrPasswordNoLowercase
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	if !hasSpecial {
		return ErrPasswordNoSpecialChar
	}

	return nil
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported hash algorithm")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("parse version: %w", err)
	}

	if version != argon2.Version {
		return false, errors.New("unsupported argon2 version")
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("parse params: %w", err)
	}

	if memory < 8*uint32(parallelism) || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 16 {
		return false, errors.New("unsafe argon2 parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	if len(salt) < 8 || len(salt) > 64 || len(expectedHash) < 16 || len(expectedHash) > 64 {
		return false, errors.New("invalid argon2 salt or key length")
	}
	actualHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
