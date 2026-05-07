package service

import (
	"context"
	"errors"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
)

func (s *GatewayService) Checkout(ctx context.Context, in *CheckoutInput) (*CheckoutResult, error) {
	if in == nil {
		return nil, errors.New("checkout request is nil")
	}
	if strings.TrimSpace(in.CustomerID) == "" {
		return nil, errors.New("customer id is required")
	}
	if len(in.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}
	if strings.TrimSpace(in.ShippingAddress.Country) == "" ||
		strings.TrimSpace(in.ShippingAddress.City) == "" ||
		strings.TrimSpace(in.ShippingAddress.Street) == "" ||
		strings.TrimSpace(in.ShippingAddress.PostalCode) == "" ||
		strings.TrimSpace(in.ShippingAddress.House) == "" {
		return nil, errors.New("complete shipping address is required")
	}
	if strings.TrimSpace(in.Payment.Method) == "" {
		return nil, errors.New("payment method is required")
	}
	if s.orderClient == nil {
		return nil, errors.New("order client is not configured")
	}
	if s.inventoryClient == nil {
		return nil, errors.New("inventory client is not configured")
	}
	if s.paymentClient == nil {
		return nil, errors.New("payment client is not configured")
	}

	orderItems, err := mapCheckoutItemsToOrderItems(in.Items)
	if err != nil {
		return nil, err
	}

	orderResp, err := s.orderClient.CreateOrder(ctx, &orderclient.CreateOrderRequest{
		CustomerID:      strings.TrimSpace(in.CustomerID),
		Items:           orderItems,
		ShippingAddress: mapAddressToOrderProto(in.ShippingAddress),
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
	})
	if err != nil {
		return nil, err
	}
	if orderResp == nil || orderResp.Order == nil {
		return nil, errors.New("order response is empty")
	}

	order := orderResp.Order
	reservedItems := make([]*orderv1.OrderItem, 0, len(order.GetItems()))
	for _, item := range order.GetItems() {
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReserveStock(ctx, &inventoryclient.ReserveStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   order.GetOrderId(),
		})
		if err != nil {
			return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
		}

		reservedItems = append(reservedItems, item)
	}

	totalAmount := order.GetTotalAmount()
	if totalAmount == nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, errors.New("order total amount is empty"))
	}

	paymentMethod, err := parsePaymentMethod(in.Payment.Method)
	if err != nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}

	paymentCurrency, err := mapOrderCurrencyToPayment(totalAmount.GetCurrency())
	if err != nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}

	paymentResp, err := s.paymentClient.CreatePayment(ctx, &paymentclient.CreatePaymentRequest{
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
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, errors.New("payment response is empty"))
	}

	return &CheckoutResult{
		OrderID:       order.GetOrderId(),
		PaymentID:     paymentResp.Payment.GetPaymentId(),
		OrderStatus:   order.GetStatus().String(),
		PaymentStatus: paymentResp.Payment.GetStatus().String(),
	}, nil
}

func (s *GatewayService) compensateCheckoutFailure(ctx context.Context, orderID string, reservedItems []*orderv1.OrderItem, originalErr error) error {
	compensationErrs := make([]error, 0, len(reservedItems)+1)

	for i := len(reservedItems) - 1; i >= 0; i-- {
		item := reservedItems[i]
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReleaseStock(ctx, &inventoryclient.ReleaseStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   orderID,
		})
		if err != nil {
			compensationErrs = append(compensationErrs, err)
		}
	}

	if strings.TrimSpace(orderID) != "" {
		_, err := s.orderClient.CancelOrder(ctx, &orderclient.CancelOrderRequest{
			OrderID: orderID,
			Reason:  "checkout failed",
		})
		if err != nil {
			compensationErrs = append(compensationErrs, err)
		}
	}

	if len(compensationErrs) == 0 {
		return originalErr
	}

	compensationErrs = append([]error{originalErr}, compensationErrs...)
	return errors.Join(compensationErrs...)
}
