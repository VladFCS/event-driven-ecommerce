package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vladfc/event-driven-ecommerce-app/internal/gateway/requestid"
)

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestid.Header))
		if requestID == "" {
			requestID = requestid.Generate()
		}

		ctx := requestid.WithContext(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(requestid.Header, requestID)

		c.Next()
	}
}

func RequestIDFromContext(ctx context.Context) string {
	return requestid.FromContext(ctx)
}
