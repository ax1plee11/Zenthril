package cqrs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"zenthril-backend/internal/event"
)

var (
	ErrStreamIDRequired     = errors.New("stream id is required")
	ErrEventRequired        = errors.New("event is required")
	ErrConcurrencyViolation = errors.New("event stream version mismatch")
)

type StoredEvent struct {
	StreamID string
	Version  int64
	Event    event.Event
}

type EventStore interface {
	Append(ctx context.Context, streamID string, expectedVersion int64, events []event.Event) ([]StoredEvent, error)
	Load(ctx context.Context, streamID string) ([]StoredEvent, error)
}

type InMemoryEventStore struct {
	mu      sync.RWMutex
	streams map[string][]StoredEvent
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{streams: make(map[string][]StoredEvent)}
}

func (s *InMemoryEventStore) Append(
	ctx context.Context,
	streamID string,
	expectedVersion int64,
	events []event.Event,
) ([]StoredEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if streamID == "" {
		return nil, ErrStreamIDRequired
	}
	if len(events) == 0 {
		return nil, ErrEventRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.streams[streamID]
	if int64(len(current)) != expectedVersion {
		return nil, fmt.Errorf("%w: stream=%s expected=%d actual=%d", ErrConcurrencyViolation, streamID, expectedVersion, len(current))
	}

	stored := make([]StoredEvent, 0, len(events))
	for _, evt := range events {
		if evt.ID == "" || evt.Type == "" {
			return nil, ErrEventRequired
		}
		next := StoredEvent{
			StreamID: streamID,
			Version:  int64(len(current) + len(stored) + 1),
			Event:    evt,
		}
		stored = append(stored, next)
	}
	s.streams[streamID] = append(current, stored...)

	// EVENT-SOURCING: append-only stream with optimistic concurrency. Production
	// adapters should persist this to Postgres/Scylla with immutable writes.
	return append([]StoredEvent(nil), stored...), nil
}

func (s *InMemoryEventStore) Load(ctx context.Context, streamID string) ([]StoredEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if streamID == "" {
		return nil, ErrStreamIDRequired
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]StoredEvent(nil), s.streams[streamID]...), nil
}

