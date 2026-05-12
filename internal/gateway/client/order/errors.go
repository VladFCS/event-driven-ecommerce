package order

import "errors"

var (
	ErrCreateOrderRequestNil    = errors.New("create order request is nil")
	ErrCancelOrderRequestNil    = errors.New("cancel order request is nil")
	ErrListOrdersRequestNil     = errors.New("list orders by customer request is nil")
	ErrOrderIDRequired          = errors.New("order id is required")
	ErrCustomerIDRequired       = errors.New("customer id is required")
	ErrUnsupportedOrderCurrency = errors.New("unsupported order currency")
)
