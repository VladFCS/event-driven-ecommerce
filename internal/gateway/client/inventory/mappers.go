package inventory

import inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"

func mapProtoStock(stock *inventoryv1.Stock) *Stock {
	if stock == nil {
		return nil
	}

	return &Stock{
		ProductID:         stock.GetProductId(),
		AvailableQuantity: stock.GetAvailableQuantity(),
		ReservedQuantity:  stock.GetReservedQuantity(),
		TotalQuantity:     stock.GetTotalQuantity(),
	}
}
