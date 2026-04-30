package domain

type Stock struct {
	ProductID string
	AvailableQuantity int32
	ReservedQuantity int32
	TotalQuantity int32
}