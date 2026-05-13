package handler

import (
	"context"
	"errors"
	"log/slog"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/service"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	service *service.PaymentService
	logger  *slog.Logger
}

func NewGRPCHandler(service *service.PaymentService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *GRPCHandler) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	payment, err := h.service.CreatePayment(ctx, domain.Payment{
		OrderID:              req.GetOrderId(),
		CustomerID:           req.GetCustomerId(),
		Amount:               convertMoney(req.GetAmount()),
		PaymentMethod:        req.GetPaymentMethod(),
		PaymentMethodDetails: req.GetPaymentMethodDetails(),
		IdempotencyKey:       req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapPaymentError(err)
	}

	h.logger.InfoContext(
		ctx,
		"payment created",
		requestIDAttr(ctx),
		slog.String("payment_id", payment.ID),
		slog.String("order_id", payment.OrderID),
		slog.String("customer_id", payment.CustomerID),
		slog.String("status", payment.Status.String()),
	)

	return &paymentv1.CreatePaymentResponse{
		Payment: convertPaymentToProto(payment),
	}, nil
}

func (h *GRPCHandler) GetPaymentByID(ctx context.Context, req *paymentv1.GetPaymentByIDRequest) (*paymentv1.GetPaymentByIDResponse, error) {
	payment, err := h.service.GetPaymentByID(ctx, req.GetPaymentId())
	if err != nil {
		return nil, mapPaymentError(err)
	}

	return &paymentv1.GetPaymentByIDResponse{
		Payment: convertPaymentToProto(payment),
	}, nil
}

func (h *GRPCHandler) GetPaymentByOrderID(ctx context.Context, req *paymentv1.GetPaymentByOrderIDRequest) (*paymentv1.GetPaymentByOrderIDResponse, error) {
	payment, err := h.service.GetPaymentByOrderID(ctx, req.GetOrderId())
	if err != nil {
		return nil, mapPaymentError(err)
	}

	return &paymentv1.GetPaymentByOrderIDResponse{
		Payment: convertPaymentToProto(payment),
	}, nil
}

func (h *GRPCHandler) CancelPayment(ctx context.Context, req *paymentv1.CancelPaymentRequest) (*paymentv1.CancelPaymentResponse, error) {
	payment, err := h.service.CancelPayment(ctx, req.GetPaymentId(), req.GetReason())
	if err != nil {
		return nil, mapPaymentError(err)
	}

	h.logger.InfoContext(
		ctx,
		"payment cancelled",
		requestIDAttr(ctx),
		slog.String("payment_id", payment.ID),
		slog.String("order_id", payment.OrderID),
		slog.String("reason", payment.CancelReason),
	)

	return &paymentv1.CancelPaymentResponse{
		Payment: convertPaymentToProto(payment),
	}, nil
}

func convertMoney(money *paymentv1.Money) domain.Money {
	if money == nil {
		return domain.Money{}
	}

	return domain.Money{
		Currency:    money.GetCurrency(),
		AmountCents: money.GetAmountCents(),
	}
}

func convertMoneyToProto(money domain.Money) *paymentv1.Money {
	return &paymentv1.Money{
		Currency:    money.Currency,
		AmountCents: money.AmountCents,
	}
}

func convertPaymentToProto(payment domain.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		PaymentId:     payment.ID,
		OrderId:       payment.OrderID,
		CustomerId:    payment.CustomerID,
		Amount:        convertMoneyToProto(payment.Amount),
		PaymentMethod: payment.PaymentMethod,
		Status:        payment.Status,
	}
}

func mapPaymentError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidPayment),
		errors.Is(err, domain.ErrInvalidPaymentID),
		errors.Is(err, domain.ErrInvalidIdempotencyKey):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrPaymentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrPaymentAlreadyExists),
		errors.Is(err, domain.ErrIdempotencyKeyAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrPaymentCannotBeCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func requestIDAttr(ctx context.Context) slog.Attr {
	return slog.String("request_id", requestid.FromContext(ctx))
}
