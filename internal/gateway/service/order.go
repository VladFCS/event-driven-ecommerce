package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *GatewayService) GetOrderByID(ctx context.Context, in *GetOrderByIDInput) (*GetOrderByIDResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: get order request is nil", ErrInvalidInput)
	}
	if strings.TrimSpace(in.OrderID) == "" {
		return nil, fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	resp, err := s.orderClient.GetOrderByID(opCtx, in.OrderID)
	if err != nil {
		return nil, wrapDownstreamError("order get", err)
	}
	if resp == nil || resp.Order == nil {
		return nil, fmt.Errorf("%w: order response is empty", ErrDownstreamFailed)
	}

	order := resp.Order
	result := &GetOrderByIDResult{
		OrderID:         order.ID,
		CustomerID:      order.CustomerID,
		OrderStatus:     order.Status,
		Items:           make([]CheckoutItem, 0, len(order.Items)),
		TotalAmount:     mapOrderMoney(order.TotalAmount),
		ShippingAddress: mapOrderAddress(order.ShippingAddress),
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}

	for _, item := range order.Items {
		result.Items = append(result.Items, CheckoutItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   mapOrderMoney(item.UnitPrice),
			TotalPrice:  mapOrderMoney(item.TotalPrice),
		})
	}

	return result, nil
}
