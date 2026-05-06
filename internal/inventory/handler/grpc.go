package handler

import (
	"context"
	"errors"
	"log/slog"

	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	inventoryv1.UnimplementedInventoryServiceServer
	service *service.InventoryService
	logger  *slog.Logger
}

func NewGRPCHandler(service *service.InventoryService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *GRPCHandler) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	stock, err := h.service.GetStockByProductID(ctx, req.GetProductId())
	if err != nil {
		return nil, mapInventoryError(err)
	}

	return &inventoryv1.GetStockResponse{
		Stock: convertStockToProto(stock),
	}, nil
}

func (h *GRPCHandler) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	stock, err := h.service.ReserveStock(ctx, req.GetProductId(), req.GetQuantity(), req.GetOrderId())
	if err != nil {
		return nil, mapInventoryError(err)
	}

	h.logger.InfoContext(
		ctx,
		"stock reserved",
		slog.String("product_id", req.GetProductId()),
		slog.String("order_id", req.GetOrderId()),
		slog.Int64("quantity", req.GetQuantity()),
	)

	return &inventoryv1.ReserveStockResponse{
		Stock: convertStockToProto(stock),
	}, nil
}

func (h *GRPCHandler) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.ReleaseStockResponse, error) {
	stock, err := h.service.ReleaseStock(ctx, req.GetProductId(), req.GetQuantity(), req.GetOrderId())
	if err != nil {
		return nil, mapInventoryError(err)
	}

	h.logger.InfoContext(
		ctx,
		"stock released",
		slog.String("product_id", req.GetProductId()),
		slog.String("order_id", req.GetOrderId()),
		slog.Int64("quantity", req.GetQuantity()),
	)

	return &inventoryv1.ReleaseStockResponse{
		Stock: convertStockToProto(stock),
	}, nil
}

func convertStockToProto(stock domain.Stock) *inventoryv1.Stock {
	return &inventoryv1.Stock{
		ProductId:         stock.ProductID,
		AvailableQuantity: stock.AvailableQuantity,
		ReservedQuantity:  stock.ReservedQuantity,
		TotalQuantity:     stock.TotalQuantity,
	}
}

func mapInventoryError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidStock):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrStockNotFound), errors.Is(err, domain.ErrReservationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInsufficientStock):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
