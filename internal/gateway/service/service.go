package service

import (
	"context"
	"time"

	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
)

type OrderClient interface {
	CreateOrder(ctx context.Context, req *orderclient.CreateOrderRequest) (*orderclient.CreateOrderResponse, error)
	GetOrderByID(ctx context.Context, orderID string) (*orderclient.GetOrderByIDResponse, error)
	CancelOrder(ctx context.Context, req *orderclient.CancelOrderRequest) (*orderclient.CancelOrderResponse, error)
	ListOrdersByCustomer(ctx context.Context, req *orderclient.ListOrdersByCustomerRequest) (*orderclient.ListOrdersByCustomerResponse, error)
}

type InventoryClient interface {
	ReserveStock(ctx context.Context, req *inventoryclient.ReserveStockRequest) (*inventoryclient.ReserveStockResponse, error)
	ReleaseStock(ctx context.Context, req *inventoryclient.ReleaseStockRequest) (*inventoryclient.ReleaseStockResponse, error)
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, req *paymentclient.CreatePaymentRequest) (*paymentclient.CreatePaymentResponse, error)
	GetPaymentByID(ctx context.Context, req *paymentclient.GetPaymentByIDRequest) (*paymentclient.GetPaymentByIDResponse, error)
}

type ReadinessStatus struct {
	Ready               bool
	MissingDependencies []string
}

type GatewayService struct {
	orderClient     OrderClient
	inventoryClient InventoryClient
	paymentClient   PaymentClient

	checkoutTimeout     time.Duration
	readTimeout         time.Duration
	compensationTimeout time.Duration
}

type Option func(*GatewayService)

const (
	defaultCheckoutTimeout     = 5 * time.Second
	defaultReadTimeout         = 2 * time.Second
	defaultCompensationTimeout = 2 * time.Second
)

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

func WithCheckoutTimeout(timeout time.Duration) Option {
	return func(s *GatewayService) {
		s.checkoutTimeout = timeout
	}
}

func WithCompensationTimeout(timeout time.Duration) Option {
	return func(s *GatewayService) {
		s.compensationTimeout = timeout
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(s *GatewayService) {
		s.readTimeout = timeout
	}
}

func NewGatewayService(orderClient OrderClient, opts ...Option) *GatewayService {
	service := &GatewayService{
		orderClient:         orderClient,
		checkoutTimeout:     defaultCheckoutTimeout,
		readTimeout:         defaultReadTimeout,
		compensationTimeout: defaultCompensationTimeout,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}

	return service
}

func (s *GatewayService) ReadinessStatus() ReadinessStatus {
	if s == nil {
		return ReadinessStatus{
			Ready:               false,
			MissingDependencies: []string{"gateway_service"},
		}
	}

	missingDependencies := make([]string, 0, 3)
	if s.orderClient == nil {
		missingDependencies = append(missingDependencies, "order_client")
	}
	if s.inventoryClient == nil {
		missingDependencies = append(missingDependencies, "inventory_client")
	}
	if s.paymentClient == nil {
		missingDependencies = append(missingDependencies, "payment_client")
	}

	return ReadinessStatus{
		Ready:               len(missingDependencies) == 0,
		MissingDependencies: missingDependencies,
	}
}
