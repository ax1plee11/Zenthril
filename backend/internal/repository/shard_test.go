package repository

import "testing"

func TestShardManagerStableSelection(t *testing.T) {
	t.Parallel()
	manager, err := NewShardManager([]Shard{
		{ID: "a", DSN: "postgres://a", Weight: 100},
		{ID: "b", DSN: "postgres://b", Weight: 100},
	}, 32)
	if err != nil {
		t.Fatalf("NewShardManager: %v", err)
	}

	first := manager.ShardForUserID("user-1")
	for i := 0; i < 10; i++ {
		if got := manager.ShardForUserID("user-1"); got.ID != first.ID {
			t.Fatalf("unstable shard: got %q, want %q", got.ID, first.ID)
		}
	}
}

func TestShardManagerRequiresShard(t *testing.T) {
	t.Parallel()
	if _, err := NewShardManager(nil, 0); err == nil {
		t.Fatal("expected error")
	}
}
