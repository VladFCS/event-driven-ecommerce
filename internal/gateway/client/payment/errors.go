package payment

import "errors"

var (
	ErrCreatePaymentRequestNil    = errors.New("create payment request is nil")
	ErrGetPaymentByIDRequestNil   = errors.New("get payment by id request is nil")
	ErrPaymentIDRequired          = errors.New("payment id is required")
	ErrUnsupportedPaymentCurrency = errors.New("unsupported payment currency")
	ErrUnsupportedPaymentMethod   = errors.New("unsupported payment method")
)
