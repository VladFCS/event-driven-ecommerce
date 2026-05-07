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

	resp, err := s.orderClient.GetOrder(ctx, in.OrderID)
	if err != nil {
		return nil, wrapDownstreamError("order get", err)
	}
	if resp == nil || resp.Order == nil {
		return nil, fmt.Errorf("%w: order response is empty", ErrDownstreamFailed)
	}

	order := resp.Order
	result := &GetOrderByIDResult{
		OrderID:         order.GetOrderId(),
		CustomerID:      order.GetCustomerId(),
		OrderStatus:     order.GetStatus().String(),
		Items:           make([]CheckoutItem, 0, len(order.GetItems())),
		TotalAmount:     mapProtoMoney(order.GetTotalAmount()),
		ShippingAddress: mapProtoAddress(order.GetShippingAddress()),
		CreatedAt:       order.GetCreatedAt(),
		UpdatedAt:       order.GetUpdatedAt(),
	}

	for _, item := range order.GetItems() {
		result.Items = append(result.Items, CheckoutItem{
			ProductID:   item.GetProductId(),
			SKU:         item.GetSku(),
			ProductName: item.GetProductName(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   mapProtoMoney(item.GetUnitPrice()),
			TotalPrice:  mapProtoMoney(item.GetTotalPrice()),
		})
	}

	return result, nil
}
