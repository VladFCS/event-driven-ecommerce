package catalog

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type GetProductByIDResponse struct {
	Product *Product
}
