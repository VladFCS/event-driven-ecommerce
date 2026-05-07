package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func writeError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, gatewayservice.ErrInvalidInput),
		errors.Is(err, gatewayservice.ErrUnsupportedCurrency),
		errors.Is(err, gatewayservice.ErrUnsupportedPaymentMethod):
		statusCode = http.StatusBadRequest
	case errors.Is(err, gatewayservice.ErrDownstreamNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, gatewayservice.ErrDownstreamFailed):
		statusCode = http.StatusBadGateway
	}

	c.JSON(statusCode, gin.H{"error": err.Error()})
}
