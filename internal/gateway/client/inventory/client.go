package inventory

import (
	"context"
	"errors"

	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/gateway/requestid"
	"google.golang.org/grpc"
)

type ReserveStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReserveStockResponse struct {
	Stock *inventoryv1.Stock
}

type ReleaseStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReleaseStockResponse struct {
	Stock *inventoryv1.Stock
}

type GRPCClient struct {
	grpcClient inventoryv1.InventoryServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: inventoryv1.NewInventoryServiceClient(conn),
	}
}

func (c *GRPCClient) ReserveStock(ctx context.Context, req *ReserveStockRequest) (*ReserveStockResponse, error) {
	if req == nil {
		return nil, errors.New("reserve stock request is nil")
	}

	grpcResp, err := c.grpcClient.ReserveStock(requestid.WithOutgoingMetadata(ctx), &inventoryv1.ReserveStockRequest{
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
		OrderId:   req.OrderID,
	})
	if err != nil {
		return nil, err
	}

	return &ReserveStockResponse{
		Stock: grpcResp.GetStock(),
	}, nil
}

func (c *GRPCClient) ReleaseStock(ctx context.Context, req *ReleaseStockRequest) (*ReleaseStockResponse, error) {
	if req == nil {
		return nil, errors.New("release stock request is nil")
	}

	grpcResp, err := c.grpcClient.ReleaseStock(requestid.WithOutgoingMetadata(ctx), &inventoryv1.ReleaseStockRequest{
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
		OrderId:   req.OrderID,
	})
	if err != nil {
		return nil, err
	}

	return &ReleaseStockResponse{
		Stock: grpcResp.GetStock(),
	}, nil
}
