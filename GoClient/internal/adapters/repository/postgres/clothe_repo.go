package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"weicloth/internal/core/apperrors"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClotheRepository struct {
	db *pgxpool.Pool
}

var _ ports.ClotheRepository = (*ClotheRepository)(nil)

func NewClotheRepository(db *pgxpool.Pool) *ClotheRepository {
	return &ClotheRepository{db: db}
}

func (pgx *ClotheRepository) CreateClothe(ctx context.Context, garment *domain.Garment) error {
	query := `
		INSERT INTO clothes (
			user_id, image_url, garment_type, name, classification_id,
			category, subcategory, color, pattern, material, season, occasion,
			confidence, source, model_name, model_version, status
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)
		RETURNING id, created_at, updated_at
	`

	// Keep defaults aligned with schema constraints.
	source := garment.Source
	if source == "" {
		source = "ai"
	}
	status := garment.Status
	if status == "" {
		status = "queued"
	}

	var imageURL any
	if garment.ImageURL != "" {
		imageURL = garment.ImageURL
	}

	var id int
	err := pgx.db.QueryRow(
		ctx,
		query,
		garment.UserID,
		imageURL,
		garment.GarmentType,
		garment.Name,
		garment.ClassificationID,
		garment.Category,
		garment.Subcategory,
		garment.Color,
		garment.Pattern,
		garment.Material,
		garment.Season,
		garment.Occasion,
		garment.Confidence,
		source,
		garment.ModelName,
		garment.ModelVersion,
		status,
	).Scan(&id, &garment.CreatedAt, &garment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create clothe: %w", mapPgErr(err))
	}

	garment.ID = strconv.Itoa(id)
	garment.Source = source
	garment.Status = status

	return nil
}

func (pgx *ClotheRepository) UpdateClotheStatus(ctx context.Context, garmentID string, status string) error {
	id, err := strconv.Atoi(garmentID)
	if err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrInvalidID, err)
	}

	query := `
		UPDATE clothes
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	cmd, err := pgx.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update clothe status: %w", mapPgErr(err))
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%w", apperrors.ErrNotFound)
	}

	return nil
}

func (pgx *ClotheRepository) SaveClassification(ctx context.Context, garment *domain.Garment) error {
	id, err := strconv.Atoi(garment.ID)
	if err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrInvalidID, err)
	}

	query := `
		UPDATE clothes
		SET
			classification_id = $1,
			name = $2,
			category = $3,
			subcategory = $4,
			color = $5,
			pattern = $6,
			material = $7,
			season = $8,
			occasion = $9,
			confidence = $10,
			source = $11,
			model_name = $12,
			model_version = $13,
			status = $14,
			processing_error = $15,
			processed_at = $16,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $17
	`

	status := garment.Status
	if status == "" {
		status = "completed"
	}
	source := garment.Source
	if source == "" {
		source = "ai"
	}
	processedAt := garment.ProcessedAt
	if processedAt == nil {
		now := time.Now().UTC()
		processedAt = &now
	}

	cmd, err := pgx.db.Exec(
		ctx,
		query,
		garment.ClassificationID,
		garment.Name,
		garment.Category,
		garment.Subcategory,
		garment.Color,
		garment.Pattern,
		garment.Material,
		garment.Season,
		garment.Occasion,
		garment.Confidence,
		source,
		garment.ModelName,
		garment.ModelVersion,
		status,
		garment.ProcessingError,
		processedAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to save clothe classification: %w", mapPgErr(err))
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%w", apperrors.ErrNotFound)
	}

	garment.Status = status
	garment.Source = source
	garment.ProcessedAt = processedAt

	return nil
}

func (pgx *ClotheRepository) GetClotheByID(ctx context.Context, garmentID string) (*domain.Garment, error) {
	id, err := strconv.Atoi(garmentID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidID, err)
	}

	query := `
		SELECT
			id, user_id, image_url, garment_type, name, classification_id,
			category, subcategory, color, pattern, material, season, occasion,
			confidence, source, model_name, model_version, status, processing_error,
			processed_at, created_at, updated_at
		FROM clothes
		WHERE id = $1
	`

	garment := &domain.Garment{}
	var dbID int
	var imageURL sql.NullString
	var name sql.NullString
	var classificationID sql.NullString
	var category sql.NullString
	var subcategory sql.NullString
	var color sql.NullString
	var pattern sql.NullString
	var material sql.NullString
	var season sql.NullString
	var occasion sql.NullString
	var confidence sql.NullFloat64
	var source sql.NullString
	var modelName sql.NullString
	var modelVersion sql.NullString
	var status sql.NullString
	var processingErr sql.NullString
	var processedAt sql.NullTime

	err = pgx.db.QueryRow(ctx, query, id).Scan(
		&dbID,
		&garment.UserID,
		&imageURL,
		&garment.GarmentType,
		&name,
		&classificationID,
		&category,
		&subcategory,
		&color,
		&pattern,
		&material,
		&season,
		&occasion,
		&confidence,
		&source,
		&modelName,
		&modelVersion,
		&status,
		&processingErr,
		&processedAt,
		&garment.CreatedAt,
		&garment.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w", apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get clothe by id: %w", err)
	}

	garment.ID = strconv.Itoa(dbID)
	if imageURL.Valid {
		garment.ImageURL = imageURL.String
	}
	if name.Valid {
		garment.Name = name.String
	}
	if classificationID.Valid {
		garment.ClassificationID = classificationID.String
	}
	if category.Valid {
		garment.Category = category.String
	}
	if subcategory.Valid {
		garment.Subcategory = subcategory.String
	}
	if color.Valid {
		garment.Color = color.String
	}
	if pattern.Valid {
		garment.Pattern = pattern.String
	}
	if material.Valid {
		garment.Material = material.String
	}
	if season.Valid {
		garment.Season = season.String
	}
	if occasion.Valid {
		garment.Occasion = occasion.String
	}
	if confidence.Valid {
		v := confidence.Float64
		garment.Confidence = &v
	}
	if source.Valid {
		garment.Source = source.String
	}
	if modelName.Valid {
		garment.ModelName = modelName.String
	}
	if modelVersion.Valid {
		garment.ModelVersion = modelVersion.String
	}
	if status.Valid {
		garment.Status = status.String
	}
	if processingErr.Valid {
		v := processingErr.String
		garment.ProcessingError = &v
	}
	if processedAt.Valid {
		t := processedAt.Time
		garment.ProcessedAt = &t
	}

	return garment, nil
}

func (pgx *ClotheRepository) ListClothesByUser(ctx context.Context, userID string) ([]domain.Garment, error) {
	query := `
		SELECT
			id, user_id, image_url, garment_type, name, classification_id,
			category, subcategory, color, pattern, material, season, occasion,
			confidence, source, model_name, model_version, status, processing_error,
			processed_at, created_at, updated_at
		FROM clothes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := pgx.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clothes by user: %w", err)
	}
	defer rows.Close()

	garments := make([]domain.Garment, 0)
	for rows.Next() {
		var garment domain.Garment
		var dbID int
		var imageURL sql.NullString
		var name sql.NullString
		var classificationID sql.NullString
		var category sql.NullString
		var subcategory sql.NullString
		var color sql.NullString
		var pattern sql.NullString
		var material sql.NullString
		var season sql.NullString
		var occasion sql.NullString
		var confidence sql.NullFloat64
		var source sql.NullString
		var modelName sql.NullString
		var modelVersion sql.NullString
		var status sql.NullString
		var processingErr sql.NullString
		var processedAt sql.NullTime

		if err := rows.Scan(
			&dbID,
			&garment.UserID,
			&imageURL,
			&garment.GarmentType,
			&name,
			&classificationID,
			&category,
			&subcategory,
			&color,
			&pattern,
			&material,
			&season,
			&occasion,
			&confidence,
			&source,
			&modelName,
			&modelVersion,
			&status,
			&processingErr,
			&processedAt,
			&garment.CreatedAt,
			&garment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan clothe row: %w", err)
		}

		garment.ID = strconv.Itoa(dbID)
		if imageURL.Valid {
			garment.ImageURL = imageURL.String
		}
		if name.Valid {
			garment.Name = name.String
		}
		if classificationID.Valid {
			garment.ClassificationID = classificationID.String
		}
		if category.Valid {
			garment.Category = category.String
		}
		if subcategory.Valid {
			garment.Subcategory = subcategory.String
		}
		if color.Valid {
			garment.Color = color.String
		}
		if pattern.Valid {
			garment.Pattern = pattern.String
		}
		if material.Valid {
			garment.Material = material.String
		}
		if season.Valid {
			garment.Season = season.String
		}
		if occasion.Valid {
			garment.Occasion = occasion.String
		}
		if confidence.Valid {
			v := confidence.Float64
			garment.Confidence = &v
		}
		if source.Valid {
			garment.Source = source.String
		}
		if modelName.Valid {
			garment.ModelName = modelName.String
		}
		if modelVersion.Valid {
			garment.ModelVersion = modelVersion.String
		}
		if status.Valid {
			garment.Status = status.String
		}
		if processingErr.Valid {
			v := processingErr.String
			garment.ProcessingError = &v
		}
		if processedAt.Valid {
			t := processedAt.Time
			garment.ProcessedAt = &t
		}

		garments = append(garments, garment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating clothes: %w", err)
	}

	return garments, nil
}
