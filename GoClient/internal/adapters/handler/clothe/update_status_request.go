package clothe

// UpdateStatusRequest is the PATCH body for status transitions; audit user id comes from the JWT.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
