package cqrs

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrQueryTypeRequired = errors.New("query type is required")
	ErrQueryHandlerNil   = errors.New("query handler is nil")
	ErrQueryNotHandled   = errors.New("query handler not registered")
)

type Query struct {
	Type          string
	ActorID       string
	CorrelationID string
	Payload       any
}

type QueryHandler func(context.Context, Query) (any, error)

type QueryBus struct {
	mu       sync.RWMutex
	handlers map[string]QueryHandler
}

func NewQueryBus() *QueryBus {
	return &QueryBus{handlers: make(map[string]QueryHandler)}
}

func (b *QueryBus) Register(queryType string, handler QueryHandler) error {
	if queryType == "" {
		return ErrQueryTypeRequired
	}
	if handler == nil {
		return ErrQueryHandlerNil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[queryType] = handler
	return nil
}

func (b *QueryBus) Ask(ctx context.Context, query Query) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query.Type == "" {
		return nil, ErrQueryTypeRequired
	}

	b.mu.RLock()
	handler := b.handlers[query.Type]
	b.mu.RUnlock()
	if handler == nil {
		return nil, fmt.Errorf("%w: %s", ErrQueryNotHandled, query.Type)
	}

	// CQRS: queries are read-only and must not publish domain events.
	return handler(ctx, query)
}

