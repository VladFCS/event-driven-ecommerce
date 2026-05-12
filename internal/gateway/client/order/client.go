package order

import (
	"context"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

type Client interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)
	GetOrderByID(ctx context.Context, orderID string) (*GetOrderByIDResponse, error)
	ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error)
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
		return nil, ErrCreateOrderRequestNil
	}

	items, err := mapCreateOrderItemsToProto(req.Items)
	if err != nil {
		return nil, err
	}

	grpcResp, err := c.grpcClient.CreateOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CreateOrderRequest{
		CustomerId:      req.CustomerID,
		Items:           items,
		ShippingAddress: mapAddressToProto(req.ShippingAddress),
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	return &CreateOrderResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}

func (c *GRPCClient) GetOrderByID(ctx context.Context, orderID string) (*GetOrderByIDResponse, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, ErrOrderIDRequired
	}

	grpcResp, err := c.grpcClient.GetOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.GetOrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, err
	}

	return &GetOrderByIDResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}

func (c *GRPCClient) ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error) {
	if req == nil {
		return nil, ErrListOrdersRequestNil
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return nil, ErrCustomerIDRequired
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
		Orders:   mapProtoOrders(grpcResp.GetOrders()),
		Page:     int(grpcResp.GetPage()),
		PageSize: int(grpcResp.GetPageSize()),
		Total:    grpcResp.GetTotal(),
	}, nil
}

func (c *GRPCClient) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if req == nil {
		return nil, ErrCancelOrderRequestNil
	}

	grpcResp, err := c.grpcClient.CancelOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CancelOrderRequest{
		OrderId: req.OrderID,
		Reason:  req.Reason,
	})
	if err != nil {
		return nil, err
	}

	return &CancelOrderResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}
