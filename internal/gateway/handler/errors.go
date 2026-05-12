package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func writeError(c *gin.Context, err error) {
	statusCode := statusCodeForError(err)
	slog.ErrorContext(
		c.Request.Context(),
		"http request failed",
		slog.String("method", c.Request.Method),
		slog.String("path", c.FullPath()),
		slog.Int("status_code", statusCode),
		slog.String("error", err.Error()),
	)

	c.JSON(statusCode, gin.H{"error": err.Error()})
}

func statusCodeForError(err error) int {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, gatewayservice.ErrTimeout):
		statusCode = http.StatusGatewayTimeout
	case errors.Is(err, gatewayservice.ErrRequestCanceled):
		statusCode = http.StatusRequestTimeout
	case errors.Is(err, gatewayservice.ErrInvalidInput),
		errors.Is(err, gatewayservice.ErrIdempotencyConflict),
		errors.Is(err, gatewayservice.ErrUnsupportedCurrency),
		errors.Is(err, gatewayservice.ErrUnsupportedPaymentMethod):
		statusCode = http.StatusBadRequest
	case errors.Is(err, gatewayservice.ErrDownstreamNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, gatewayservice.ErrDownstreamFailed):
		statusCode = http.StatusBadGateway
	}

	if errors.Is(err, gatewayservice.ErrIdempotencyConflict) {
		statusCode = http.StatusConflict
	}

	return statusCode
}
