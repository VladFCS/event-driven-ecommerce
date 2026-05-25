package domain

import (
	"errors"
)

type Stock struct {
	ProductID         string
	AvailableQuantity int64
	ReservedQuantity  int64
	TotalQuantity     int64
}

type StockReservation struct {
	ProductID string
	OrderID   string
	Quantity  int64
}

var (
	ErrStockNotFound       = errors.New("stock not found")
	ErrInvalidStock        = errors.New("invalid stock")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrInvalidProductID = errors.New("invalid product")
)
