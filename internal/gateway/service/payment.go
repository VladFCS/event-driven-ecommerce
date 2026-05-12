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
