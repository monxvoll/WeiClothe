package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

const clotheAuditTopic = "audit.clothes.v1"

// ClotheService orchestrates garment persistence and operational audit events.
type ClotheService struct {
	clotheRepository ports.ClotheRepository
	eventPublisher   ports.EventPublisher
}

type clotheAuditEvent struct {
	EventType    string `json:"event_type"`
	Operation    string `json:"operation"`
	GarmentID    string `json:"garment_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Status       string `json:"status"`
	OccurredAt   string `json:"occurred_at"`
	Source       string `json:"source,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	ModelVersion string `json:"model_version,omitempty"`
}

func NewClotheService(repo ports.ClotheRepository, events ports.EventPublisher) *ClotheService {
	return &ClotheService{
		clotheRepository: repo,
		eventPublisher:   events,
	}
}

func (s *ClotheService) RegisterClothe(ctx context.Context, garment *domain.Garment) error {
	if err := s.clotheRepository.CreateClothe(ctx, garment); err != nil {
		return fmt.Errorf("failed to create clothe: %w", err)
	}

	s.publishAuditEvent(ctx, clotheAuditEvent{
		EventType:  "clothe.created",
		Operation:  "create_clothe",
		GarmentID:  garment.ID,
		UserID:     garment.UserID,
		Status:     garment.Status,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		Source:     garment.Source,
	})

	return nil
}

func (s *ClotheService) UpdateClotheStatus(ctx context.Context, garmentID string, status string, userID string) error {
	if err := s.clotheRepository.UpdateClotheStatus(ctx, garmentID, status); err != nil {
		return fmt.Errorf("failed to update clothe status: %w", err)
	}

	s.publishAuditEvent(ctx, clotheAuditEvent{
		EventType:  "clothe.status.updated",
		Operation:  "update_clothe_status",
		GarmentID:  garmentID,
		UserID:     userID,
		Status:     status,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	})

	return nil
}

func (s *ClotheService) SaveClassification(ctx context.Context, garment *domain.Garment) error {
	if err := s.clotheRepository.SaveClassification(ctx, garment); err != nil {
		return fmt.Errorf("failed to save classification: %w", err)
	}

	s.publishAuditEvent(ctx, clotheAuditEvent{
		EventType:    "clothe.classification.saved",
		Operation:    "save_classification",
		GarmentID:    garment.ID,
		UserID:       garment.UserID,
		Status:       garment.Status,
		OccurredAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Source:       garment.Source,
		ModelName:    garment.ModelName,
		ModelVersion: garment.ModelVersion,
	})

	return nil
}

func (s *ClotheService) GetClotheByID(ctx context.Context, garmentID string) (*domain.Garment, error) {
	garment, err := s.clotheRepository.GetClotheByID(ctx, garmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clothe by id: %w", err)
	}
	return garment, nil
}

func (s *ClotheService) ListClothesByUser(ctx context.Context, userID string) ([]domain.Garment, error) {
	garments, err := s.clotheRepository.ListClothesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clothes by user: %w", err)
	}
	return garments, nil
}

func (s *ClotheService) publishAuditEvent(ctx context.Context, event clotheAuditEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Warning: failed to marshal clothe audit event: %v\n", err)
		return
	}

	key := event.GarmentID
	if key == "" {
		key = event.UserID
	}

	publishCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if err := s.eventPublisher.Publish(publishCtx, clotheAuditTopic, key, payload); err != nil {
		// Best effort: business operation is already completed.
		fmt.Printf("Warning: clothe audit publication failed: %v\n", err)
	}
}
