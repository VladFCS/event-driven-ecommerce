package handler

import (
	"context"
	"errors"
	"log/slog"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	catalogv1.UnimplementedCatalogServiceServer
	service *service.CatalogService
	logger  *slog.Logger
}

func NewGRPCHandler(service *service.CatalogService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *GRPCHandler) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	product, err := h.service.GetProductByID(ctx, req.GetProductId())
	if err != nil {
		return nil, mapCatalogError(err)
	}

	res := &catalogv1.Product{
		ProductId:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}

	return &catalogv1.GetProductResponse{
		Product: res,
	}, nil
}

func (h *GRPCHandler) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	if req.GetProduct() == nil {
		return nil, status.Error(codes.InvalidArgument, "product is required")
	}

	product, err := h.service.CreateProduct(ctx, domain.Product{
		ID:          req.GetProduct().GetProductId(),
		Name:        req.GetProduct().GetName(),
		Description: req.GetProduct().GetDescription(),
		PriceCents:  req.GetProduct().GetPriceCents(),
		Currency:    req.GetProduct().GetCurrency(),
	})
	if err != nil {
		return nil, mapCatalogError(err)
	}

	res := &catalogv1.Product{
		ProductId:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}

	h.logger.InfoContext(ctx, "product created", slog.String("product_id", product.ID))

	return &catalogv1.CreateProductResponse{
		Product: res,
	}, nil
}

func mapCatalogError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidProduct):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrProductNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
