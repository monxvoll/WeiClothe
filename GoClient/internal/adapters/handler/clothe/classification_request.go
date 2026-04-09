package clothe

// SaveClassificationRequest is the PATCH body for persisting labels and model metadata on an existing row.
type SaveClassificationRequest struct {
	ClassificationID string   `json:"classification_id"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Subcategory      string   `json:"subcategory"`
	Color            string   `json:"color"`
	Pattern          string   `json:"pattern"`
	Material         string   `json:"material"`
	Season           string   `json:"season"`
	Occasion         string   `json:"occasion"`
	Confidence       *float64 `json:"confidence"`
	Source           string   `json:"source"`
	ModelName        string   `json:"model_name"`
	ModelVersion     string   `json:"model_version"`
	Status           string   `json:"status"`
	ProcessingError  *string  `json:"processing_error"`
	// ProcessedAt is optional RFC3339 or RFC3339Nano; empty/absent lets the repository set a default.
	ProcessedAt *string `json:"processed_at"`
}
