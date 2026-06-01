package cqrs

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrCommandTypeRequired = errors.New("command type is required")
	ErrCommandHandlerNil   = errors.New("command handler is nil")
	ErrCommandNotHandled   = errors.New("command handler not registered")
)

type Command struct {
	Type          string
	AggregateID   string
	ActorID       string
	CorrelationID string
	Payload       any
}

type CommandHandler func(context.Context, Command) error

type CommandBus struct {
	mu       sync.RWMutex
	handlers map[string]CommandHandler
}

func NewCommandBus() *CommandBus {
	return &CommandBus{handlers: make(map[string]CommandHandler)}
}

func (b *CommandBus) Register(commandType string, handler CommandHandler) error {
	if commandType == "" {
		return ErrCommandTypeRequired
	}
	if handler == nil {
		return ErrCommandHandlerNil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[commandType] = handler
	return nil
}

func (b *CommandBus) Dispatch(ctx context.Context, cmd Command) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd.Type == "" {
		return ErrCommandTypeRequired
	}

	b.mu.RLock()
	handler := b.handlers[cmd.Type]
	b.mu.RUnlock()
	if handler == nil {
		return fmt.Errorf("%w: %s", ErrCommandNotHandled, cmd.Type)
	}

	// CQRS: commands are the only write-side entry point for new domain modules.
	// SECURITY: command handlers must perform authorization before mutating state.
	return handler(ctx, cmd)
}

