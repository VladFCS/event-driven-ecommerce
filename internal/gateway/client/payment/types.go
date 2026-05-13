package payment

type Money struct {
	Currency    string
	AmountCents int64
}

type Payment struct {
	ID            string
	OrderID       string
	CustomerID    string
	Amount        Money
	PaymentMethod string
	Status        string
}

type CreatePaymentRequest struct {
	OrderID              string
	CustomerID           string
	Amount               Money
	PaymentMethod        string
	PaymentMethodDetails string
	IdempotencyKey       string
}

type GetPaymentByIDRequest struct {
	PaymentID string
}

type GetPaymentByIDResponse struct {
	Payment *Payment
}

type GetPaymentByOrderIDRequest struct {
	OrderID string
}

type GetPaymentByOrderIDResponse struct {
	Payment *Payment
}

type CancelPaymentRequest struct {
	PaymentID string
	Reason    string
}

type CancelPaymentResponse struct {
	Payment *Payment
}

type CreatePaymentResponse struct {
	Payment *Payment
}
