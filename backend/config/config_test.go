package config

import (
	"testing"
)

func TestSplitCommaList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"a", []string{"a"}},
		{"a,b", []string{"a", "b"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{",,,", nil},
	}
	for _, tt := range tests {
		got := splitCommaList(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("splitCommaList(%q): len %d, want %d (%v vs %v)", tt.in, len(got), len(tt.want), got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("splitCommaList(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestLoad_RequiresDBAndJWT(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DB_URL and validation fails")
	}
}

func TestLoad_OK(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "x")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.com, https://b.com")
	t.Setenv("ADMIN_USER_IDS", " id-1 , id-2 ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://a.com" {
		t.Fatalf("CORSAllowedOrigins: %#v", cfg.CORSAllowedOrigins)
	}
	if len(cfg.AdminUserIDs) != 2 || cfg.AdminUserIDs[0] != "id-1" {
		t.Fatalf("AdminUserIDs: %#v", cfg.AdminUserIDs)
	}
}
