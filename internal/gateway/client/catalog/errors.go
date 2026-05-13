package catalog

import "errors"

var (
	ErrProductIDRequired      = errors.New("product id is required")
	ErrListProductsRequestNil = errors.New("list products request is nil")
)
