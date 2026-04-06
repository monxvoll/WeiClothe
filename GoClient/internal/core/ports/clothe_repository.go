package ports

import (
	"context"
	"weicloth/internal/core/domain"
)

type ClotheRepository interface {
	// CreateClothe stores the initial garment metadata when upload is accepted.
	// Expected initial status is "queued" or "processing".
	CreateClothe(ctx context.Context, garment *domain.Garment) error

	// UpdateClotheStatus updates processing state transitions
	// (queued -> processing -> completed/failed).
	UpdateClotheStatus(ctx context.Context, garmentID string, status string) error

	// SaveClassification updates garment fields once worker/ML returns results.
	// Typical fields: classification id, category, color, season, and name.
	SaveClassification(ctx context.Context, garment *domain.Garment) error

	// GetClotheByID fetches one garment for status/result queries.
	GetClotheByID(ctx context.Context, garmentID string) (*domain.Garment, error)

	// ListClothesByUser fetches all garments owned by one user.
	ListClothesByUser(ctx context.Context, userID string) ([]domain.Garment, error)
}
