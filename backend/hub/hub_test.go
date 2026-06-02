package hub

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewUpgraderRejectsCrossSiteOrigins(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	if upgrader.CheckOrigin(req) {
		t.Fatal("cross-site websocket origin was accepted")
	}
}

func TestNewUpgraderAcceptsExactAllowedOrigin(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if !upgrader.CheckOrigin(req) {
		t.Fatal("allowed websocket origin was rejected")
	}
}

func TestNewUpgraderFailsClosedWithoutAllowedOrigins(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader(nil, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if upgrader.CheckOrigin(req) {
		t.Fatal("websocket origin was accepted without WS_ALLOWED_ORIGINS")
	}
}

func TestNewUpgraderRejectsMissingOrigin(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)

	if upgrader.CheckOrigin(req) {
		t.Fatal("websocket request without Origin was accepted")
	}
}

func TestBroadcastToGuildOnlyTargetsMatchingGuild(t *testing.T) {
	t.Parallel()

	h := NewHub(nil)
	target := &Client{
		UserID:   "user-1",
		GuildIDs: []string{"guild-1"},
		Send:     make(chan []byte, 1),
	}
	other := &Client{
		UserID:   "user-2",
		GuildIDs: []string{"guild-2"},
		Send:     make(chan []byte, 1),
	}

	h.mu.Lock()
	h.users[target.UserID] = map[*Client]bool{target: true}
	h.users[other.UserID] = map[*Client]bool{other: true}
	h.mu.Unlock()

	h.BroadcastToGuild("guild-1", []byte(`{"type":"guild:update"}`))

	select {
	case msg := <-target.Send:
		if string(msg) != `{"type":"guild:update"}` {
			t.Fatalf("message = %s", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("matching guild client did not receive broadcast")
	}

	select {
	case msg := <-other.Send:
		t.Fatalf("non-matching guild client received broadcast: %s", msg)
	default:
	}
}

func TestBroadcastToGuildRejectsEmptyGuildID(t *testing.T) {
	t.Parallel()

	h := NewHub(nil)
	client := &Client{
		UserID:   "user-1",
		GuildIDs: []string{"guild-1"},
		Send:     make(chan []byte, 1),
	}
	h.mu.Lock()
	h.users[client.UserID] = map[*Client]bool{client: true}
	h.mu.Unlock()

	h.BroadcastToGuild("", []byte(`{"type":"guild:update"}`))

	select {
	case msg := <-client.Send:
		t.Fatalf("client received empty-guild broadcast: %s", msg)
	default:
	}
}

func TestUnregisterClientIsIdempotent(t *testing.T) {
	t.Parallel()

	h := NewHub(nil)
	client := &Client{
		UserID: "user-1",
		Send:   make(chan []byte, 1),
	}

	h.mu.Lock()
	h.users[client.UserID] = map[*Client]bool{client: true}
	h.channels["channel-1"] = map[*Client]bool{client: true}
	h.voiceChannels["channel-1"] = map[string]bool{client.UserID: true}
	h.mu.Unlock()

	if !h.unregisterClient(client) {
		t.Fatal("first unregister did not remove registered client")
	}
	if h.unregisterClient(client) {
		t.Fatal("second unregister reported registered client")
	}
	select {
	case _, ok := <-client.Send:
		if ok {
			t.Fatal("send channel remained open after unregister")
		}
	default:
		t.Fatal("send channel was not closed")
	}
}

func TestUserGuildAffinityUpdatesConnectedClients(t *testing.T) {
	t.Parallel()

	h := NewHub(nil)
	client := &Client{
		UserID: "user-1",
		Send:   make(chan []byte, 1),
	}
	h.mu.Lock()
	h.users[client.UserID] = map[*Client]bool{client: true}
	h.mu.Unlock()

	h.SetUserGuilds("user-1", []string{"guild-1", "guild-1", "", "guild-2"})
	if !clientInGuild(client, "guild-1") || !clientInGuild(client, "guild-2") {
		t.Fatalf("guild affinity was not set: %#v", client.GuildIDs)
	}
	if len(client.GuildIDs) != 2 {
		t.Fatalf("guild ids = %#v, want two deduped ids", client.GuildIDs)
	}

	h.AddUserToGuild("user-1", "guild-3")
	if !clientInGuild(client, "guild-3") {
		t.Fatalf("guild-3 was not added: %#v", client.GuildIDs)
	}

	h.RemoveUserFromGuild("user-1", "guild-1")
	if clientInGuild(client, "guild-1") {
		t.Fatalf("guild-1 was not removed: %#v", client.GuildIDs)
	}
}
