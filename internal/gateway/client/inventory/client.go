package inventory

import (
	"context"
	"errors"

	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

var (
	ErrReserveStockRequestNil = errors.New("reserve stock request is nil")
	ErrReleaseStockRequestNil = errors.New("release stock request is nil")
)

type Stock struct {
	ProductID         string
	AvailableQuantity int64
	ReservedQuantity  int64
	TotalQuantity     int64
}

type ReserveStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReserveStockResponse struct {
	Stock *Stock
}

type ReleaseStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReleaseStockResponse struct {
	Stock *Stock
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
		return nil, ErrReserveStockRequestNil
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
		Stock: mapProtoStock(grpcResp.GetStock()),
	}, nil
}

func (c *GRPCClient) ReleaseStock(ctx context.Context, req *ReleaseStockRequest) (*ReleaseStockResponse, error) {
	if req == nil {
		return nil, ErrReleaseStockRequestNil
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
		Stock: mapProtoStock(grpcResp.GetStock()),
	}, nil
}

func mapProtoStock(stock *inventoryv1.Stock) *Stock {
	if stock == nil {
		return nil
	}

	return &Stock{
		ProductID:         stock.GetProductId(),
		AvailableQuantity: stock.GetAvailableQuantity(),
		ReservedQuantity:  stock.GetReservedQuantity(),
		TotalQuantity:     stock.GetTotalQuantity(),
	}
}
