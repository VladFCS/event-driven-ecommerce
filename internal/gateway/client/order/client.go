package order

import (
	"context"
	"errors"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

type Client interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error)
	ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error)
}

type CreateOrderRequest struct {
	CustomerID      string
	Items           []*orderv1.CreateOrderItem
	ShippingAddress *orderv1.Address
	IdempotencyKey  string
}

type CreateOrderResponse struct {
	Order *orderv1.Order
}

type CancelOrderRequest struct {
	OrderID string
	Reason  string
}

type CancelOrderResponse struct {
	Order *orderv1.Order
}

type GetOrderResponse struct {
	Order *orderv1.Order
}

type ListOrdersByCustomerRequest struct {
	CustomerID string
	Page       int
	PageSize   int
}

type ListOrdersByCustomerResponse struct {
	Orders   []*orderv1.Order
	Page     int
	PageSize int
	Total    int64
}

type GRPCClient struct {
	grpcClient orderv1.OrderServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: orderv1.NewOrderServiceClient(conn),
	}
}

func (c *GRPCClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if req == nil {
		return nil, errors.New("create order request is nil")
	}

	grpcResp, err := c.grpcClient.CreateOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CreateOrderRequest{
		CustomerId:      req.CustomerID,
		Items:           req.Items,
		ShippingAddress: req.ShippingAddress,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	return &CreateOrderResponse{
		Order: grpcResp.GetOrder(),
	}, nil
}

func (c *GRPCClient) GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("order id is required")
	}

	grpcResp, err := c.grpcClient.GetOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.GetOrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, err
	}

	return &GetOrderResponse{
		Order: grpcResp.GetOrder(),
	}, nil
}

func (c *GRPCClient) ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error) {
	if req == nil {
		return nil, errors.New("list orders by customer request is nil")
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return nil, errors.New("customer id is required")
	}

	grpcResp, err := c.grpcClient.ListOrdersByCustomer(requestid.WithOutgoingMetadata(ctx), &orderv1.ListOrdersByCustomerRequest{
		CustomerId: req.CustomerID,
		Page:       int32(req.Page),
		PageSize:   int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	return &ListOrdersByCustomerResponse{
		Orders:   grpcResp.GetOrders(),
		Page:     int(grpcResp.GetPage()),
		PageSize: int(grpcResp.GetPageSize()),
		Total:    grpcResp.GetTotal(),
	}, nil
}

func (c *GRPCClient) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if req == nil {
		return nil, errors.New("cancel order request is nil")
	}

	grpcResp, err := c.grpcClient.CancelOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CancelOrderRequest{
		OrderId: req.OrderID,
		Reason:  req.Reason,
	})
	if err != nil {
		return nil, err
	}

	return &CancelOrderResponse{
		Order: grpcResp.GetOrder(),
	}, nil
}
