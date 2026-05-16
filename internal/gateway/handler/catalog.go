package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

const (
	defaultProductListPage     = 1
	defaultProductListPageSize = 20
	maxProductListPageSize     = 100
)

func (h *HTTPHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err, req, "invalid request body")
		return
	}

	resp, err := h.gatewayService.CreateProduct(c.Request.Context(), &gatewayservice.CreateProductInput{
		ProductID:   req.ProductID,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toCreateProductResponse(resp))
}

func (h *HTTPHandler) UpdateProduct(c *gin.Context) {
	var uriReq UpdateProductURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		writeBindError(c, err, uriReq, "invalid request path parameters")
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err, req, "invalid request body")
		return
	}

	resp, err := h.gatewayService.UpdateProduct(c.Request.Context(), &gatewayservice.UpdateProductInput{
		ProductID:   uriReq.ProductID,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUpdateProductResponse(resp))
}

func (h *HTTPHandler) DeleteProduct(c *gin.Context) {
	var req DeleteProductURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeBindError(c, err, req, "invalid request path parameters")
		return
	}

	if err := h.gatewayService.DeleteProduct(c.Request.Context(), &gatewayservice.DeleteProductInput{
		ProductID: req.ProductID,
	}); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *HTTPHandler) GetProductByID(c *gin.Context) {
	var req GetProductByIDURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeBindError(c, err, req, "invalid request path parameters")
		return
	}

	resp, err := h.gatewayService.GetProductByID(c.Request.Context(), &gatewayservice.GetProductByIDInput{
		ProductID: req.ProductID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGetProductByIDResponse(resp))
}

func toGetProductByIDResponse(result *gatewayservice.GetProductByIDResult) *GetProductByIDResponse {
	return &GetProductByIDResponse{
		ProductID:   result.ProductID,
		Name:        result.Name,
		Description: result.Description,
		PriceCents:  result.PriceCents,
		Currency:    result.Currency,
	}
}

func (h *HTTPHandler) ListProducts(c *gin.Context) {
	var queryReq ListProductsQueryRequest
	if err := c.ShouldBindQuery(&queryReq); err != nil {
		writeBindError(c, err, queryReq, "invalid query parameters")
		return
	}

	page := queryReq.Page
	if page <= 0 {
		page = defaultProductListPage
	}

	pageSize := queryReq.PageSize
	if pageSize <= 0 {
		pageSize = defaultProductListPageSize
	}
	if pageSize > maxProductListPageSize {
		pageSize = maxProductListPageSize
	}

	resp, err := h.gatewayService.ListProducts(c.Request.Context(), &gatewayservice.ListProductsInput{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListProductsResponse(resp))
}

func toListProductsResponse(result *gatewayservice.ListProductsResult) *ListProductsResponse {
	response := &ListProductsResponse{
		Products: make([]GetProductByIDResponse, 0, len(result.Products)),
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}

	for _, product := range result.Products {
		product := product
		response.Products = append(response.Products, GetProductByIDResponse{
			ProductID:   product.ProductID,
			Name:        product.Name,
			Description: product.Description,
			PriceCents:  product.PriceCents,
			Currency:    product.Currency,
		})
	}

	return response
}

func toCreateProductResponse(result *gatewayservice.CreateProductResult) *CreateProductResponse {
	return &CreateProductResponse{
		ProductID:   result.ProductID,
		Name:        result.Name,
		Description: result.Description,
		PriceCents:  result.PriceCents,
		Currency:    result.Currency,
	}
}

func toUpdateProductResponse(result *gatewayservice.UpdateProductResult) *UpdateProductResponse {
	return &UpdateProductResponse{
		ProductID:   result.ProductID,
		Name:        result.Name,
		Description: result.Description,
		PriceCents:  result.PriceCents,
		Currency:    result.Currency,
	}
}
