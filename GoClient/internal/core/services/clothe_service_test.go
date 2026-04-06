package services

import (
	"context"
	"errors"
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
	publishedTopic string
	publishedKey   string
	callCount      int
	publishErr     error
}

func (m *eventPublisherSpy) Publish(_ context.Context, topic string, key string, payload []byte) error {
	_ = payload
	m.publishedTopic = topic
	m.publishedKey = key
	m.callCount++
	return m.publishErr
}

func (m *eventPublisherSpy) PublishBatch(_ context.Context, _ string, _ []ports.Message) error {
	return nil
}

func (m *eventPublisherSpy) Close() error { return nil }

func TestClotheService_RegisterClothe_PublishesAuditEvent(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{}
	svc := NewClotheService(repo, pub)

	garment := &domain.Garment{
		UserID:      "77",
		ImageURL:    "s3://bucket/img.jpg",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.callCount != 1 {
		t.Fatalf("expected one kafka publish call, got %d", pub.callCount)
	}
	if pub.publishedTopic != clotheAuditTopic {
		t.Fatalf("expected topic %q, got %q", clotheAuditTopic, pub.publishedTopic)
	}
	if pub.publishedKey != "123" {
		t.Fatalf("expected key to be garment id, got %q", pub.publishedKey)
	}
}

func TestClotheService_RegisterClothe_DoesNotFailWhenKafkaFails(t *testing.T) {
	repo := &clotheRepoMock{}
	pub := &eventPublisherSpy{publishErr: errors.New("kafka down")}
	svc := NewClotheService(repo, pub)

	garment := &domain.Garment{
		UserID:      "77",
		ImageURL:    "s3://bucket/img.jpg",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment)
	if err != nil {
		t.Fatalf("kafka failure should not break business operation, got: %v", err)
	}
}

func TestClotheService_RegisterClothe_ReturnsRepoError(t *testing.T) {
	repo := &clotheRepoMock{createErr: errors.New("insert failed")}
	pub := &eventPublisherSpy{}
	svc := NewClotheService(repo, pub)

	garment := &domain.Garment{
		UserID:      "77",
		ImageURL:    "s3://bucket/img.jpg",
		GarmentType: "shirt",
	}

	err := svc.RegisterClothe(context.Background(), garment)
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
	if pub.callCount != 0 {
		t.Fatalf("did not expect kafka publish when repo fails, got %d calls", pub.callCount)
	}
}
