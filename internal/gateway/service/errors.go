package service

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput             = errors.New("invalid input")
	ErrIdempotencyConflict      = errors.New("idempotency key conflict")
	ErrUnsupportedCurrency      = errors.New("unsupported currency")
	ErrUnsupportedPaymentMethod = errors.New("unsupported payment method")
	ErrDownstreamNotFound       = errors.New("downstream resource not found")
	ErrDownstreamFailed         = errors.New("downstream service failed")
	ErrTimeout                  = errors.New("operation timed out")
	ErrRequestCanceled          = errors.New("request canceled")
)

func wrapDownstreamError(operation string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded), status.Code(err) == codes.DeadlineExceeded:
		return fmt.Errorf("%w: %s: %v", ErrTimeout, operation, err)
	case errors.Is(err, context.Canceled), status.Code(err) == codes.Canceled:
		return fmt.Errorf("%w: %s: %v", ErrRequestCanceled, operation, err)
	case status.Code(err) == codes.NotFound:
		return fmt.Errorf("%w: %s: %v", ErrDownstreamNotFound, operation, err)
	}

	return fmt.Errorf("%w: %s: %v", ErrDownstreamFailed, operation, err)
}
