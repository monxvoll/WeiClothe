package httperrors

import (
	"errors"
	"net/http"

	"weicloth/internal/core/apperrors"

	"github.com/gin-gonic/gin"
)

// WriteServiceError maps sentinel errors to HTTP status codes without leaking internals.
func WriteServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	case errors.Is(err, apperrors.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, apperrors.ErrInvalidID), errors.Is(err, apperrors.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": clientMessage(err)})
	case errors.Is(err, apperrors.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func clientMessage(err error) string {
	var v *apperrors.ValidationError
	if errors.As(err, &v) && v.Message != "" {
		return v.Message
	}
	if errors.Is(err, apperrors.ErrInvalidID) {
		return "invalid id"
	}
	return "invalid input"
}
