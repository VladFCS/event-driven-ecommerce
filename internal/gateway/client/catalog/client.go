package catalog

import (
	"context"
	"strings"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

type GRPCClient struct {
	grpcClient catalogv1.CatalogServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: catalogv1.NewCatalogServiceClient(conn),
	}
}

func (c *GRPCClient) CreateProduct(ctx context.Context, req *CreateProductRequest) (*CreateProductResponse, error) {
	if req == nil {
		return nil, ErrCreateProductRequestNil
	}

	currency, err := parseCurrency(req.Currency)
	if err != nil {
		return nil, err
	}

	grpcResp, err := c.grpcClient.CreateProduct(requestid.WithOutgoingMetadata(ctx), &catalogv1.CreateProductRequest{
		Product: &catalogv1.Product{
			ProductId:   strings.TrimSpace(req.ProductID),
			Name:        strings.TrimSpace(req.Name),
			Description: strings.TrimSpace(req.Description),
			PriceCents:  req.PriceCents,
			Currency:    currency,
		},
	})
	if err != nil {
		return nil, err
	}

	return &CreateProductResponse{
		Product: mapProtoProduct(grpcResp.GetProduct()),
	}, nil
}

func (c *GRPCClient) UpdateProduct(ctx context.Context, req *UpdateProductRequest) (*UpdateProductResponse, error) {
	if req == nil {
		return nil, ErrUpdateProductRequestNil
	}

	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		return nil, ErrProductIDRequired
	}

	grpcReq := &catalogv1.UpdateProductRequest{
		ProductId: productID,
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		grpcReq.Name = &name
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		grpcReq.Description = &description
	}
	if req.PriceCents != nil {
		priceCents := *req.PriceCents
		grpcReq.PriceCents = &priceCents
	}
	if req.Currency != nil {
		currency, err := parseCurrency(*req.Currency)
		if err != nil {
			return nil, err
		}
		grpcReq.Currency = &currency
	}

	grpcResp, err := c.grpcClient.UpdateProduct(requestid.WithOutgoingMetadata(ctx), grpcReq)
	if err != nil {
		return nil, err
	}

	return &UpdateProductResponse{
		Product: mapProtoProduct(grpcResp.GetProduct()),
	}, nil
}

func (c *GRPCClient) GetProductByID(ctx context.Context, productID string) (*GetProductByIDResponse, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, ErrProductIDRequired
	}

	grpcResp, err := c.grpcClient.GetProductByID(requestid.WithOutgoingMetadata(ctx), &catalogv1.GetProductByIDRequest{
		ProductId: productID,
	})
	if err != nil {
		return nil, err
	}

	return &GetProductByIDResponse{
		Product: mapProtoProduct(grpcResp.GetProduct()),
	}, nil
}

func (c *GRPCClient) ListProducts(ctx context.Context, req *ListProductsRequest) (*ListProductsResponse, error) {
	if req == nil {
		return nil, ErrListProductsRequestNil
	}

	grpcResp, err := c.grpcClient.ListProducts(requestid.WithOutgoingMetadata(ctx), &catalogv1.ListProductsRequest{
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	return &ListProductsResponse{
		Products: mapProtoProducts(grpcResp.GetProducts()),
		Page:     int(grpcResp.GetPage()),
		PageSize: int(grpcResp.GetPageSize()),
		Total:    grpcResp.GetTotal(),
	}, nil
}

func (c *GRPCClient) DeleteProduct(ctx context.Context, productID string) error {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ErrProductIDRequired
	}

	_, err := c.grpcClient.DeleteProduct(requestid.WithOutgoingMetadata(ctx), &catalogv1.DeleteProductRequest{
		ProductId: productID,
	})
	if err != nil {
		return err
	}

	return nil
}
