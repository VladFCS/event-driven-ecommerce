package domain

import "errors"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidProduct  = errors.New("invalid product")
)

type Product struct {
	ID string
	Name string
	Description string
	PriceCents float64
	Currency string
}