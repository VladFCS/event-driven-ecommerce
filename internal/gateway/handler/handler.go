package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

type GatewayService interface {
	Checkout(ctx context.Context, in *gatewayservice.CheckoutInput) (*gatewayservice.CheckoutResult, error)
	CancelOrder(ctx context.Context, in *gatewayservice.CancelOrderInput) (*gatewayservice.CancelOrderResult, error)
	GetOrderByID(ctx context.Context, in *gatewayservice.GetOrderByIDInput) (*gatewayservice.GetOrderByIDResult, error)
	GetPaymentByID(ctx context.Context, in *gatewayservice.GetPaymentByIDInput) (*gatewayservice.GetPaymentByIDResult, error)
	GetProductByID(ctx context.Context, in *gatewayservice.GetProductByIDInput) (*gatewayservice.GetProductByIDResult, error)
	GetStockByProductID(ctx context.Context, in *gatewayservice.GetStockByProductIDInput) (*gatewayservice.GetStockByProductIDResult, error)
	ListOrdersByCustomer(ctx context.Context, in *gatewayservice.ListOrdersByCustomerInput) (*gatewayservice.ListOrdersByCustomerResult, error)
	ReadinessStatus() gatewayservice.ReadinessStatus
}

type HTTPHandler struct {
	gatewayService GatewayService
}

func NewHTTPHandler(gatewayService GatewayService) *HTTPHandler {
	return &HTTPHandler{gatewayService: gatewayService}
}

func (h *HTTPHandler) Register(r *gin.Engine) {
	r.Use(requestIDMiddleware())

	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)

	r.POST("/checkout", h.Checkout)
	r.POST("/orders/:order_id/cancel", h.CancelOrder)
	r.GET("/orders/:order_id", h.GetOrderByID)
	r.GET("/customers/:customer_id/orders", h.ListOrdersByCustomer)

	r.GET("/payments/:payment_id", h.GetPaymentByID)
	r.GET("/catalog/products/:product_id", h.GetProductByID)
	r.GET("/inventory/products/:product_id/stock", h.GetStockByProductID)
}
