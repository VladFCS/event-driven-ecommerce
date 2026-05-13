package catalog

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type CreateProductRequest struct {
	ProductID   string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type CreateProductResponse struct {
	Product *Product
}

type GetProductByIDResponse struct {
	Product *Product
}

type ListProductsRequest struct {
	Page     int
	PageSize int
}

type ListProductsResponse struct {
	Products []Product
	Page     int
	PageSize int
	Total    int64
}
