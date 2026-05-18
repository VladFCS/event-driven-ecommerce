package handler

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

type GatewayService interface {
	Checkout(ctx context.Context, in *gatewayservice.CheckoutInput) (*gatewayservice.CheckoutResult, error)
	CancelOrder(ctx context.Context, in *gatewayservice.CancelOrderInput) (*gatewayservice.CancelOrderResult, error)
	CancelPayment(ctx context.Context, in *gatewayservice.CancelPaymentInput) (*gatewayservice.CancelPaymentResult, error)
	CreateProduct(ctx context.Context, in *gatewayservice.CreateProductInput) (*gatewayservice.CreateProductResult, error)
	UpdateProduct(ctx context.Context, in *gatewayservice.UpdateProductInput) (*gatewayservice.UpdateProductResult, error)
	DeleteProduct(ctx context.Context, in *gatewayservice.DeleteProductInput) error
	GetOrderByID(ctx context.Context, in *gatewayservice.GetOrderByIDInput) (*gatewayservice.GetOrderByIDResult, error)
	GetPaymentByID(ctx context.Context, in *gatewayservice.GetPaymentByIDInput) (*gatewayservice.GetPaymentByIDResult, error)
	GetPaymentByOrderID(ctx context.Context, in *gatewayservice.GetPaymentByOrderIDInput) (*gatewayservice.GetPaymentByOrderIDResult, error)
	ListPaymentsByCustomer(ctx context.Context, in *gatewayservice.ListPaymentsByCustomerInput) (*gatewayservice.ListPaymentsByCustomerResult, error)
	GetProductByID(ctx context.Context, in *gatewayservice.GetProductByIDInput) (*gatewayservice.GetProductByIDResult, error)
	ListProducts(ctx context.Context, in *gatewayservice.ListProductsInput) (*gatewayservice.ListProductsResult, error)
	GetStockByProductID(ctx context.Context, in *gatewayservice.GetStockByProductIDInput) (*gatewayservice.GetStockByProductIDResult, error)
	ListOrdersByCustomer(ctx context.Context, in *gatewayservice.ListOrdersByCustomerInput) (*gatewayservice.ListOrdersByCustomerResult, error)
	ReadinessStatus() gatewayservice.ReadinessStatus
}

type HTTPHandler struct {
	gatewayService GatewayService
	logger         *slog.Logger
}

func NewHTTPHandler(gatewayService GatewayService, logger *slog.Logger) *HTTPHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &HTTPHandler{
		gatewayService: gatewayService,
		logger:         logger,
	}
}

func (h *HTTPHandler) Register(r *gin.Engine) {
	r.Use(requestIDMiddleware())
	r.Use(requestMetricsMiddleware())
	r.Use(requestLoggingMiddleware(h.logger))
	r.Use(gin.Recovery())

	r.GET("/metrics", metricsHandler())
	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)

	r.POST("/checkout", h.Checkout)
	r.POST("/orders/:order_id/cancel", h.CancelOrder)
	r.POST("/payments/:payment_id/cancel", h.CancelPayment)
	r.GET("/orders/:order_id", h.GetOrderByID)
	r.GET("/orders/:order_id/payment", h.GetOrderPayment)
	r.GET("/customers/:customer_id/orders", h.ListOrdersByCustomer)
	r.GET("/customers/:customer_id/payments", h.ListPaymentsByCustomer)

	r.GET("/payments/:payment_id", h.GetPaymentByID)
	r.GET("/payments/order/:order_id", h.GetPaymentByOrderID)
	r.GET("/catalog/products/:product_id", h.GetProductByID)
	r.GET("/catalog/products", h.ListProducts)
	r.POST("/catalog/products", h.CreateProduct)
	r.PATCH("/catalog/products/:product_id", h.UpdateProduct)
	r.DELETE("/catalog/products/:product_id", h.DeleteProduct)
	r.GET("/inventory/products/:product_id/stock", h.GetStockByProductID)
}
