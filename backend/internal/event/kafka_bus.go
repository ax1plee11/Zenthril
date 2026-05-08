package event

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"zenthril-backend/internal/config"
)

type KafkaBus struct {
	brokers []string
	groupID string
	logger  *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	writers map[string]*kafka.Writer
	readers []*kafka.Reader
}

func NewKafkaBus(cfg config.EventBusConfig, logger *slog.Logger) (*KafkaBus, error) {
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &KafkaBus{
		brokers: cfg.KafkaBrokers,
		groupID: cfg.ConsumerGroup,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		writers: make(map[string]*kafka.Writer),
	}, nil
}

func (b *KafkaBus) Publish(ctx context.Context, topic string, evt Event, opts PublishOptions) error {
	if evt.ID == "" {
		evt.ID = newEventID(time.Now().UTC())
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	key := opts.Key
	if key == "" {
		key = evt.PartitionKey
	}

	return b.writer(topic).WriteMessages(ctx, kafka.Message{
		Key:     []byte(key),
		Value:   payload,
		Time:    evt.OccurredAt,
		Headers: kafkaHeaders(opts.Headers),
	})
}

func (b *KafkaBus) Subscribe(ctx context.Context, sub Subscription) error {
	if sub.Topic == "" {
		return errors.New("subscription topic is required")
	}
	if sub.Handler == nil {
		return errors.New("subscription handler is required")
	}
	groupID := sub.Group
	if groupID == "" {
		groupID = b.groupID
	}
	if groupID == "" {
		return errors.New("consumer group is required")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        b.brokers,
		Topic:          sub.Topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		CommitInterval: 0,
	})

	b.mu.Lock()
	b.readers = append(b.readers, reader)
	b.mu.Unlock()

	go b.consume(ctx, reader, sub)
	return nil
}

func (b *KafkaBus) Close(ctx context.Context) error {
	b.cancel()

	b.mu.Lock()
	writers := make([]*kafka.Writer, 0, len(b.writers))
	for _, writer := range b.writers {
		writers = append(writers, writer)
	}
	readers := append([]*kafka.Reader(nil), b.readers...)
	b.mu.Unlock()

	var err error
	for _, reader := range readers {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	for _, writer := range writers {
		done := make(chan error, 1)
		go func(writer *kafka.Writer) {
			done <- writer.Close()
		}(writer)
		select {
		case <-ctx.Done():
			if err == nil {
				err = ctx.Err()
			}
		case closeErr := <-done:
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}
	}
	return err
}

func (b *KafkaBus) consume(ctx context.Context, reader *kafka.Reader, sub Subscription) {
	for {
		msg, err := reader.FetchMessage(b.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				return
			}
			b.logger.Error("kafka fetch failed", "topic", sub.Topic, "error", err)
			continue
		}

		var evt Event
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			b.logger.Error("kafka event decode failed", "topic", sub.Topic, "error", err)
			continue
		}
		if err := sub.Handler(ctx, evt); err != nil {
			b.logger.Error("kafka event handler failed", "topic", sub.Topic, "event_id", evt.ID, "error", err)
			continue
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			b.logger.Error("kafka commit failed", "topic", sub.Topic, "event_id", evt.ID, "error", err)
		}
	}
}

func (b *KafkaBus) writer(topic string) *kafka.Writer {
	b.mu.Lock()
	defer b.mu.Unlock()

	if writer := b.writers[topic]; writer != nil {
		return writer
	}
	writer := &kafka.Writer{
		Addr:         kafka.TCP(b.brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchTimeout: 10 * time.Millisecond,
	}
	b.writers[topic] = writer
	return writer
}

func kafkaHeaders(headers map[string]string) []kafka.Header {
	if len(headers) == 0 {
		return nil
	}
	out := make([]kafka.Header, 0, len(headers))
	for key, value := range headers {
		out = append(out, kafka.Header{Key: key, Value: []byte(value)})
	}
	return out
}
