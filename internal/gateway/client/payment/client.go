package payment

import (
	"context"
	"strings"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

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
		return nil, ErrGetPaymentByIDRequestNil
	}
	if strings.TrimSpace(req.PaymentID) == "" {
		return nil, ErrPaymentIDRequired
	}

	grpcResp, err := c.grpcClient.GetPaymentByID(requestid.WithOutgoingMetadata(ctx), &paymentv1.GetPaymentByIDRequest{
		PaymentId: req.PaymentID,
	})
	if err != nil {
		return nil, err
	}

	return &GetPaymentByIDResponse{
		Payment: mapProtoPayment(grpcResp.GetPayment()),
	}, nil
}

func (c *GRPCClient) GetPaymentByOrderID(ctx context.Context, req *GetPaymentByOrderIDRequest) (*GetPaymentByOrderIDResponse, error) {
	if req == nil {
		return nil, ErrGetPaymentByOrderIDRequestNil
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return nil, ErrOrderIDRequired
	}

	grpcResp, err := c.grpcClient.GetPaymentByOrderID(requestid.WithOutgoingMetadata(ctx), &paymentv1.GetPaymentByOrderIDRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, err
	}

	return &GetPaymentByOrderIDResponse{
		Payment: mapProtoPayment(grpcResp.GetPayment()),
	}, nil
}

func (c *GRPCClient) ListPaymentsByCustomer(ctx context.Context, req *ListPaymentsByCustomerRequest) (*ListPaymentsByCustomerResponse, error) {
	if req == nil {
		return nil, ErrListPaymentsRequestNil
	}

	customerID := strings.TrimSpace(req.CustomerID)
	if customerID == "" {
		return nil, ErrCustomerIDRequired
	}

	grpcResp, err := c.grpcClient.ListPaymentsByCustomer(requestid.WithOutgoingMetadata(ctx), &paymentv1.ListPaymentsByCustomerRequest{
		CustomerId: customerID,
		Page:       int32(req.Page),
		PageSize:   int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	return &ListPaymentsByCustomerResponse{
		Payments: mapProtoPayments(grpcResp.GetPayments()),
		Page:     int(grpcResp.GetPage()),
		PageSize: int(grpcResp.GetPageSize()),
		Total:    grpcResp.GetTotal(),
	}, nil
}

func (c *GRPCClient) CancelPayment(ctx context.Context, req *CancelPaymentRequest) (*CancelPaymentResponse, error) {
	if req == nil {
		return nil, ErrCancelPaymentRequestNil
	}

	paymentID := strings.TrimSpace(req.PaymentID)
	if paymentID == "" {
		return nil, ErrPaymentIDRequired
	}

	grpcResp, err := c.grpcClient.CancelPayment(requestid.WithOutgoingMetadata(ctx), &paymentv1.CancelPaymentRequest{
		PaymentId: paymentID,
		Reason:    strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return nil, err
	}

	return &CancelPaymentResponse{
		Payment: mapProtoPayment(grpcResp.GetPayment()),
	}, nil
}
