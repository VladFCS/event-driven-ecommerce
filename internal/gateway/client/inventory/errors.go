package inventory

import "errors"

var (
	ErrReserveStockRequestNil = errors.New("reserve stock request is nil")
	ErrReleaseStockRequestNil = errors.New("release stock request is nil")
)
