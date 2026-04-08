package handler

import (
	"weicloth/internal/adapters/handler/clothe"
	"weicloth/internal/adapters/handler/user"
	"weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	user   *user.HTTPHandler
	clothe *clothe.HTTPHandler
}

func NewHTTPHandler(userService *services.UserService, clotheService *services.ClotheService) *HTTPHandler {
	return &HTTPHandler{
		user:   user.NewHTTPHandler(userService),
		clothe: clothe.NewHTTPHandler(clotheService),
	}
}

func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/wei")
	user.RegisterRoutes(v1.Group("/users"), h.user)
	clothe.RegisterRoutes(v1.Group("/clothes"), h.clothe)
}
