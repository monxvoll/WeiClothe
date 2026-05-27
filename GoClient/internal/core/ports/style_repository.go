package ports

import (
	"context"

	"weicloth/internal/core/domain"
)

// StyleRepository persists user style preferences for recommendations.
type StyleRepository interface {
	GetUserStylePreferences(ctx context.Context, userID string) (*domain.UserStylePreferences, error)
	UpsertUserStylePreferences(ctx context.Context, prefs *domain.UserStylePreferences) error
}
