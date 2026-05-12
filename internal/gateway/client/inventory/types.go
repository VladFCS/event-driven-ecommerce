package inventory

type Stock struct {
	ProductID         string
	AvailableQuantity int64
	ReservedQuantity  int64
	TotalQuantity     int64
}

type ReserveStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReserveStockResponse struct {
	Stock *Stock
}

type ReleaseStockRequest struct {
	ProductID string
	Quantity  int64
	OrderID   string
}

type ReleaseStockResponse struct {
	Stock *Stock
}
