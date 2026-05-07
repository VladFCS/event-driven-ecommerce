package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
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
	if s.inventoryClient == nil {
		return nil, fmt.Errorf("%w: inventory client is not configured", ErrDownstreamFailed)
	}
	if s.paymentClient == nil {
		return nil, fmt.Errorf("%w: payment client is not configured", ErrDownstreamFailed)
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

	orderResp, err := s.orderClient.CreateOrder(opCtx, &orderclient.CreateOrderRequest{
		CustomerID:      strings.TrimSpace(in.CustomerID),
		Items:           orderItems,
		ShippingAddress: mapAddressToOrderProto(in.ShippingAddress),
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
	})
	if err != nil {
		return nil, wrapDownstreamError("order create", err)
	}
	if orderResp == nil || orderResp.Order == nil {
		return nil, fmt.Errorf("%w: order response is empty", ErrDownstreamFailed)
	}

	order := orderResp.Order
	reservedItems := make([]*orderv1.OrderItem, 0, len(order.GetItems()))
	for _, item := range order.GetItems() {
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReserveStock(opCtx, &inventoryclient.ReserveStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   order.GetOrderId(),
		})
		if err != nil {
			return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, wrapDownstreamError("inventory reserve stock", err))
		}

		reservedItems = append(reservedItems, item)
	}

	totalAmount := order.GetTotalAmount()
	if totalAmount == nil {
		return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, fmt.Errorf("%w: order total amount is empty", ErrDownstreamFailed))
	}

	paymentMethod, err := parsePaymentMethod(in.Payment.Method)
	if err != nil {
		return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, err)
	}

	paymentCurrency, err := mapOrderCurrencyToPayment(totalAmount.GetCurrency())
	if err != nil {
		return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, err)
	}

	paymentResp, err := s.paymentClient.CreatePayment(opCtx, &paymentclient.CreatePaymentRequest{
		OrderID:    order.GetOrderId(),
		CustomerID: order.GetCustomerId(),
		Amount: &paymentv1.Money{
			Currency:    paymentCurrency,
			AmountCents: totalAmount.GetAmountCents(),
		},
		PaymentMethod:        paymentMethod,
		PaymentMethodDetails: strings.TrimSpace(in.Payment.MethodDetails),
		IdempotencyKey:       strings.TrimSpace(in.IdempotencyKey),
	})
	if err != nil {
		return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, wrapDownstreamError("payment create", err))
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, s.compensateCheckoutFailure(order.GetOrderId(), reservedItems, fmt.Errorf("%w: payment response is empty", ErrDownstreamFailed))
	}

	return &CheckoutResult{
		OrderID:       order.GetOrderId(),
		PaymentID:     paymentResp.Payment.GetPaymentId(),
		OrderStatus:   order.GetStatus().String(),
		PaymentStatus: paymentResp.Payment.GetStatus().String(),
	}, nil
}

func (s *GatewayService) compensateCheckoutFailure(orderID string, reservedItems []*orderv1.OrderItem, originalErr error) error {
	compensationErrs := make([]error, 0, len(reservedItems)+1)
	cleanupCtx := context.Background()
	cancel := func() {}
	if s.compensationTimeout > 0 {
		cleanupCtx, cancel = context.WithTimeout(context.Background(), s.compensationTimeout)
	}
	defer cancel()

	for i := len(reservedItems) - 1; i >= 0; i-- {
		item := reservedItems[i]
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReleaseStock(cleanupCtx, &inventoryclient.ReleaseStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   orderID,
		})
		if err != nil {
			compensationErrs = append(compensationErrs, wrapDownstreamError("inventory release stock", err))
		}
	}

	if strings.TrimSpace(orderID) != "" {
		_, err := s.orderClient.CancelOrder(cleanupCtx, &orderclient.CancelOrderRequest{
			OrderID: orderID,
			Reason:  "checkout failed",
		})
		if err != nil {
			compensationErrs = append(compensationErrs, wrapDownstreamError("order cancel", err))
		}
	}

	if len(compensationErrs) == 0 {
		return originalErr
	}

	compensationErrs = append([]error{originalErr}, compensationErrs...)
	return errors.Join(compensationErrs...)
}
