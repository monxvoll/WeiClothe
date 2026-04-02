package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"

	"weicloth/internal/core/ports"

	segmentio "github.com/segmentio/kafka-go"
)

// ── mock writer ─────────────────────────────────────────────────

type recorded struct {
	topic string
	msgs  []segmentio.Message
}

type mockWriter struct {
	mu       sync.Mutex
	records  []recorded
	topic    string
	writeErr error
	closeErr error
	closed   bool
}

func (m *mockWriter) WriteMessages(_ context.Context, msgs ...segmentio.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return m.writeErr
	}
	m.records = append(m.records, recorded{topic: m.topic, msgs: msgs})
	return nil
}

func (m *mockWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
}

func (m *mockWriter) messageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, r := range m.records {
		n += len(r.msgs)
	}
	return n
}

// mockFactory returns a writerFactory that stores every created writer
// in the provided map so tests can inspect them after the fact.
func mockFactory(writers map[string]*mockWriter, writeErr error) writerFactory {
	return func(topic string) messageWriter {
		w := &mockWriter{topic: topic, writeErr: writeErr}
		writers[topic] = w
		return w
	}
}

// ── tests ───────────────────────────────────────────────────────

func TestPublish_SingleMessage(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	err := p.Publish(context.Background(), "garment.ingest", "user-1", []byte(`{"img":"s3://a.jpg"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w, ok := writers["garment.ingest"]
	if !ok {
		t.Fatal("writer for topic garment.ingest was never created")
	}
	if w.messageCount() != 1 {
		t.Fatalf("expected 1 message, got %d", w.messageCount())
	}

	msg := w.records[0].msgs[0]
	if string(msg.Key) != "user-1" {
		t.Errorf("expected key 'user-1', got '%s'", string(msg.Key))
	}
	if string(msg.Value) != `{"img":"s3://a.jpg"}` {
		t.Errorf("unexpected payload: %s", string(msg.Value))
	}
}

func TestPublish_EmptyKey_OmitsKeyField(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	_ = p.Publish(context.Background(), "user.login", "", []byte(`{}`))

	msg := writers["user.login"].records[0].msgs[0]
	if msg.Key != nil {
		t.Errorf("expected nil key when empty string passed, got '%s'", string(msg.Key))
	}
}

func TestPublish_BrokerError_PropagatesWrapped(t *testing.T) {
	brokerErr := errors.New("connection refused")
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, brokerErr))

	err := p.Publish(context.Background(), "garment.ingest", "u1", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, brokerErr) {
		t.Errorf("expected wrapped broker error, got: %v", err)
	}
}

func TestPublishBatch_MultipleMessages(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	batch := []ports.Message{
		{Key: "u1", Payload: []byte(`{"a":1}`)},
		{Key: "u2", Payload: []byte(`{"a":2}`)},
		{Key: "", Payload: []byte(`{"a":3}`)},
	}

	err := p.PublishBatch(context.Background(), "garment.ingest", batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := writers["garment.ingest"]
	if w.messageCount() != 3 {
		t.Fatalf("expected 3 messages, got %d", w.messageCount())
	}

	msgs := w.records[0].msgs
	if string(msgs[0].Key) != "u1" {
		t.Errorf("msg[0] key: expected 'u1', got '%s'", string(msgs[0].Key))
	}
	if msgs[2].Key != nil {
		t.Errorf("msg[2] key: expected nil for empty key, got '%s'", string(msgs[2].Key))
	}
}

func TestPublishBatch_BrokerError(t *testing.T) {
	brokerErr := errors.New("timeout")
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, brokerErr))

	err := p.PublishBatch(context.Background(), "t", []ports.Message{{Payload: []byte(`x`)}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, brokerErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestWriterPool_ReusesSameTopicWriter(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	_ = p.Publish(context.Background(), "t1", "", []byte(`a`))
	_ = p.Publish(context.Background(), "t1", "", []byte(`b`))

	if len(writers) != 1 {
		t.Fatalf("expected 1 writer (reused), factory created %d", len(writers))
	}
	if writers["t1"].messageCount() != 2 {
		t.Fatalf("expected 2 messages on same writer, got %d", writers["t1"].messageCount())
	}
}

func TestWriterPool_SeparateWritersPerTopic(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	_ = p.Publish(context.Background(), "t1", "", []byte(`a`))
	_ = p.Publish(context.Background(), "t2", "", []byte(`b`))

	if len(writers) != 2 {
		t.Fatalf("expected 2 writers, got %d", len(writers))
	}
}

func TestClose_ClosesAllWriters(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	_ = p.Publish(context.Background(), "t1", "", []byte(`a`))
	_ = p.Publish(context.Background(), "t2", "", []byte(`b`))

	err := p.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for topic, w := range writers {
		if !w.closed {
			t.Errorf("writer for topic %q was not closed", topic)
		}
	}
}

func TestClose_ReturnsFirstError(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(mockFactory(writers, nil))

	_ = p.Publish(context.Background(), "t1", "", []byte(`a`))

	writers["t1"].closeErr = errors.New("flush failed")

	err := p.Close()
	if err == nil {
		t.Fatal("expected error from Close, got nil")
	}
}

func TestContextCancellation_ReturnsError(t *testing.T) {
	writers := make(map[string]*mockWriter)
	p := newProducer(func(topic string) messageWriter {
		w := &mockWriter{
			topic: topic,
			writeErr: context.Canceled,
		}
		writers[topic] = w
		return w
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Publish(ctx, "t1", "", []byte(`a`))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}
