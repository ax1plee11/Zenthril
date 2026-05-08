package event

import (
	"context"
	"errors"
	"sync"
)

type MemoryBus struct {
	mu       sync.RWMutex
	closed   bool
	handlers map[string][]Subscription
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers: make(map[string][]Subscription),
	}
}

func (b *MemoryBus) Publish(ctx context.Context, topic string, evt Event, _ PublishOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errors.New("event bus closed")
	}
	handlers := append([]Subscription(nil), b.handlers[topic]...)
	b.mu.RUnlock()

	for _, sub := range handlers {
		if sub.Handler == nil {
			continue
		}
		if err := sub.Handler(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

func (b *MemoryBus) Subscribe(ctx context.Context, sub Subscription) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if sub.Topic == "" {
		return errors.New("subscription topic is required")
	}
	if sub.Handler == nil {
		return errors.New("subscription handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("event bus closed")
	}
	b.handlers[sub.Topic] = append(b.handlers[sub.Topic], sub)
	return nil
}

func (b *MemoryBus) Close(context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.handlers = map[string][]Subscription{}
	return nil
}
