package inventory

import (
	"context"
	"strings"

	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

type GRPCClient struct {
	grpcClient inventoryv1.InventoryServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: inventoryv1.NewInventoryServiceClient(conn),
	}
}

func (c *GRPCClient) GetStockByProductID(ctx context.Context, req *GetStockByProductIDRequest) (*GetStockByProductIDResponse, error) {
	if req == nil {
		return nil, ErrGetStockRequestNil
	}

	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		return nil, ErrProductIDRequired
	}

	grpcResp, err := c.grpcClient.GetStockByProductID(requestid.WithOutgoingMetadata(ctx), &inventoryv1.GetStockByProductIDRequest{
		ProductId: productID,
	})
	if err != nil {
		return nil, err
	}

	return &GetStockByProductIDResponse{
		Stock: mapProtoStock(grpcResp.GetStock()),
	}, nil
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
