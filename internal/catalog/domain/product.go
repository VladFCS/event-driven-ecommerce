package domain

import (
	"errors"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidProduct  = errors.New("invalid product")
)

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Currency    catalogv1.Currency
}

type ProductPatch struct {
	Name        *string
	Description *string
	PriceCents  *int64
	Currency    *catalogv1.Currency
}

func (p ProductPatch) Empty() bool {
	return p.Name == nil && p.Description == nil && p.PriceCents == nil && p.Currency == nil
}
