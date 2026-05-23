package service

import (
	"context"
	"fmt"
	"strings"

	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
)

func (s *GatewayService) Checkout(ctx context.Context, in *CheckoutInput) (*CheckoutResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: checkout request is nil", ErrInvalidInput)
	}
	if strings.TrimSpace(in.CustomerID) == "" {
		return nil, fmt.Errorf("%w: customer id is required", ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one item is required", ErrInvalidInput)
	}
	if strings.TrimSpace(in.ShippingAddress.Country) == "" ||
		strings.TrimSpace(in.ShippingAddress.City) == "" ||
		strings.TrimSpace(in.ShippingAddress.Street) == "" ||
		strings.TrimSpace(in.ShippingAddress.PostalCode) == "" ||
		strings.TrimSpace(in.ShippingAddress.House) == "" {
		return nil, fmt.Errorf("%w: complete shipping address is required", ErrInvalidInput)
	}
	if strings.TrimSpace(in.Payment.Method) == "" {
		return nil, fmt.Errorf("%w: payment method is required", ErrInvalidInput)
	}
	if s.orderClient == nil {
		return nil, fmt.Errorf("%w: order client is not configured", ErrDownstreamFailed)
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	orderItems, err := mapCheckoutItemsToOrderItems(in.Items)
	if err != nil {
		return nil, err
	}

	paymentMethod, err := normalizePaymentMethod(in.Payment.Method)
	if err != nil {
		return nil, err
	}

	orderResp, err := s.orderClient.CreateOrder(opCtx, &orderclient.CreateOrderRequest{
		CustomerID:      strings.TrimSpace(in.CustomerID),
		Items:           orderItems,
		ShippingAddress: mapAddressToOrderClient(in.ShippingAddress),
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
		Payment: orderclient.PaymentDetails{
			Method:        paymentMethod,
			MethodDetails: strings.TrimSpace(in.Payment.MethodDetails),
		},
	})
	if err != nil {
		return nil, wrapDownstreamError("order create", err)
	}
	if orderResp == nil || orderResp.Order == nil {
		return nil, fmt.Errorf("%w: order response is empty", ErrDownstreamFailed)
	}

	order := orderResp.Order

	return &CheckoutResult{
		OrderID:     order.ID,
		OrderStatus: order.Status,
	}, nil
}
