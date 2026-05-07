package service

import (
	"fmt"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
)

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

func mapCheckoutItemsToOrderItems(items []CheckoutItem) ([]*orderv1.CreateOrderItem, error) {
	converted := make([]*orderv1.CreateOrderItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 || item.UnitPrice.AmountCents <= 0 {
			return nil, fmt.Errorf("%w: invalid checkout item", ErrInvalidInput)
		}

		currency, err := parseOrderCurrency(item.UnitPrice.Currency)
		if err != nil {
			return nil, err
		}

		converted = append(converted, &orderv1.CreateOrderItem{
			ProductId:   strings.TrimSpace(item.ProductID),
			Sku:         strings.TrimSpace(item.SKU),
			ProductName: strings.TrimSpace(item.ProductName),
			Quantity:    item.Quantity,
			UnitPrice: &orderv1.Money{
				Currency:    currency,
				AmountCents: item.UnitPrice.AmountCents,
			},
		})
	}

	return converted, nil
}

func mapAddressToOrderProto(address Address) *orderv1.Address {
	return &orderv1.Address{
		Country:    strings.TrimSpace(address.Country),
		City:       strings.TrimSpace(address.City),
		Street:     strings.TrimSpace(address.Street),
		PostalCode: strings.TrimSpace(address.PostalCode),
		House:      strings.TrimSpace(address.House),
		Apartment:  strings.TrimSpace(address.Apartment),
	}
}

func parseOrderCurrency(value string) (orderv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return orderv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return orderv1.Currency_CURRENCY_EUR, nil
	default:
		return orderv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedCurrency, value)
	}
}

func mapOrderCurrencyToPayment(currency orderv1.Currency) (paymentv1.Currency, error) {
	switch currency {
	case orderv1.Currency_CURRENCY_USD:
		return paymentv1.Currency_CURRENCY_USD, nil
	case orderv1.Currency_CURRENCY_EUR:
		return paymentv1.Currency_CURRENCY_EUR, nil
	default:
		return paymentv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency.String())
	}
}

func parsePaymentMethod(value string) (paymentv1.PaymentMethodType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CARD", "PAYMENT_METHOD_TYPE_CARD":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD, nil
	case "CASH", "PAYMENT_METHOD_TYPE_CASH":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH, nil
	default:
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedPaymentMethod, value)
	}
}
