package service

import (
	"context"
	"fmt"
	"strings"
)

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
