package inventory

import "errors"

var (
	ErrProductIDRequired      = errors.New("product id is required")
	ErrGetStockRequestNil     = errors.New("get stock request is nil")
	ErrReserveStockRequestNil = errors.New("reserve stock request is nil")
	ErrReleaseStockRequestNil = errors.New("release stock request is nil")
)
