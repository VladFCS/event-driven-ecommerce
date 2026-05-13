package service

import (
	"context"
	"fmt"
	"strings"

	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
)

func (s *GatewayService) GetStockByProductID(ctx context.Context, in *GetStockByProductIDInput) (*GetStockByProductIDResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: get stock by product id request is nil", ErrInvalidInput)
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

	resp, err := s.inventoryClient.GetStockByProductID(opCtx, &inventoryclient.GetStockByProductIDRequest{
		ProductID: productID,
	})
	if err != nil {
		return nil, wrapDownstreamError("inventory stock get", err)
	}
	if resp == nil || resp.Stock == nil {
		return nil, fmt.Errorf("%w: stock response is empty", ErrDownstreamFailed)
	}

	return &GetStockByProductIDResult{
		ProductID: resp.Stock.ProductID,
		Available: resp.Stock.AvailableQuantity,
		Reserved:  resp.Stock.ReservedQuantity,
	}, nil
}
