package service

import (
	"context"
	"fmt"
	"strings"

	catalogclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/catalog"
)

func (s *GatewayService) CreateProduct(ctx context.Context, in *CreateProductInput) (*CreateProductResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: create product request is nil", ErrInvalidInput)
	}

	productID := strings.TrimSpace(in.ProductID)
	if productID == "" {
		return nil, fmt.Errorf("%w: product id is required", ErrInvalidInput)
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: product name is required", ErrInvalidInput)
	}
	if in.PriceCents <= 0 {
		return nil, fmt.Errorf("%w: price cents must be greater than 0", ErrInvalidInput)
	}

	currency, err := normalizeCurrency(in.Currency)
	if err != nil {
		return nil, err
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	resp, err := s.catalogClient.CreateProduct(opCtx, &catalogclient.CreateProductRequest{
		ProductID:   productID,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		PriceCents:  in.PriceCents,
		Currency:    currency,
	})
	if err != nil {
		return nil, wrapDownstreamError("catalog product create", err)
	}
	if resp == nil || resp.Product == nil {
		return nil, fmt.Errorf("%w: create product response is empty", ErrDownstreamFailed)
	}

	product := resp.Product
	return &CreateProductResult{
		ProductID:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}, nil
}

func (s *GatewayService) UpdateProduct(ctx context.Context, in *UpdateProductInput) (*UpdateProductResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: update product request is nil", ErrInvalidInput)
	}

	productID := strings.TrimSpace(in.ProductID)
	if productID == "" {
		return nil, fmt.Errorf("%w: product id is required", ErrInvalidInput)
	}

	var (
		name        *string
		description *string
		priceCents  *int64
		currency    *string
	)

	if in.Name != nil {
		trimmedName := strings.TrimSpace(*in.Name)
		if trimmedName == "" {
			return nil, fmt.Errorf("%w: product name is required", ErrInvalidInput)
		}
		name = &trimmedName
	}

	if in.Description != nil {
		trimmedDescription := strings.TrimSpace(*in.Description)
		description = &trimmedDescription
	}

	if in.PriceCents != nil {
		if *in.PriceCents <= 0 {
			return nil, fmt.Errorf("%w: price cents must be greater than 0", ErrInvalidInput)
		}
		value := *in.PriceCents
		priceCents = &value
	}

	if in.Currency != nil {
		normalizedCurrency, err := normalizeCurrency(*in.Currency)
		if err != nil {
			return nil, err
		}
		currency = &normalizedCurrency
	}

	if name == nil && description == nil && priceCents == nil && currency == nil {
		return nil, fmt.Errorf("%w: at least one field must be provided", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	resp, err := s.catalogClient.UpdateProduct(opCtx, &catalogclient.UpdateProductRequest{
		ProductID:   productID,
		Name:        name,
		Description: description,
		PriceCents:  priceCents,
		Currency:    currency,
	})
	if err != nil {
		return nil, wrapDownstreamError("catalog product update", err)
	}
	if resp == nil || resp.Product == nil {
		return nil, fmt.Errorf("%w: update product response is empty", ErrDownstreamFailed)
	}

	product := resp.Product
	return &UpdateProductResult{
		ProductID:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}, nil
}

func (s *GatewayService) GetProductByID(ctx context.Context, in *GetProductByIDInput) (*GetProductByIDResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: get product by id request is nil", ErrInvalidInput)
	}

	productID := strings.TrimSpace(in.ProductID)
	if productID == "" {
		return nil, fmt.Errorf("%w: product id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	resp, err := s.catalogClient.GetProductByID(opCtx, productID)
	if err != nil {
		return nil, wrapDownstreamError("catalog product get", err)
	}
	if resp == nil || resp.Product == nil {
		return nil, fmt.Errorf("%w: product response is empty", ErrDownstreamFailed)
	}

	product := resp.Product
	return &GetProductByIDResult{
		ProductID:   product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}, nil
}

func (s *GatewayService) DeleteProduct(ctx context.Context, in *DeleteProductInput) error {
	if in == nil {
		return fmt.Errorf("%w: delete product request is nil", ErrInvalidInput)
	}

	productID := strings.TrimSpace(in.ProductID)
	if productID == "" {
		return fmt.Errorf("%w: product id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	if err := s.catalogClient.DeleteProduct(opCtx, productID); err != nil {
		return wrapDownstreamError("catalog product delete", err)
	}

	return nil
}

func (s *GatewayService) ListProducts(ctx context.Context, in *ListProductsInput) (*ListProductsResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: list products request is nil", ErrInvalidInput)
	}

	if in.Page < 0 || in.PageSize < 0 {
		return nil, fmt.Errorf("%w: page and page size must be non-negative", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	resp, err := s.catalogClient.ListProducts(opCtx, &catalogclient.ListProductsRequest{
		Page:     in.Page,
		PageSize: in.PageSize,
	})
	if err != nil {
		return nil, wrapDownstreamError("catalog products list", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: list products response is empty", ErrDownstreamFailed)
	}

	result := &ListProductsResult{
		Products: make([]ProductResult, 0, len(resp.Products)),
		Page:     resp.Page,
		PageSize: resp.PageSize,
		Total:    resp.Total,
	}

	for _, product := range resp.Products {
		result.Products = append(result.Products, ProductResult{
			ProductID:   product.ID,
			Name:        product.Name,
			Description: product.Description,
			PriceCents:  product.PriceCents,
			Currency:    product.Currency,
		})
	}

	return result, nil
}
