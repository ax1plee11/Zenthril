package gateway

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestRegistryDeliversToSubscribedChannel(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{NodeID: "node-a", MaxConnections: 10})
	conn := NewConnection("conn-1", "user-1", "device-1", "node-a", 1)
	if err := registry.Register(conn); err != nil {
		t.Fatalf("Register: %v", err)
	}
	defer registry.Unregister(conn.ID)

	if err := registry.Subscribe(conn.ID, "channel-1"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	delivered := registry.DeliverChannel(context.Background(), "channel-1", Envelope{
		Type:      "message.created",
		ChannelID: "channel-1",
		SentAt:    time.Now().UTC(),
	})
	if delivered != 1 {
		t.Fatalf("delivered = %d, want 1", delivered)
	}

	select {
	case payload := <-conn.Send():
		var env Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if env.ChannelID != "channel-1" {
			t.Fatalf("channel_id = %q, want channel-1", env.ChannelID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivery")
	}
}

func TestRegistryDrainingRejectsNewConnections(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{NodeID: "node-a"})
	registry.StartDraining()
	if err := registry.Register(NewConnection("conn-1", "user-1", "", "node-a", 1)); err != ErrDraining {
		t.Fatalf("Register error = %v, want ErrDraining", err)
	}
}
