package pubsub

import "context"

type Message struct {
	Channel string
	Payload []byte
}

type Subscription interface {
	Messages() <-chan Message
	Close() error
}

type PubSub interface {
	Publish(ctx context.Context, channel string, payload []byte) error
	Subscribe(ctx context.Context, channels ...string) (Subscription, error)
	Close() error
}
