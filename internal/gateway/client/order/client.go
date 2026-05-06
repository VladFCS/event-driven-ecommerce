package order

import (
	"context"
	"errors"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"google.golang.org/grpc"
)

type Client interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error)
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

	grpcResp, err := c.grpcClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
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