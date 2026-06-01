package cqrs

import (
	"context"
	"errors"
	"testing"

	"zenthril-backend/internal/event"
)

func TestCommandBusDispatchesRegisteredHandler(t *testing.T) {
	bus := NewCommandBus()
	called := false

	if err := bus.Register("guild.create", func(ctx context.Context, cmd Command) error {
		called = true
		if cmd.AggregateID != "guild-1" {
			t.Fatalf("unexpected aggregate id: %s", cmd.AggregateID)
		}
		return nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	if err := bus.Dispatch(context.Background(), Command{Type: "guild.create", AggregateID: "guild-1"}); err != nil {
		t.Fatalf("dispatch command: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestCommandBusRejectsUnknownCommand(t *testing.T) {
	err := NewCommandBus().Dispatch(context.Background(), Command{Type: "missing"})
	if !errors.Is(err, ErrCommandNotHandled) {
		t.Fatalf("expected ErrCommandNotHandled, got %v", err)
	}
}

func TestQueryBusReturnsHandlerResult(t *testing.T) {
	bus := NewQueryBus()
	if err := bus.Register("guild.by_id", func(ctx context.Context, query Query) (any, error) {
		return "guild-result", nil
	}); err != nil {
		t.Fatalf("register query: %v", err)
	}

	got, err := bus.Ask(context.Background(), Query{Type: "guild.by_id"})
	if err != nil {
		t.Fatalf("ask query: %v", err)
	}
	if got != "guild-result" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestInMemoryEventStoreAppendLoadAndConcurrency(t *testing.T) {
	store := NewInMemoryEventStore()
	evt, err := event.New("message.created", "message-1", "channel-1", map[string]string{"id": "message-1"})
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	stored, err := store.Append(context.Background(), "message-1", 0, []event.Event{evt})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if stored[0].Version != 1 {
		t.Fatalf("unexpected version: %d", stored[0].Version)
	}

	loaded, err := store.Load(context.Background(), "message-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Event.ID != evt.ID {
		t.Fatalf("unexpected loaded events: %+v", loaded)
	}

	_, err = store.Append(context.Background(), "message-1", 0, []event.Event{evt})
	if !errors.Is(err, ErrConcurrencyViolation) {
		t.Fatalf("expected concurrency violation, got %v", err)
	}
}

