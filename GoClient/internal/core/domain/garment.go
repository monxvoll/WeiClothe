package domain

import "time"

type Garment struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	ImageURL         string     `json:"image_url"`
	GarmentType      string     `json:"garment_type"`
	Name             string     `json:"name"`
	ClassificationID string     `json:"classification_id"`
	Category         string     `json:"category"`
	Subcategory      string     `json:"subcategory"`
	Color            string     `json:"color"`
	Pattern          string     `json:"pattern"`
	Material         string     `json:"material"`
	Season           string     `json:"season"`
	Occasion         string     `json:"occasion"`
	Confidence       *float64   `json:"confidence"`
	Source           string     `json:"source"`
	ModelName        string     `json:"model_name"`
	ModelVersion     string     `json:"model_version"`
	Status           string     `json:"status"`
	ProcessingError  *string    `json:"processing_error"`
	ProcessedAt      *time.Time `json:"processed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// OutfitRecommendation represents a complete clothing combination suggested to the user.
type OutfitRecommendation struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Top         Garment          `json:"top"`
	Bottom      Garment          `json:"bottom"`
	Footwear    Garment          `json:"footwear"`
	Score       float64          `json:"score"`
	Reasoning   OutfitReasoning  `json:"reasoning"`
	Description string           `json:"description"`
}
