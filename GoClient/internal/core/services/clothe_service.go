package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

const clotheAuditTopic = "audit.clothes.v1"

// ClotheService orchestrates garment persistence and operational audit events.
type ClotheService struct {
	clotheRepository ports.ClotheRepository
	eventPublisher   ports.EventPublisher
	analysisTopic    string // e.g. vusion.analysis.request; empty disables ML queue publish
	storage          ports.StorageUploader
	log              *slog.Logger
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

// clotheAnalysisRequestPayload is consumed by vusion-ml (Kafka topic vusion.analysis.request).
type clotheAnalysisRequestPayload struct {
	GarmentID  int    `json:"garment_id"`
	StagingKey string `json:"staging_key"`
	UserID     string `json:"user_id"`
	Attempt    int    `json:"attempt,omitempty"`
}

func NewClotheService(
	repo ports.ClotheRepository,
	events ports.EventPublisher,
	analysisTopic string,
	storage ports.StorageUploader,
	logger *slog.Logger,
) *ClotheService {
	return &ClotheService{
		clotheRepository: repo,
		eventPublisher:   events,
		analysisTopic:    analysisTopic,
		storage:          storage,
		log:              logger.With("service", "clothe"),
	}
}

// RegisterClothe persists the garment, stages raw bytes to object storage when analysis is enabled, then publishes Kafka jobs.
func (s *ClotheService) RegisterClothe(
	ctx context.Context,
	garment *domain.Garment,
	rawImage []byte,
	rawExt string,
	imageContentType string,
) error {
	garment.ImageURL = ""

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

	if s.analysisTopic != "" && s.storage != nil && len(rawImage) > 0 {
		key := fmt.Sprintf("raw/%s/%s/original%s", garment.UserID, garment.ID, rawExt)
		if err := s.storage.StageRaw(ctx, key, rawImage, imageContentType); err != nil {
			if uerr := s.clotheRepository.UpdateClotheStatus(ctx, garment.ID, "failed"); uerr != nil {
				s.log.Error("mark garment failed after staging error", "garment_id", garment.ID, "err", uerr)
			}
			return fmt.Errorf("stage raw image: %w", err)
		}
		s.publishAnalysisRequest(ctx, garment, key)
	}

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
		s.log.Error("marshal audit event", "event_type", event.EventType, "err", err)
		return
	}

	key := event.GarmentID
	if key == "" {
		key = event.UserID
	}

	publishCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if err := s.eventPublisher.Publish(publishCtx, clotheAuditTopic, key, payload); err != nil {
		s.log.Warn("audit publish failed", "topic", clotheAuditTopic, "garment_id", event.GarmentID, "err", err)
	}
}

func (s *ClotheService) publishAnalysisRequest(ctx context.Context, garment *domain.Garment, stagingKey string) {
	if s.analysisTopic == "" {
		return
	}
	id, err := strconv.Atoi(garment.ID)
	if err != nil {
		s.log.Warn("garment id not numeric, skipping analysis publish", "garment_id", garment.ID, "err", err)
		return
	}
	body := clotheAnalysisRequestPayload{
		GarmentID:  id,
		StagingKey: stagingKey,
		UserID:     garment.UserID,
		Attempt:    0,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		s.log.Error("marshal analysis request", "garment_id", garment.ID, "err", err)
		return
	}
	publishCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	key := garment.ID
	if err := s.eventPublisher.Publish(publishCtx, s.analysisTopic, key, payload); err != nil {
		s.log.Warn("analysis publish failed", "topic", s.analysisTopic, "garment_id", garment.ID, "err", err)
	}
}
