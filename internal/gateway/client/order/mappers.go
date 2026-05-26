package order

import (
	"fmt"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
)

func mapCreateOrderItemsToProto(items []CreateOrderItem) ([]*orderv1.CreateOrderItem, error) {
	converted := make([]*orderv1.CreateOrderItem, 0, len(items))
	for _, item := range items {
		unitPrice, err := mapMoneyToProto(item.UnitPrice)
		if err != nil {
			return nil, err
		}

		converted = append(converted, &orderv1.CreateOrderItem{
			ProductId:   strings.TrimSpace(item.ProductID),
			Sku:         strings.TrimSpace(item.SKU),
			ProductName: strings.TrimSpace(item.ProductName),
			Quantity:    item.Quantity,
			UnitPrice:   unitPrice,
		})
	}

	return converted, nil
}

func mapAddressToProto(address Address) *orderv1.Address {
	return &orderv1.Address{
		Country:    strings.TrimSpace(address.Country),
		City:       strings.TrimSpace(address.City),
		Street:     strings.TrimSpace(address.Street),
		PostalCode: strings.TrimSpace(address.PostalCode),
		House:      strings.TrimSpace(address.House),
		Apartment:  strings.TrimSpace(address.Apartment),
	}
}

func mapPaymentDetailsToProto(payment PaymentDetails) (orderv1.PaymentMethodType, string, error) {
	method, err := parsePaymentMethod(payment.Method)
	if err != nil {
		return orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, "", err
	}

	return method, strings.TrimSpace(payment.MethodDetails), nil
}

func mapMoneyToProto(money Money) (*orderv1.Money, error) {
	currency, err := parseCurrency(money.Currency)
	if err != nil {
		return nil, err
	}

	return &orderv1.Money{
		Currency:    currency,
		AmountCents: money.AmountCents,
	}, nil
}

func mapProtoOrder(order *orderv1.Order) *Order {
	if order == nil {
		return nil
	}

	return &Order{
		ID:              order.GetOrderId(),
		CustomerID:      order.GetCustomerId(),
		Items:           mapProtoOrderItems(order.GetItems()),
		TotalAmount:     mapProtoMoney(order.GetTotalAmount()),
		Status:          order.GetStatus().String(),
		ShippingAddress: mapProtoAddress(order.GetShippingAddress()),
		Payment:         mapProtoPaymentDetails(order.GetPayment()),
		CancelReason:    order.GetCancelReason(),
		CreatedAt:       order.GetCreatedAt(),
		UpdatedAt:       order.GetUpdatedAt(),
	}
}

func mapProtoOrders(orders []*orderv1.Order) []Order {
	converted := make([]Order, 0, len(orders))
	for _, order := range orders {
		if mapped := mapProtoOrder(order); mapped != nil {
			converted = append(converted, *mapped)
		}
	}

	return converted
}

func mapProtoOrderItems(items []*orderv1.OrderItem) []OrderItem {
	converted := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		converted = append(converted, OrderItem{
			ProductID:   item.GetProductId(),
			SKU:         item.GetSku(),
			ProductName: item.GetProductName(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   mapProtoMoney(item.GetUnitPrice()),
			TotalPrice:  mapProtoMoney(item.GetTotalPrice()),
		})
	}

	return converted
}

func mapProtoMoney(money *orderv1.Money) Money {
	if money == nil {
		return Money{}
	}

	return Money{
		Currency:    money.GetCurrency().String(),
		AmountCents: money.GetAmountCents(),
	}
}

func mapProtoAddress(address *orderv1.Address) Address {
	if address == nil {
		return Address{}
	}

	return Address{
		Country:    address.GetCountry(),
		City:       address.GetCity(),
		Street:     address.GetStreet(),
		PostalCode: address.GetPostalCode(),
		House:      address.GetHouse(),
		Apartment:  address.GetApartment(),
	}
}

func mapProtoPaymentDetails(details *orderv1.PaymentDetails) PaymentDetails {
	if details == nil {
		return PaymentDetails{}
	}

	return PaymentDetails{
		Method:        details.GetMethod().String(),
		MethodDetails: details.GetMethodDetails(),
	}
}

func parseCurrency(value string) (orderv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return orderv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return orderv1.Currency_CURRENCY_EUR, nil
	default:
		return orderv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedOrderCurrency, value)
	}
}

func parsePaymentMethod(value string) (orderv1.PaymentMethodType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CARD", "PAYMENT_METHOD_TYPE_CARD":
		return orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD, nil
	case "CASH", "PAYMENT_METHOD_TYPE_CASH":
		return orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH, nil
	default:
		return orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedPaymentMethod, value)
	}
}
