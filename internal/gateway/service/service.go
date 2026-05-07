package service

import (
	"context"

	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
)

type OrderClient interface {
	CreateOrder(ctx context.Context, req *orderclient.CreateOrderRequest) (*orderclient.CreateOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*orderclient.GetOrderResponse, error)
	CancelOrder(ctx context.Context, req *orderclient.CancelOrderRequest) (*orderclient.CancelOrderResponse, error)
}

type InventoryClient interface {
	ReserveStock(ctx context.Context, req *inventoryclient.ReserveStockRequest) (*inventoryclient.ReserveStockResponse, error)
	ReleaseStock(ctx context.Context, req *inventoryclient.ReleaseStockRequest) (*inventoryclient.ReleaseStockResponse, error)
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, req *paymentclient.CreatePaymentRequest) (*paymentclient.CreatePaymentResponse, error)
}

type GatewayService struct {
	orderClient     OrderClient
	inventoryClient InventoryClient
	paymentClient   PaymentClient
}

type Option func(*GatewayService)

func WithInventoryClient(client InventoryClient) Option {
	return func(s *GatewayService) {
		s.inventoryClient = client
	}
}

func WithPaymentClient(client PaymentClient) Option {
	return func(s *GatewayService) {
		s.paymentClient = client
	}
}

func NewGatewayService(orderClient OrderClient, opts ...Option) *GatewayService {
	service := &GatewayService{
		orderClient: orderClient,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}

	return service
}
