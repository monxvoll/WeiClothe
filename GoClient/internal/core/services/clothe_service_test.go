package services

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

type clotheRepoMock struct {
	createErr             error
	updateStatusErr       error
	saveClassificationErr error
	getByIDErr            error
	listByUserErr         error

	lastCreated *domain.Garment
}

func (m *clotheRepoMock) CreateClothe(_ context.Context, garment *domain.Garment) error {
	if m.createErr != nil {
		return m.createErr
	}
	garment.ID = "123"
	if garment.Status == "" {
		garment.Status = "queued"
	}
	m.lastCreated = garment
	return nil
}

func (m *clotheRepoMock) UpdateClotheStatus(_ context.Context, garmentID string, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	_ = garmentID
	_ = status
	return nil
}

func (m *clotheRepoMock) SaveClassification(_ context.Context, garment *domain.Garment) error {
	if m.saveClassificationErr != nil {
		return m.saveClassificationErr
	}
	if garment.Status == "" {
		garment.Status = "completed"
	}
	return nil
}

func (m *clotheRepoMock) GetClotheByID(_ context.Context, garmentID string) (*domain.Garment, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return &domain.Garment{ID: garmentID, UserID: "u1"}, nil
}

func (m *clotheRepoMock) ListClothesByUser(_ context.Context, userID string) ([]domain.Garment, error) {
	if m.listByUserErr != nil {
		return nil, m.listByUserErr
	}
	return []domain.Garment{
		{ID: "1", UserID: userID},
		{ID: "2", UserID: userID},
	}, nil
}

type eventPublisherSpy struct {
	publishedTopics   []string
	publishedKeys     []string
	publishedPayloads [][]byte
	callCount         int
	publishErr        error
}

func (m *eventPublisherSpy) Publish(_ context.Context, topic string, key string, payload []byte) error {
	m.publishedTopics = append(m.publishedTopics, topic)
	m.publishedKeys = append(m.publishedKeys, key)
	m.publishedPayloads = append(m.publishedPayloads, payload)
	m.callCount++
	return m.publishErr
}

func (m *eventPublisherSpy) PublishBatch(_ context.Context, _ string, _ []ports.Message) error {
	return nil
}

func (m *eventPublisherSpy) Close() error { return nil }

type storageSpy struct {
	lastKey   string
	lastBytes []byte
	lastCT    string
	stagedKeys []string
	err       error
}

func (s *storageSpy) StageRaw(_ context.Context, key string, data []byte, contentType string) error {
	if s.err != nil {
		return s.err
	}
	s.lastKey = key
	s.lastBytes = data
	s.lastCT = contentType
	s.stagedKeys = append(s.stagedKeys, key)
	return nil
}

func (s *storageSpy) Delete(_ context.Context, key string) error { return nil }

func TestClotheService_RegisterClothe_PublishesAuditEvent(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{}
	svc := NewClotheService(repo, pub, "", nil, slog.Default())

	garment := &domain.Garment{
		UserID:      "77",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment, nil, ".jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.callCount != 1 {
		t.Fatalf("expected one kafka publish call, got %d", pub.callCount)
	}
	if pub.publishedTopics[0] != clotheAuditTopic {
		t.Fatalf("expected topic %q, got %q", clotheAuditTopic, pub.publishedTopics[0])
	}
	if pub.publishedKeys[0] != "123" {
		t.Fatalf("expected key to be garment id, got %q", pub.publishedKeys[0])
	}
}

func TestClotheService_RegisterClothe_DoesNotFailWhenKafkaFails(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{publishErr: errors.New("kafka down")}
	svc := NewClotheService(repo, pub, "", nil, slog.Default())

	garment := &domain.Garment{
		UserID:      "77",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment, nil, ".jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("kafka failure should not break business operation, got: %v", err)
	}
}

func TestClotheService_RegisterClothe_ReturnsRepoError(t *testing.T) {
	repo := &clotheRepoMock{createErr: errors.New("insert failed")}
	pub := &eventPublisherSpy{}
	svc := NewClotheService(repo, pub, "", nil, slog.Default())

	garment := &domain.Garment{
		UserID:      "77",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment, nil, ".jpg", "image/jpeg")
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
	if pub.callCount != 0 {
		t.Fatalf("did not expect kafka publish when repo fails, got %d calls", pub.callCount)
	}
}

func TestClotheService_RegisterClothe_PublishesAnalysisRequest(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{}
	topic := "vusion.analysis.request"
	store := &storageSpy{}
	svc := NewClotheService(repo, pub, topic, store, slog.Default())

	garment := &domain.Garment{
		UserID:      "77",
		GarmentType: "shirt",
	}

	raw := []byte{0xff, 0xd8, 0xff, 0xe0}
	err := svc.RegisterClothe(context.Background(), garment, raw, ".jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.lastKey != "raw/77/123/original.jpg" {
		t.Fatalf("unexpected staging key %q", store.lastKey)
	}

	if pub.callCount != 2 {
		t.Fatalf("expected two kafka publish calls, got %d", pub.callCount)
	}
	if pub.publishedTopics[0] != clotheAuditTopic || pub.publishedTopics[1] != topic {
		t.Fatalf("topics: got %v", pub.publishedTopics)
	}
	var body clotheAnalysisRequestPayload
	if err := json.Unmarshal(pub.publishedPayloads[1], &body); err != nil {
		t.Fatalf("unmarshal analysis payload: %v", err)
	}
	if body.GarmentID != 123 || body.StagingKey != store.lastKey || body.UserID != "77" || body.Attempt != 0 {
		t.Fatalf("unexpected analysis payload: %+v", body)
	}
}

func TestClotheService_RegisterClothe_SkipsAnalysisWhenTopicEmpty(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{}
	store := &storageSpy{}
	svc := NewClotheService(repo, pub, "", store, slog.Default())

	garment := &domain.Garment{UserID: "77", GarmentType: "shirt"}
	raw := []byte{1, 2, 3}
	err := svc.RegisterClothe(context.Background(), garment, raw, ".jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastKey != "" {
		t.Fatal("expected no staging when analysis topic empty")
	}
	if pub.callCount != 1 {
		t.Fatalf("expected audit only, got %d publishes", pub.callCount)
	}
}
