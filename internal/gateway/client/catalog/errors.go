package catalog

import "errors"

var (
	ErrCreateProductRequestNil = errors.New("create product request is nil")
	ErrProductIDRequired       = errors.New("product id is required")
	ErrListProductsRequestNil  = errors.New("list products request is nil")
	ErrUnsupportedCurrency     = errors.New("unsupported catalog currency")
)
