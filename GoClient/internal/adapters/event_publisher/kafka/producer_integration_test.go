//go:build integration

package kafka

import (
	"context"
	"os"
	"testing"
	"time"

	"weicloth/internal/core/ports"

	segmentio "github.com/segmentio/kafka-go"
)

func brokerAddr() string {
	if addr := os.Getenv("KAFKA_BROKERS"); addr != "" {
		return addr
	}
	return "localhost:9093"
}

func readMessages(t *testing.T, topic string, expected int) []segmentio.Message {
	t.Helper()
	reader := segmentio.NewReader(segmentio.ReaderConfig{
		Brokers:   []string{brokerAddr()},
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  1e6,
		MaxWait:   3 * time.Second,
	})
	defer reader.Close()

	var msgs []segmentio.Message
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for range expected {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("failed to read message %d/%d: %v", len(msgs)+1, expected, err)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func TestIntegration_Publish_and_Consume(t *testing.T) {
	broker := brokerAddr()
	cfg := DefaultProducerConfig([]string{broker})
	cfg.BatchSize = 1
	cfg.BatchTimeout = time.Millisecond

	producer := NewProducer(cfg)
	defer producer.Close()

	topic := "integration.test.single"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload := []byte(`{"garment":"shirt","color":"blue"}`)
	err := producer.Publish(ctx, topic, "user-42", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	t.Logf("Published 1 message to topic %q", topic)

	msgs := readMessages(t, topic, 1)

	if string(msgs[0].Key) != "user-42" {
		t.Errorf("key: expected 'user-42', got '%s'", string(msgs[0].Key))
	}
	if string(msgs[0].Value) != string(payload) {
		t.Errorf("payload mismatch:\n  want: %s\n  got:  %s", payload, msgs[0].Value)
	}
	t.Logf("Consumed message: key=%s value=%s", msgs[0].Key, msgs[0].Value)
}

func TestIntegration_PublishBatch_and_Consume(t *testing.T) {
	broker := brokerAddr()
	cfg := DefaultProducerConfig([]string{broker})
	cfg.BatchSize = 10
	cfg.BatchTimeout = time.Millisecond

	producer := NewProducer(cfg)
	defer producer.Close()

	topic := "integration.test.batch"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch := []ports.Message{
		{Key: "u1", Payload: []byte(`{"item":"pants"}`)},
		{Key: "u2", Payload: []byte(`{"item":"jacket"}`)},
		{Key: "u3", Payload: []byte(`{"item":"shoes"}`)},
	}

	err := producer.PublishBatch(ctx, topic, batch)
	if err != nil {
		t.Fatalf("PublishBatch failed: %v", err)
	}
	t.Logf("Published %d messages to topic %q", len(batch), topic)

	msgs := readMessages(t, topic, 3)

	for i, m := range msgs {
		t.Logf("  [%d] key=%s value=%s", i, m.Key, m.Value)
		if string(m.Key) != batch[i].Key {
			t.Errorf("msg[%d] key: expected '%s', got '%s'", i, batch[i].Key, string(m.Key))
		}
		if string(m.Value) != string(batch[i].Payload) {
			t.Errorf("msg[%d] payload mismatch", i)
		}
	}
}

func TestIntegration_Close_FlushesAndDisconnects(t *testing.T) {
	broker := brokerAddr()
	cfg := DefaultProducerConfig([]string{broker})
	cfg.BatchSize = 1
	cfg.BatchTimeout = time.Millisecond

	producer := NewProducer(cfg)

	ctx := context.Background()
	_ = producer.Publish(ctx, "integration.test.close", "", []byte(`flush-me`))

	err := producer.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	msgs := readMessages(t, "integration.test.close", 1)
	if string(msgs[0].Value) != "flush-me" {
		t.Errorf("expected flushed message, got: %s", msgs[0].Value)
	}
	t.Log("Close flushed pending messages before disconnecting")
}
