package service

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput             = errors.New("invalid input")
	ErrUnsupportedCurrency      = errors.New("unsupported currency")
	ErrUnsupportedPaymentMethod = errors.New("unsupported payment method")
	ErrDownstreamNotFound       = errors.New("downstream resource not found")
	ErrDownstreamFailed         = errors.New("downstream service failed")
)

func wrapDownstreamError(operation string, err error) error {
	if err == nil {
		return nil
	}

	if status.Code(err) == codes.NotFound {
		return fmt.Errorf("%w: %s: %v", ErrDownstreamNotFound, operation, err)
	}

	return fmt.Errorf("%w: %s: %v", ErrDownstreamFailed, operation, err)
}
