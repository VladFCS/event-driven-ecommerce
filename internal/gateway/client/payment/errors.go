package payment

import "errors"

var (
	ErrCreatePaymentRequestNil    = errors.New("create payment request is nil")
	ErrGetPaymentRequestNil       = errors.New("get payment request is nil")
	ErrPaymentIDRequired          = errors.New("payment id is required")
	ErrUnsupportedPaymentCurrency = errors.New("unsupported payment currency")
	ErrUnsupportedPaymentMethod   = errors.New("unsupported payment method")
)
