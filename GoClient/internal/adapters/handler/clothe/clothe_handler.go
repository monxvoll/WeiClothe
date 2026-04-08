package clothe

import (
	"net/http"
	"weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	clotheService *services.ClotheService
}

func NewHTTPHandler(clotheService *services.ClotheService) *HTTPHandler {
	return &HTTPHandler{
		clotheService: clotheService,
	}
}

func (h *HTTPHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "pong"})
}
