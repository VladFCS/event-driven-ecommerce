package payment

import "errors"

var (
	ErrCreatePaymentRequestNil       = errors.New("create payment request is nil")
	ErrCancelPaymentRequestNil       = errors.New("cancel payment request is nil")
	ErrGetPaymentByIDRequestNil      = errors.New("get payment by id request is nil")
	ErrGetPaymentByOrderIDRequestNil = errors.New("get payment by order id request is nil")
	ErrListPaymentsRequestNil        = errors.New("list payments by customer request is nil")
	ErrCustomerIDRequired            = errors.New("customer id is required")
	ErrOrderIDRequired               = errors.New("order id is required")
	ErrPaymentIDRequired             = errors.New("payment id is required")
	ErrUnsupportedPaymentCurrency    = errors.New("unsupported payment currency")
	ErrUnsupportedPaymentMethod      = errors.New("unsupported payment method")
)
