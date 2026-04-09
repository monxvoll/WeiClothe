package clothe

// RegisterClotheRequest is the POST body for initial garment persistence before classification.
// The owner user_id is taken from the JWT subject, not from the body.
type RegisterClotheRequest struct {
	ImageURL    string `json:"image_url" binding:"required"`
	GarmentType string `json:"garment_type" binding:"required"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Status      string `json:"status"`
}
