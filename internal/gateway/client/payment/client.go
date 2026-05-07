package payment

import (
	"context"
	"errors"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"google.golang.org/grpc"
)

type CreatePaymentRequest struct {
	OrderID              string
	CustomerID           string
	Amount               *paymentv1.Money
	PaymentMethod        paymentv1.PaymentMethodType
	PaymentMethodDetails string
	IdempotencyKey       string
}

type CreatePaymentResponse struct {
	Payment *paymentv1.Payment
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
		return nil, errors.New("create payment request is nil")
	}

	grpcResp, err := c.grpcClient.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		OrderId:              req.OrderID,
		CustomerId:           req.CustomerID,
		Amount:               req.Amount,
		PaymentMethod:        req.PaymentMethod,
		PaymentMethodDetails: req.PaymentMethodDetails,
		IdempotencyKey:       req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		Payment: grpcResp.GetPayment(),
	}, nil
}
