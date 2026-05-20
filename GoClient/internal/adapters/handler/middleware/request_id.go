package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type ctxKey struct{}

// RequestIDMiddleware reads X-Request-ID from the incoming request (or generates
// a random one) and attaches a child *slog.Logger enriched with "request_id" to
// the request context. Downstream code retrieves it via LoggerFromCtx.
func RequestIDMiddleware(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = newID()
		}
		c.Header("X-Request-ID", rid)

		reqLogger := base.With("request_id", rid)
		ctx := context.WithValue(c.Request.Context(), ctxKey{}, reqLogger)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// LoggerFromCtx extracts the per-request *slog.Logger stored by RequestIDMiddleware.
// Falls back to slog.Default() when called outside an HTTP context.
func LoggerFromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// UUID v4 layout
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
