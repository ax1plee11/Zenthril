package gateway

import (
	"context"
	"errors"
	"testing"
)

type fakeChannelAccess struct {
	allowed map[string]bool
}

func (f *fakeChannelAccess) UserHasChannelAccess(_ context.Context, userID, channelID string) (bool, error) {
	if f.allowed == nil {
		return false, nil
	}
	return f.allowed[userID+":"+channelID], nil
}

func TestRegistrySubscribeRequiresChannelAccess(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{
		NodeID: "node-a",
		ChannelAccess: &fakeChannelAccess{
			allowed: map[string]bool{
				"user-1:allowed-channel": true,
			},
		},
	})
	conn := NewConnection("conn-1", "user-1", "device-1", "node-a", "", 1)
	if err := registry.Register(conn); err != nil {
		t.Fatalf("Register: %v", err)
	}
	defer registry.Unregister(conn.ID)

	if err := registry.Subscribe(context.Background(), conn.ID, "forbidden-channel", conn.UserID); !errors.Is(err, ErrChannelAccessDenied) {
		t.Fatalf("Subscribe forbidden error = %v, want ErrChannelAccessDenied", err)
	}

	if err := registry.Subscribe(context.Background(), conn.ID, "allowed-channel", conn.UserID); err != nil {
		t.Fatalf("Subscribe allowed channel: %v", err)
	}
}

func TestRegistryUserConnectionLimit(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{
		NodeID:                "node-a",
		MaxConnectionsPerUser: 2,
		ConnectionGuard:       NewConnectionGuard(100, 2),
	})

	for i := 0; i < 2; i++ {
		conn := NewConnection("conn-"+string(rune('a'+i)), "user-1", "", "node-a", "", 1)
		if err := registry.Register(conn); err != nil {
			t.Fatalf("Register %d: %v", i, err)
		}
	}

	third := NewConnection("conn-c", "user-1", "", "node-a", "", 1)
	if err := registry.Register(third); !errors.Is(err, ErrUserConnectionLimit) {
		t.Fatalf("Register third connection error = %v, want ErrUserConnectionLimit", err)
	}
}

func TestRegistryIPConnectionLimit(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{
		NodeID:          "node-a",
		ConnectionGuard: NewConnectionGuard(2, 10),
	})

	for i := 0; i < 2; i++ {
		conn := NewConnection("conn-"+string(rune('a'+i)), "user-"+string(rune('1'+i)), "", "node-a", "10.0.0.1", 1)
		if err := registry.Register(conn); err != nil {
			t.Fatalf("Register %d: %v", i, err)
		}
	}

	third := NewConnection("conn-c", "user-3", "", "node-a", "10.0.0.1", 1)
	if err := registry.Register(third); !errors.Is(err, ErrIPConnectionLimit) {
		t.Fatalf("Register third IP connection error = %v, want ErrIPConnectionLimit", err)
	}
}
