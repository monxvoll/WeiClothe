package clothe

import (
	"weicloth/internal/adapters/handler/auth"
	"weicloth/internal/core/ports"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts garment routes on g (typically /clothes under the API prefix).
// All routes require a valid Bearer JWT validated through idp.
func RegisterRoutes(g *gin.RouterGroup, h *HTTPHandler, idp ports.IdentityProvider) {
	secured := g.Group("")
	secured.Use(auth.BearerMiddleware(idp))
	secured.GET("/ping", h.Ping)
	secured.POST("", h.Register)
	secured.GET("", h.ListByUser)
	secured.GET("/:id", h.GetByID)
	secured.PATCH("/:id/status", h.UpdateStatus)
	secured.PATCH("/:id/classification", h.SaveClassification)
	secured.GET("/recommendations", h.GetRecommendations)
}
