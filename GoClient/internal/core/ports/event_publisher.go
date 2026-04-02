package ports

import "context"

// EventPublisher is a driven port that abstracts asynchronous event
// production toward an external message broker (Kafka).
//
// The Go core uses this port to decouple domain-level intents
// (garment ingestion, classification requests, metadata propagation)
// from the underlying transport and serialisation details.
//
// Implementations are expected to live in internal/adapters (e.g. a
// Kafka producer adapter) and be injected into domain services at
// bootstrap time.
type EventPublisher interface {
	// Publish sends a single domain event to the specified topic.
	// The key is used for partition routing; pass an empty string
	// to let the broker decide.
	Publish(ctx context.Context, topic string, key string, payload []byte) error

	// PublishBatch sends multiple payloads to the same topic in a
	// single broker round-trip. Implementations should guarantee
	// atomicity or document partial-failure semantics.
	PublishBatch(ctx context.Context, topic string, messages []Message) error

	// Close performs a graceful shutdown: flushes pending writes and
	// releases broker connections.
	Close() error
}

// Message represents a single unit of work inside a batch publish.
type Message struct {
	Key     string
	Payload []byte
}