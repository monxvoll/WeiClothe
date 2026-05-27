package postgres

import (
	"context"
	"errors"
	"fmt"

	"weicloth/internal/core/apperrors"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StyleRepository struct {
	db *pgxpool.Pool
}

var _ ports.StyleRepository = (*StyleRepository)(nil)

func NewStyleRepository(db *pgxpool.Pool) *StyleRepository {
	return &StyleRepository{db: db}
}

func (r *StyleRepository) GetUserStylePreferences(ctx context.Context, userID string) (*domain.UserStylePreferences, error) {
	query := `
		SELECT preferred_colors, preferred_occasions, preferred_seasons, avoid_colors
		FROM user_style_preferences
		WHERE user_id = $1
	`
	var preferredColors, preferredOccasions, preferredSeasons, avoidColors []string
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&preferredColors,
		&preferredOccasions,
		&preferredSeasons,
		&avoidColors,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("get user style preferences: %w", err)
	}

	return &domain.UserStylePreferences{
		UserID:             userID,
		PreferredColors:    preferredColors,
		PreferredOccasions: preferredOccasions,
		PreferredSeasons:   preferredSeasons,
		AvoidColors:        avoidColors,
	}, nil
}

func (r *StyleRepository) UpsertUserStylePreferences(ctx context.Context, prefs *domain.UserStylePreferences) error {
	if prefs == nil || prefs.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	preferredColors := prefs.PreferredColors
	if preferredColors == nil {
		preferredColors = []string{}
	}
	preferredOccasions := prefs.PreferredOccasions
	if preferredOccasions == nil {
		preferredOccasions = []string{}
	}
	preferredSeasons := prefs.PreferredSeasons
	if preferredSeasons == nil {
		preferredSeasons = []string{}
	}
	avoidColors := prefs.AvoidColors
	if avoidColors == nil {
		avoidColors = []string{}
	}

	query := `
		INSERT INTO user_style_preferences (
			user_id, preferred_colors, preferred_occasions, preferred_seasons, avoid_colors, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			preferred_colors = EXCLUDED.preferred_colors,
			preferred_occasions = EXCLUDED.preferred_occasions,
			preferred_seasons = EXCLUDED.preferred_seasons,
			avoid_colors = EXCLUDED.avoid_colors,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query,
		prefs.UserID,
		preferredColors,
		preferredOccasions,
		preferredSeasons,
		avoidColors,
	)
	if err != nil {
		return fmt.Errorf("upsert user style preferences: %w", err)
	}
	return nil
}
