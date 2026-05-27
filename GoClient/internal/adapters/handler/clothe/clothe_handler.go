// Package clothe exposes REST handlers for garment registration, classification, and queries.
package clothe

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"weicloth/internal/adapters/handler/auth"
	"weicloth/internal/adapters/handler/httperrors"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
)

// HTTPHandler serves garment endpoints backed by ClotheService.
type HTTPHandler struct {
	clotheService *services.ClotheService
}

// NewHTTPHandler constructs a handler with the given service.
func NewHTTPHandler(clotheService *services.ClotheService) *HTTPHandler {
	return &HTTPHandler{
		clotheService: clotheService,
	}
}

// Ping reports that the clothes routes are reachable.
func (h *HTTPHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "pong"})
}

// Register creates a garment from multipart/form-data (field `image` + `garment_type`, optional name/source/status).
func (h *HTTPHandler) Register(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	const multipartOverhead = 1024 * 1024 // small margin for multipart boundaries
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRegisterImageBytes+multipartOverhead)
	if err := c.Request.ParseMultipartForm(MaxRegisterImageBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart form required or payload too large"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}

	garmentType, err := domain.ResolveGarmentType(c.PostForm(RegisterMultipartFieldGarmentType))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusField := strings.TrimSpace(c.PostForm("status"))
	if err := domain.ValidateOptionalGarmentStatus(statusField); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sourceField := strings.TrimSpace(c.PostForm("source"))
	if err := domain.ValidateOptionalGarmentSource(sourceField); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, header, err := c.Request.FormFile(RegisterMultipartFieldImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required (form field: image)"})
		return
	}
	defer file.Close()

	if header.Size > MaxRegisterImageBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image exceeds maximum size of 10MB"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read image"})
		return
	}
	if int64(len(data)) > MaxRegisterImageBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image exceeds maximum size of 10MB"})
		return
	}

	detected := http.DetectContentType(data)
	ext, contentType, valid := garmentImageExtFromMIME(detected)
	if !valid {
		ext, contentType, valid = garmentImageExtFromFilename(header.Filename)
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type; use jpeg, png, or webp"})
		return
	}

	garment := &domain.Garment{
		UserID:      subject,
		GarmentType: garmentType,
		Name:        strings.TrimSpace(c.PostForm("name")),
		Source:      sourceField,
		Status:      statusField,
	}

	if err := h.clotheService.RegisterClothe(c.Request.Context(), garment, data, ext, contentType); err != nil {
		writeClotheServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, garment)
}

func garmentImageExtFromMIME(detected string) (ext string, contentType string, ok bool) {
	switch detected {
	case "image/jpeg":
		return ".jpg", "image/jpeg", true
	case "image/png":
		return ".png", "image/png", true
	case "image/webp":
		return ".webp", "image/webp", true
	default:
		return "", "", false
	}
}

func garmentImageExtFromFilename(name string) (ext string, contentType string, ok bool) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return ".jpg", "image/jpeg", true
	case ".png":
		return ".png", "image/png", true
	case ".webp":
		return ".webp", "image/webp", true
	default:
		return "", "", false
	}
}

// UpdateStatus changes lifecycle status for :id; audit user id is the JWT subject.
func (h *HTTPHandler) UpdateStatus(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	garmentID := c.Param("id")
	if _, ok := h.requireGarmentOwner(c, garmentID, subject); !ok {
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := domain.ValidateGarmentStatus(req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clotheService.UpdateClotheStatus(c.Request.Context(), garmentID, req.Status, subject); err != nil {
		writeClotheServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// SaveClassification persists model or manual labels for :id; repository applies defaults for omitted timing fields.
func (h *HTTPHandler) SaveClassification(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	garmentID := c.Param("id")
	if _, ok := h.requireGarmentOwner(c, garmentID, subject); !ok {
		return
	}

	var req SaveClassificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := domain.ValidateOptionalGarmentStatus(req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := domain.ValidateOptionalGarmentSource(req.Source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var processedAt *time.Time
	if req.ProcessedAt != nil && *req.ProcessedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, *req.ProcessedAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, *req.ProcessedAt)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid processed_at; use RFC3339"})
				return
			}
		}
		processedAt = &t
	}

	garment := &domain.Garment{
		ID:               garmentID,
		ClassificationID: req.ClassificationID,
		Name:             req.Name,
		Category:         req.Category,
		Subcategory:      req.Subcategory,
		Color:            req.Color,
		Pattern:          req.Pattern,
		Material:         req.Material,
		Season:           req.Season,
		Occasion:         req.Occasion,
		Confidence:       req.Confidence,
		Source:           req.Source,
		ModelName:        req.ModelName,
		ModelVersion:     req.ModelVersion,
		Status:           req.Status,
		ProcessingError:  req.ProcessingError,
		ProcessedAt:      processedAt,
	}

	if err := h.clotheService.SaveClassification(c.Request.Context(), garment); err != nil {
		writeClotheServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "classification saved"})
}

// GetByID returns a single garment by numeric :id if the caller owns it.
func (h *HTTPHandler) GetByID(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	garmentID := c.Param("id")
	garment, ok := h.requireGarmentOwner(c, garmentID, subject)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, garment)
}

type listClothesQuery struct {
	// UserID is the owner filter (required query parameter).
	UserID string `form:"user_id" binding:"required"`
}

// ListByUser returns all garments for the given user_id query parameter, newest first.
func (h *HTTPHandler) ListByUser(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	var q listClothesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if q.UserID != subject {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot list garments for another user"})
		return
	}

	garments, err := h.clotheService.ListClothesByUser(c.Request.Context(), q.UserID)
	if err != nil {
		writeClotheServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, garments)
}

// authenticatedSubject returns the JWT subject set by BearerMiddleware, or 401 if missing.
func (h *HTTPHandler) authenticatedSubject(c *gin.Context) (string, bool) {
	sub, ok := auth.Subject(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return "", false
	}
	return sub, true
}

// requireGarmentOwner loads the garment and rejects the request with 403 if subject is not its owner.
// On repository errors it uses the same mapping as other garment handlers. Returns (nil, false) when the response is already written.
func (h *HTTPHandler) requireGarmentOwner(c *gin.Context, garmentID, subject string) (*domain.Garment, bool) {
	garment, err := h.clotheService.GetClotheByID(c.Request.Context(), garmentID)
	if err != nil {
		writeClotheServiceError(c, err)
		return nil, false
	}
	if garment.UserID != subject {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not own this garment"})
		return nil, false
	}
	return garment, true
}

// writeClotheServiceError maps repository/service errors to 400 / 404 / 500 without leaking internals.
func writeClotheServiceError(c *gin.Context, err error) {
	httperrors.WriteServiceError(c, err)
}

type recommendationsQuery struct {
	UserID   string `form:"user_id" binding:"required"`
	Season   string `form:"season"`
	Occasion string `form:"occasion"`
	Limit    int    `form:"limit"`
}

// GetRecommendations returns metadata-scored outfit combinations for the user's wardrobe.
func (h *HTTPHandler) GetRecommendations(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	var q recommendationsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if q.UserID != subject {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot get recommendations for another user"})
		return
	}

	req := domain.RecommendationRequest{
		UserID:   q.UserID,
		Season:   strings.TrimSpace(q.Season),
		Occasion: strings.TrimSpace(q.Occasion),
		Limit:    q.Limit,
	}

	recommendations, err := h.clotheService.GetRecommendations(c.Request.Context(), req)
	if err != nil {
		writeClotheServiceError(c, err)
		return
	}

	if recommendations == nil {
		recommendations = []domain.OutfitRecommendation{}
	}

	c.JSON(http.StatusOK, recommendations)
}

type stylePreferencesQuery struct {
	UserID string `form:"user_id" binding:"required"`
}

type upsertStylePreferencesRequest struct {
	PreferredColors    []string `json:"preferred_colors"`
	PreferredOccasions []string `json:"preferred_occasions"`
	PreferredSeasons   []string `json:"preferred_seasons"`
	AvoidColors        []string `json:"avoid_colors"`
}

// GetStylePreferences returns the user's saved palette and style filters.
func (h *HTTPHandler) GetStylePreferences(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	var q stylePreferencesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.UserID != subject {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot get preferences for another user"})
		return
	}

	prefs, err := h.clotheService.GetUserStylePreferences(c.Request.Context(), q.UserID)
	if err != nil {
		writeClotheServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

// PutStylePreferences creates or updates the user's palette and style filters.
func (h *HTTPHandler) PutStylePreferences(c *gin.Context) {
	subject, ok := h.authenticatedSubject(c)
	if !ok {
		return
	}

	var q stylePreferencesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.UserID != subject {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot update preferences for another user"})
		return
	}

	var body upsertStylePreferencesRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := &domain.UserStylePreferences{
		UserID:             q.UserID,
		PreferredColors:    body.PreferredColors,
		PreferredOccasions: body.PreferredOccasions,
		PreferredSeasons:   body.PreferredSeasons,
		AvoidColors:        body.AvoidColors,
	}

	if err := h.clotheService.UpsertUserStylePreferences(c.Request.Context(), prefs); err != nil {
		writeClotheServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}
