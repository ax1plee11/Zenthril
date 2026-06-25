package user

import (
	"testing"
)

func TestValidateSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{"empty string", "", ErrQueryTooShort},
		{"one char", "a", ErrQueryTooShort},
		{"two chars ok", "ab", nil},
		{"normal query", "alice", nil},
		{"with underscore", "alice_bob", nil},
		{"with hyphen", "alice-bob", nil},
		{"too long", string(make([]rune, MaxSearchQueryLength+1)), ErrQueryTooLong},
		{"exactly max", string(make([]rune, MaxSearchQueryLength)), nil},
		{"sql injection attempt", "' OR 1=1--", ErrQueryInvalid},
		{"semicolon", "abc;", ErrQueryInvalid},
		{"percent sign", "abc%", ErrQueryInvalid},
		{"backslash", `abc\`, ErrQueryInvalid},
		{"unicode letters ok", "алиса", nil},
		{"unicode digits ok", "ab123", nil},
		{"space disallowed", "alice bob", ErrQueryInvalid},
	}

	// Fill the "exactly max" runes with valid chars.
	tests[7].query = ""
	for i := 0; i < MaxSearchQueryLength; i++ {
		tests[7].query += "a"
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSearchQuery(tt.query)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestIsAllowedSearchChar(t *testing.T) {
	t.Parallel()

	allowed := []rune{'a', 'Z', '0', '9', '_', '-', 'а', 'Я'}
	for _, r := range allowed {
		if !isAllowedSearchChar(r) {
			t.Errorf("expected rune %q to be allowed", r)
		}
	}

	disallowed := []rune{' ', '\t', '\'', '"', ';', '%', '\\', '/', '(', ')'}
	for _, r := range disallowed {
		if isAllowedSearchChar(r) {
			t.Errorf("expected rune %q to be disallowed", r)
		}
	}
}
