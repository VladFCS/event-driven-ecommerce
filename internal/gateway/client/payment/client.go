package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

var (
	ErrCreatePaymentRequestNil    = errors.New("create payment request is nil")
	ErrGetPaymentRequestNil       = errors.New("get payment request is nil")
	ErrPaymentIDRequired          = errors.New("payment id is required")
	ErrUnsupportedPaymentCurrency = errors.New("unsupported payment currency")
	ErrUnsupportedPaymentMethod   = errors.New("unsupported payment method")
)

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

type CreatePaymentResponse struct {
	Payment *Payment
}

type GRPCClient struct {
	grpcClient paymentv1.PaymentServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: paymentv1.NewPaymentServiceClient(conn),
	}
}

func (c *GRPCClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	if req == nil {
		return nil, ErrCreatePaymentRequestNil
	}

	amount, err := mapMoneyToProto(req.Amount)
	if err != nil {
		return nil, err
	}

	paymentMethod, err := parsePaymentMethod(req.PaymentMethod)
	if err != nil {
		return nil, err
	}

	grpcResp, err := c.grpcClient.CreatePayment(requestid.WithOutgoingMetadata(ctx), &paymentv1.CreatePaymentRequest{
		OrderId:              req.OrderID,
		CustomerId:           req.CustomerID,
		Amount:               amount,
		PaymentMethod:        paymentMethod,
		PaymentMethodDetails: req.PaymentMethodDetails,
		IdempotencyKey:       req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		Payment: mapProtoPayment(grpcResp.GetPayment()),
	}, nil
}

func (c *GRPCClient) GetPaymentByID(ctx context.Context, req *GetPaymentByIDRequest) (*GetPaymentByIDResponse, error) {
	if req == nil {
		return nil, ErrGetPaymentRequestNil
	}
	if strings.TrimSpace(req.PaymentID) == "" {
		return nil, ErrPaymentIDRequired
	}

	grpcResp, err := c.grpcClient.GetPayment(requestid.WithOutgoingMetadata(ctx), &paymentv1.GetPaymentRequest{
		PaymentId: req.PaymentID,
	})
	if err != nil {
		return nil, err
	}

	return &GetPaymentByIDResponse{
		Payment: mapProtoPayment(grpcResp.GetPayment()),
	}, nil
}

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
