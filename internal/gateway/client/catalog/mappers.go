package catalog

import catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"

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
