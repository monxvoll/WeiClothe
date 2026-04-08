package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *HTTPHandler) {
	g.GET("/ping", h.Ping)
}
