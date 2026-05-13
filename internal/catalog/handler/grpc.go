package handler

import (
	"context"
	"errors"
	"log/slog"
	"math"

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

func (h *GRPCHandler) GetProductByID(ctx context.Context, req *catalogv1.GetProductByIDRequest) (*catalogv1.GetProductByIDResponse, error) {
	product, err := h.service.GetProductByID(ctx, req.GetProductId())
	if err != nil {
		return nil, mapCatalogError(err)
	}

	return &catalogv1.GetProductByIDResponse{
		Product: convertProductToProto(product),
	}, nil
}

func (h *GRPCHandler) ListProducts(ctx context.Context, req *catalogv1.ListProductsRequest) (*catalogv1.ListProductsResponse, error) {
	products, total, err := h.service.ListProducts(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, mapCatalogError(err)
	}

	page := req.GetPage()
	if page <= 0 {
		page = 1
	}

	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		if total > math.MaxInt32 {
			pageSize = math.MaxInt32
		} else {
			pageSize = int32(total)
		}
	}

	protoProducts := make([]*catalogv1.Product, 0, len(products))
	for _, product := range products {
		protoProducts = append(protoProducts, convertProductToProto(product))
	}

	return &catalogv1.ListProductsResponse{
		Products: protoProducts,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
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

	h.logger.InfoContext(ctx, "product created", slog.String("product_id", product.ID))

	return &catalogv1.CreateProductResponse{
		Product: convertProductToProto(product),
	}, nil
}

func (h *GRPCHandler) DeleteProduct(ctx context.Context, req *catalogv1.DeleteProductRequest) (*catalogv1.DeleteProductResponse, error) {
	if err := h.service.DeleteProduct(ctx, req.GetProductId()); err != nil {
		return nil, mapCatalogError(err)
	}

	h.logger.InfoContext(ctx, "product deleted", slog.String("product_id", req.GetProductId()))

	return &catalogv1.DeleteProductResponse{}, nil
}

func convertProductToProto(product domain.Product) *catalogv1.Product {
	return &catalogv1.Product{
		ProductId:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}
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
