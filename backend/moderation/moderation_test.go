package moderation

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGuildStateManagerIsTrusted(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"admin-1", "admin-2"})

	if !m.IsTrusted("admin-1") {
		t.Fatal("expected admin-1 to be trusted")
	}
	if !m.IsTrusted("admin-2") {
		t.Fatal("expected admin-2 to be trusted")
	}
	if m.IsTrusted("user-3") {
		t.Fatal("expected user-3 not to be trusted")
	}
	if m.IsTrusted("") {
		t.Fatal("expected empty id not to be trusted")
	}
}

func TestGuildStateManagerLockUnlock(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"admin-1"})
	guildID := "guild-1"
	adminID := "admin-1"

	if err := m.LockGuild(guildID, adminID, "spam", true); err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}

	active, state := m.IsGuildLocked(guildID)
	if !active {
		t.Fatal("expected guild to be locked")
	}
	if state.ManagedBy != adminID {
		t.Fatalf("managed_by = %q, want %q", state.ManagedBy, adminID)
	}
	if state.Reason != "spam" {
		t.Fatalf("reason = %q, want %q", state.Reason, "spam")
	}
	if !state.AutoReset {
		t.Fatal("expected auto_reset to be true")
	}

	ok, msg := m.CanUserLeaveGuild(guildID, "user-1")
	if ok {
		t.Fatal("expected non-admin to be unable to leave locked guild")
	}
	if msg == "" {
		t.Fatal("expected a lock reason message")
	}

	ok, msg = m.CanUserLeaveGuild(guildID, adminID)
	if !ok {
		t.Fatalf("expected admin to be able to leave, got: %s", msg)
	}

	if err := m.UnlockGuild(guildID, adminID); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}

	active, _ = m.IsGuildLocked(guildID)
	if active {
		t.Fatal("expected guild to be unlocked")
	}

	if err := m.UnlockGuild(guildID, adminID); err == nil {
		t.Fatal("expected unlock of unlocked guild to fail")
	}
}

func TestGuildStateManagerLockRejectsNonTrusted(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"admin-1"})

	if err := m.LockGuild("guild-1", "user-2", "spam", true); err == nil {
		t.Fatal("expected lock by non-trusted user to fail")
	}
	if err := m.UnlockGuild("guild-1", "user-2"); err == nil {
		t.Fatal("expected unlock by non-trusted user to fail")
	}
}

func TestGuildStateManagerOwnerJoinLeave(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"owner-1"})
	guildID := "guild-1"
	ownerID := "owner-1"

	m.OnOwnerJoinGuild(guildID, ownerID)
	active, _ := m.IsGuildLocked(guildID)
	if !active {
		t.Fatal("expected guild to be locked on owner join")
	}

	m.OnOwnerLeaveGuild(guildID, ownerID)
	active, _ = m.IsGuildLocked(guildID)
	if active {
		t.Fatal("expected guild lock to auto-reset on owner leave")
	}

	m.OnOwnerJoinGuild(guildID, ownerID)
	m.OnOwnerLeaveGuild(guildID, "other-user")
	active, _ = m.IsGuildLocked(guildID)
	if !active {
		t.Fatal("expected lock to persist when non-owner leaves")
	}
}

func TestGuildStateManagerGetLockedGuilds(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"admin-1"})

	if got := len(m.GetLockedGuilds()); got != 0 {
		t.Fatalf("expected 0 locked guilds, got %d", got)
	}

	if err := m.LockGuild("guild-1", "admin-1", "reason", false); err != nil {
		t.Fatalf("lock error: %v", err)
	}
	if err := m.LockGuild("guild-2", "admin-1", "reason", false); err != nil {
		t.Fatalf("lock error: %v", err)
	}

	locked := m.GetLockedGuilds()
	if len(locked) != 2 {
		t.Fatalf("expected 2 locked guilds, got %d", len(locked))
	}
}

func TestGuildStateManagerGetGuildStatus(t *testing.T) {
	t.Parallel()

	m := NewGuildStateManager(nil, []string{"admin-1"})

	status := m.GetGuildStatus("guild-1")
	if status["locked"] != false {
		t.Fatal("expected unlocked guild to report locked=false")
	}

	if err := m.LockGuild("guild-1", "admin-1", "raid", true); err != nil {
		t.Fatalf("lock error: %v", err)
	}

	status = m.GetGuildStatus("guild-1")
	if status["locked"] != true {
		t.Fatal("expected locked guild to report locked=true")
	}
	if status["reason"] != "raid" {
		t.Fatalf("reason = %v, want raid", status["reason"])
	}
	if status["locked_by"] != "admin-1" {
		t.Fatalf("locked_by = %v, want admin-1", status["locked_by"])
	}
	if !status["auto_unlock"].(bool) {
		t.Fatal("expected auto_unlock=true")
	}
}

func TestExtractRealIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		want     []string
		wantLess bool
	}{
		{
			name:    "x_forwarded_for single",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.5"},
			remote:  "10.0.0.1",
			want:    []string{"203.0.113.5"},
		},
		{
			name:    "x_forwarded_for multiple picks first valid public",
			headers: map[string]string{"X-Forwarded-For": "10.0.0.1, 203.0.113.5, 198.51.100.2"},
			remote:  "10.0.0.1",
			want:    []string{"203.0.113.5", "198.51.100.2"},
		},
		{
			name:    "cf_connecting_ip",
			headers: map[string]string{"CF-Connecting-IP": "198.51.100.7"},
			remote:  "10.0.0.1",
			want:    []string{"198.51.100.7"},
		},
		{
			name:    "private ips are skipped",
			headers: map[string]string{"X-Real-IP": "10.0.0.5", "X-Forwarded-For": "172.16.0.1"},
			remote:  "10.0.0.1",
			want:    nil,
		},
		{
			name:    "invalid ips are skipped",
			headers: map[string]string{"X-Forwarded-For": "not-an-ip, 203.0.113.9"},
			remote:  "10.0.0.1",
			want:    []string{"203.0.113.9"},
		},
		{
			name:    "loopback remote skipped",
			headers: map[string]string{},
			remote:  "127.0.0.1",
			want:    nil,
		},
		{
			name:    "empty input",
			headers: nil,
			remote:  "",
			want:    nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractRealIP(tc.headers, tc.remote)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	t.Parallel()

	valid := []string{"203.0.113.5", "198.51.100.2", "8.8.8.8"}
	for _, ip := range valid {
		if !isValidIP(ip) {
			t.Fatalf("expected %s to be valid", ip)
		}
	}

	invalid := []string{"10.0.0.1", "172.16.0.1", "192.168.0.1", "127.0.0.1", "::1", "not-an-ip", "", "256.256.256.256"}
	for _, ip := range invalid {
		if isValidIP(ip) {
			t.Fatalf("expected %s to be invalid", ip)
		}
	}
}

func TestUUIDValidationHelpers(t *testing.T) {
	t.Parallel()

	valid := uuid.NewString()
	if _, err := uuid.Parse(valid); err != nil {
		t.Fatalf("valid UUID %q should parse: %v", valid, err)
	}

	invalid := []string{"not-a-uuid", "", "   ", "12345"}
	for _, s := range invalid {
		if _, err := uuid.Parse(s); err == nil {
			t.Fatalf("expected %q to fail parsing", s)
		}
	}
}

func TestTimeHelpers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	if now.IsZero() {
		t.Fatal("expected non-zero time")
	}

	future := now.Add(time.Hour)
	if !future.After(now) {
		t.Fatal("expected future to be after now")
	}

	past := now.Add(-time.Hour)
	if !past.Before(now) {
		t.Fatal("expected past to be before now")
	}
}
