// Package auth provides HTTP middleware for validating Bearer tokens via IdentityProvider.
package auth

import (
	"net/http"
	"strings"
	"weicloth/internal/core/ports"

	"github.com/gin-gonic/gin"
)

const contextSubjectKey = "auth_subject"

// Subject returns the Keycloak user id (sub) set by BearerMiddleware, if present.
func Subject(c *gin.Context) (string, bool) {
	v, ok := c.Get(contextSubjectKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

// BearerMiddleware validates Authorization: Bearer <JWT> and stores the subject in the Gin context.
func BearerMiddleware(idp ports.IdentityProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		const prefix = "Bearer "
		if len(raw) < len(prefix) || !strings.EqualFold(raw[:len(prefix)], prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header; expected Bearer token"})
			return
		}
		token := strings.TrimSpace(raw[len(prefix):])
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "empty bearer token"})
			return
		}

		uid, err := idp.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(contextSubjectKey, uid)
		c.Next()
	}
}
