package service

import (
	"fmt"
	"strings"

	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
)

func mapOrderMoney(money orderclient.Money) Money {
	return Money{
		Currency:    money.Currency,
		AmountCents: money.AmountCents,
	}
}

func mapOrderAddress(address orderclient.Address) Address {
	return Address{
		Country:    address.Country,
		City:       address.City,
		Street:     address.Street,
		PostalCode: address.PostalCode,
		House:      address.House,
		Apartment:  address.Apartment,
	}
}

func mapCheckoutItemsToOrderItems(items []CheckoutItem) ([]orderclient.CreateOrderItem, error) {
	converted := make([]orderclient.CreateOrderItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 || item.UnitPrice.AmountCents <= 0 {
			return nil, fmt.Errorf("%w: invalid checkout item", ErrInvalidInput)
		}

		currency, err := normalizeCurrency(item.UnitPrice.Currency)
		if err != nil {
			return nil, err
		}

		converted = append(converted, orderclient.CreateOrderItem{
			ProductID:   strings.TrimSpace(item.ProductID),
			SKU:         strings.TrimSpace(item.SKU),
			ProductName: strings.TrimSpace(item.ProductName),
			Quantity:    item.Quantity,
			UnitPrice: orderclient.Money{
				Currency:    currency,
				AmountCents: item.UnitPrice.AmountCents,
			},
		})
	}

	return converted, nil
}

func mapAddressToOrderClient(address Address) orderclient.Address {
	return orderclient.Address{
		Country:    strings.TrimSpace(address.Country),
		City:       strings.TrimSpace(address.City),
		Street:     strings.TrimSpace(address.Street),
		PostalCode: strings.TrimSpace(address.PostalCode),
		House:      strings.TrimSpace(address.House),
		Apartment:  strings.TrimSpace(address.Apartment),
	}
}

func normalizeCurrency(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return "CURRENCY_USD", nil
	case "EUR", "CURRENCY_EUR":
		return "CURRENCY_EUR", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedCurrency, value)
	}
}

func normalizePaymentMethod(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CARD", "PAYMENT_METHOD_TYPE_CARD":
		return "PAYMENT_METHOD_TYPE_CARD", nil
	case "CASH", "PAYMENT_METHOD_TYPE_CASH":
		return "PAYMENT_METHOD_TYPE_CASH", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedPaymentMethod, value)
	}
}
