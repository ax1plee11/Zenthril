package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TopicGatewayDeliver  = "gateway.deliver.v1"
	TopicMessageCreated  = "message.created.v1"
	TopicPresenceChanged = "presence.changed.v1"
	TopicAuditLog        = "audit.log.v1"
)

type Event struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	AggregateID  string            `json:"aggregate_id,omitempty"`
	PartitionKey string            `json:"partition_key"`
	Data         json.RawMessage   `json:"data"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	OccurredAt   time.Time         `json:"occurred_at"`
}

type PublishOptions struct {
	Key     string
	Headers map[string]string
}

type Subscription struct {
	Topic   string
	Group   string
	Handler Handler
}

type Handler func(context.Context, Event) error

type Bus interface {
	Publish(ctx context.Context, topic string, evt Event, opts PublishOptions) error
	Subscribe(ctx context.Context, sub Subscription) error
	Close(ctx context.Context) error
}

func New(typeName, aggregateID, partitionKey string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	now := time.Now().UTC()
	return Event{
		ID:           newEventID(now),
		Type:         typeName,
		AggregateID:  aggregateID,
		PartitionKey: partitionKey,
		Data:         data,
		OccurredAt:   now,
	}, nil
}

func newEventID(t time.Time) string {
	return t.Format("20060102150405.000000000") + "-" + uuid.NewString()
}
