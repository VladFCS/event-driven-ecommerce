package service

import (
	"context"
	"fmt"
	"strings"

	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
)

func (s *GatewayService) GetPaymentByID(ctx context.Context, in *GetPaymentByIDInput) (*GetPaymentByIDResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: get payment by id request is nil", ErrInvalidInput)
	}
	if strings.TrimSpace(in.PaymentID) == "" {
		return nil, fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	paymentResp, err := s.paymentClient.GetPaymentByID(opCtx, &paymentclient.GetPaymentByIDRequest{
		PaymentID: strings.TrimSpace(in.PaymentID),
	})
	if err != nil {
		return nil, wrapDownstreamError("payment get", err)
	}

	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, fmt.Errorf("%w: payment response is empty", ErrDownstreamFailed)
	}

	payment := paymentResp.Payment

	return &GetPaymentByIDResult{
		PaymentID:  payment.ID,
		OrderID:    payment.OrderID,
		CustomerID: payment.CustomerID,
		Status:     payment.Status,
		Amount: Money{
			Currency:    payment.Amount.Currency,
			AmountCents: payment.Amount.AmountCents,
		},
		PaymentMethod: payment.PaymentMethod,
	}, nil
}

func (s *GatewayService) GetPaymentByOrderID(ctx context.Context, in *GetPaymentByOrderIDInput) (*GetPaymentByOrderIDResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: get payment by order id request is nil", ErrInvalidInput)
	}

	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	paymentResp, err := s.paymentClient.GetPaymentByOrderID(opCtx, &paymentclient.GetPaymentByOrderIDRequest{
		OrderID: orderID,
	})
	if err != nil {
		return nil, wrapDownstreamError("payment get by order", err)
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, fmt.Errorf("%w: payment response is empty", ErrDownstreamFailed)
	}

	payment := paymentResp.Payment
	return &GetPaymentByOrderIDResult{
		PaymentID:  payment.ID,
		OrderID:    payment.OrderID,
		CustomerID: payment.CustomerID,
		Status:     payment.Status,
		Amount: Money{
			Currency:    payment.Amount.Currency,
			AmountCents: payment.Amount.AmountCents,
		},
		PaymentMethod: payment.PaymentMethod,
	}, nil
}

func (s *GatewayService) ListPaymentsByCustomer(ctx context.Context, in *ListPaymentsByCustomerInput) (*ListPaymentsByCustomerResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: list payments by customer request is nil", ErrInvalidInput)
	}

	customerID := strings.TrimSpace(in.CustomerID)
	if customerID == "" {
		return nil, fmt.Errorf("%w: customer id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.readTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.readTimeout)
	}
	defer cancel()

	paymentResp, err := s.paymentClient.ListPaymentsByCustomer(opCtx, &paymentclient.ListPaymentsByCustomerRequest{
		CustomerID: customerID,
		Page:       in.Page,
		PageSize:   in.PageSize,
	})
	if err != nil {
		return nil, wrapDownstreamError("payment list by customer", err)
	}
	if paymentResp == nil {
		return nil, fmt.Errorf("%w: list payments response is empty", ErrDownstreamFailed)
	}

	result := &ListPaymentsByCustomerResult{
		Payments: make([]PaymentResult, 0, len(paymentResp.Payments)),
		Page:     paymentResp.Page,
		PageSize: paymentResp.PageSize,
		Total:    paymentResp.Total,
	}

	for _, payment := range paymentResp.Payments {
		result.Payments = append(result.Payments, PaymentResult{
			PaymentID:  payment.ID,
			OrderID:    payment.OrderID,
			CustomerID: payment.CustomerID,
			Status:     payment.Status,
			Amount: Money{
				Currency:    payment.Amount.Currency,
				AmountCents: payment.Amount.AmountCents,
			},
			PaymentMethod: payment.PaymentMethod,
		})
	}

	return result, nil
}

func (s *GatewayService) CancelPayment(ctx context.Context, in *CancelPaymentInput) (*CancelPaymentResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: cancel payment request is nil", ErrInvalidInput)
	}

	paymentID := strings.TrimSpace(in.PaymentID)
	if paymentID == "" {
		return nil, fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}

	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: cancel reason is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	paymentResp, err := s.paymentClient.CancelPayment(opCtx, &paymentclient.CancelPaymentRequest{
		PaymentID: paymentID,
		Reason:    reason,
	})
	if err != nil {
		return nil, wrapDownstreamError("payment cancel", err)
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, fmt.Errorf("%w: cancel payment response is empty", ErrDownstreamFailed)
	}

	payment := paymentResp.Payment
	return &CancelPaymentResult{
		PaymentID:  payment.ID,
		OrderID:    payment.OrderID,
		CustomerID: payment.CustomerID,
		Status:     payment.Status,
	}, nil
}

func (s *GatewayService) CapturePayment(ctx context.Context, in *CapturePaymentInput) (*CapturePaymentResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: capture payment request is nil", ErrInvalidInput)
	}
	paymentID := strings.TrimSpace(in.PaymentID)
	if paymentID == "" {
		return nil, fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}

	opCtx := ctx
	cancel := func() {}
	if s.checkoutTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, s.checkoutTimeout)
	}
	defer cancel()

	paymentResp, err := s.paymentClient.CapturePayment(opCtx, &paymentclient.CapturePaymentRequest{
		PaymentID: paymentID,
	})
	if err != nil {
		return nil, wrapDownstreamError("payment capture", err)
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, fmt.Errorf("%w: capture payment response is empty", ErrDownstreamFailed)
	}

	payment := paymentResp.Payment
	return &CapturePaymentResult{
		PaymentID:  payment.ID,
		OrderID:    payment.OrderID,
		CustomerID: payment.CustomerID,
		Status:     payment.Status,
	}, nil
}
