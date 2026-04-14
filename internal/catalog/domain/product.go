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