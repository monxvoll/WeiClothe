package user

import (
	"net/http"
	"weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
)

type (
	HTTPHandler struct {
		userService *services.UserService
	}
)

func NewHTTPHandler(userService *services.UserService) *HTTPHandler {
	return &HTTPHandler{
		userService: userService,
	}
}

func (h *HTTPHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "pong"})
}
