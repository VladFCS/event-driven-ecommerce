package catalog

import (
	"fmt"
	"strings"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
)

func mapProtoProduct(product *catalogv1.Product) *Product {
	if product == nil {
		return nil
	}

	return &Product{
		ID:          product.GetProductId(),
		Name:        product.GetName(),
		Description: product.GetDescription(),
		PriceCents:  product.GetPriceCents(),
		Currency:    product.GetCurrency().String(),
	}
}

func mapProtoProducts(products []*catalogv1.Product) []Product {
	converted := make([]Product, 0, len(products))
	for _, product := range products {
		if mapped := mapProtoProduct(product); mapped != nil {
			converted = append(converted, *mapped)
		}
	}

	return converted
}

func parseCurrency(value string) (catalogv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return catalogv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return catalogv1.Currency_CURRENCY_EUR, nil
	default:
		return catalogv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedCurrency, value)
	}
}
