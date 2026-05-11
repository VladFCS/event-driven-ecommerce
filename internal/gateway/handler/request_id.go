package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladfc/event-driven-ecommerce-app/internal/gateway/requestid"
)

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestid.Header))
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := requestid.WithContext(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(requestid.Header, requestID)

		c.Next()
	}
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}

	return hex.EncodeToString(buf)
}

func RequestIDFromContext(ctx context.Context) string {
	return requestid.FromContext(ctx)
}
