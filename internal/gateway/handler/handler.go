package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

type CheckoutService interface {
	Checkout(ctx context.Context, in *gatewayservice.CheckoutInput) (*gatewayservice.CheckoutResult, error)
	GetOrderByID(ctx context.Context, in *gatewayservice.GetOrderByIDInput) (*gatewayservice.GetOrderByIDResult, error)
	ReadinessStatus() gatewayservice.ReadinessStatus
}

type HTTPHandler struct {
	gatewayService CheckoutService
}

func NewHTTPHandler(gatewayService CheckoutService) *HTTPHandler {
	return &HTTPHandler{gatewayService: gatewayService}
}

func (h *HTTPHandler) Register(r *gin.Engine) {
	r.Use(requestIDMiddleware())

	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)

	r.POST("/checkout", h.Checkout)
	r.GET("/orders/:order_id", h.GetOrderByID)
}
