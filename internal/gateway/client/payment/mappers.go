package payment

import (
	"fmt"
	"strings"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
)

func mapMoneyToProto(money Money) (*paymentv1.Money, error) {
	currency, err := parseCurrency(money.Currency)
	if err != nil {
		return nil, err
	}

	return &paymentv1.Money{
		Currency:    currency,
		AmountCents: money.AmountCents,
	}, nil
}

func mapProtoPayment(payment *paymentv1.Payment) *Payment {
	if payment == nil {
		return nil
	}

	return &Payment{
		ID:            payment.GetPaymentId(),
		OrderID:       payment.GetOrderId(),
		CustomerID:    payment.GetCustomerId(),
		Amount:        mapProtoMoney(payment.GetAmount()),
		PaymentMethod: payment.GetPaymentMethod().String(),
		Status:        payment.GetStatus().String(),
	}
}

func mapProtoPayments(payments []*paymentv1.Payment) []Payment {
	converted := make([]Payment, 0, len(payments))
	for _, payment := range payments {
		if mapped := mapProtoPayment(payment); mapped != nil {
			converted = append(converted, *mapped)
		}
	}

	return converted
}

func mapProtoMoney(money *paymentv1.Money) Money {
	if money == nil {
		return Money{}
	}

	return Money{
		Currency:    money.GetCurrency().String(),
		AmountCents: money.GetAmountCents(),
	}
}

func parseCurrency(value string) (paymentv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return paymentv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return paymentv1.Currency_CURRENCY_EUR, nil
	default:
		return paymentv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedPaymentCurrency, value)
	}
}

func parsePaymentMethod(value string) (paymentv1.PaymentMethodType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CARD", "PAYMENT_METHOD_TYPE_CARD":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD, nil
	case "CASH", "PAYMENT_METHOD_TYPE_CASH":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH, nil
	default:
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedPaymentMethod, value)
	}
}
