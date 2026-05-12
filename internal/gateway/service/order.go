package service

import (
	"context"
	"fmt"
	"strings"

	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
)

func (s *GatewayService) ListOrdersByCustomer(ctx context.Context, in *ListOrdersByCustomerInput) (*ListOrdersByCustomerResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: list orders request is nil", ErrInvalidInput)
	}
	customerID := strings.TrimSpace(in.CustomerID)
	if customerID == "" {
		return nil, fmt.Errorf("%w: customer id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	resp, err := s.orderClient.ListOrdersByCustomer(opCtx, &orderclient.ListOrdersByCustomerRequest{
		CustomerID: customerID,
		Page:       in.Page,
		PageSize:   in.PageSize,
	})
	if err != nil {
		return nil, wrapDownstreamError("order list by customer", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: list orders response is empty", ErrDownstreamFailed)
	}

	result := &ListOrdersByCustomerResult{
		Orders:   make([]GetOrderByIDResult, 0, len(resp.Orders)),
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}

	for _, order := range resp.Orders {
		result.Orders = append(result.Orders, mapClientOrderToResult(order))
	}

	return result, nil
}

func (s *GatewayService) CancelOrder(ctx context.Context, in *CancelOrderInput) (*CancelOrderResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: cancel order request is nil", ErrInvalidInput)
	}

	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}

	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: cancel reason is required", ErrInvalidInput)
	}

	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	if idempotencyKey == "" {
		return s.performCancelOrder(ctx, orderID, reason)
	}

	record, existing, err := s.beginCancelIdempotency(idempotencyKey, cancelRequestFingerprint(orderID, reason))
	if err != nil {
		return nil, err
	}
	if existing {
		<-record.done
		if record.result != nil {
			resultCopy := *record.result
			return &resultCopy, nil
		}

		return nil, record.err
	}

	result, err := s.performCancelOrder(ctx, orderID, reason)
	s.finishCancelIdempotency(idempotencyKey, record, result, err)

	return result, err
}

func (s *GatewayService) performCancelOrder(ctx context.Context, orderID, reason string) (*CancelOrderResult, error) {
	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	resp, err := s.orderClient.CancelOrder(opCtx, &orderclient.CancelOrderRequest{
		OrderID: orderID,
		Reason:  reason,
	})
	if err != nil {
		return nil, wrapDownstreamError("order cancel", err)
	}
	if resp == nil || resp.Order == nil {
		return nil, fmt.Errorf("%w: cancel order response is empty", ErrDownstreamFailed)
	}

	return &CancelOrderResult{
		OrderID:     resp.Order.ID,
		CustomerID:  resp.Order.CustomerID,
		OrderStatus: resp.Order.Status,
		UpdatedAt:   resp.Order.UpdatedAt,
	}, nil
}

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

	result := mapClientOrderToResult(*resp.Order)
	return &result, nil
}

func mapClientOrderToResult(order orderclient.Order) GetOrderByIDResult {
	result := GetOrderByIDResult{
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

	return result
}
