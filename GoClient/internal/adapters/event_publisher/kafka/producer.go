package kafka

import (
	"context"
	"fmt"
	"time"

	"weicloth/internal/core/ports"

	"github.com/segmentio/kafka-go"
)

// messageWriter abstracts the write/close operations so the Producer
// can be unit-tested without a live broker.
type messageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// writerFactory builds a messageWriter for a given topic.
type writerFactory func(topic string) messageWriter

// ProducerConfig holds the tunables for a Kafka producer adapter.
type ProducerConfig struct {
	Brokers      []string
	BatchSize    int
	BatchTimeout time.Duration
	WriteTimeout time.Duration
	RequiredAcks kafka.RequiredAcks
	Async        bool
}

// DefaultProducerConfig returns production-safe defaults.
func DefaultProducerConfig(brokers []string) ProducerConfig {
	return ProducerConfig{
		Brokers:      brokers,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
}

// Producer implements ports.EventPublisher on top of segmentio/kafka-go.
//
// A single Producer can write to multiple topics; an internal writer
// pool is keyed by topic name so each topic gets its own batching and
// connection pipeline.
type Producer struct {
	factory writerFactory
	writers map[string]messageWriter
}

var _ ports.EventPublisher = (*Producer)(nil)

// NewProducer creates a ready-to-use Kafka producer adapter.
func NewProducer(cfg ProducerConfig) *Producer {
	return newProducer(defaultWriterFactory(cfg))
}

func newProducer(f writerFactory) *Producer {
	return &Producer{
		factory: f,
		writers: make(map[string]messageWriter),
	}
}

func defaultWriterFactory(cfg ProducerConfig) writerFactory {
	return func(topic string) messageWriter {
		return &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			BatchSize:              cfg.BatchSize,
			BatchTimeout:           cfg.BatchTimeout,
			WriteTimeout:           cfg.WriteTimeout,
			RequiredAcks:           cfg.RequiredAcks,
			Async:                  cfg.Async,
			AllowAutoTopicCreation: true,
		}
	}
}

// writer returns (or lazily creates) a messageWriter for the given topic.
func (p *Producer) writer(topic string) messageWriter {
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := p.factory(topic)
	p.writers[topic] = w
	return w
}

func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	msg := kafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}

	if err := p.writer(topic).WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish [topic=%s]: %w", topic, err)
	}
	return nil
}

func (p *Producer) PublishBatch(ctx context.Context, topic string, messages []ports.Message) error {
	msgs := make([]kafka.Message, len(messages))
	for i, m := range messages {
		msgs[i] = kafka.Message{Value: m.Payload}
		if m.Key != "" {
			msgs[i].Key = []byte(m.Key)
		}
	}

	if err := p.writer(topic).WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("kafka publish batch [topic=%s, count=%d]: %w", topic, len(messages), err)
	}
	return nil
}

func (p *Producer) Close() error {
	var firstErr error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("kafka close writer [topic=%s]: %w", topic, err)
		}
	}
	return firstErr
}
