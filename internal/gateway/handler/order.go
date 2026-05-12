package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

const (
	defaultOrderListPage     = 1
	defaultOrderListPageSize = 20
	maxOrderListPageSize     = 100
)

func (h *HTTPHandler) GetOrderByID(c *gin.Context) {
	var req GetOrderByIDRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.gatewayService.GetOrderByID(c.Request.Context(), &gatewayservice.GetOrderByIDInput{
		OrderID: req.OrderID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGetOrderByIDResponse(resp))
}

func (h *HTTPHandler) ListOrdersByCustomer(c *gin.Context) {
	var uriReq ListOrdersByCustomerURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var queryReq ListOrdersByCustomerQueryRequest
	if err := c.ShouldBindQuery(&queryReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page := queryReq.Page
	if page <= 0 {
		page = defaultOrderListPage
	}

	pageSize := queryReq.PageSize
	if pageSize <= 0 {
		pageSize = defaultOrderListPageSize
	}
	if pageSize > maxOrderListPageSize {
		pageSize = maxOrderListPageSize
	}

	resp, err := h.gatewayService.ListOrdersByCustomer(c.Request.Context(), &gatewayservice.ListOrdersByCustomerInput{
		CustomerID: uriReq.CustomerID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListOrdersByCustomerResponse(resp))
}

func toGetOrderByIDResponse(result *gatewayservice.GetOrderByIDResult) *GetOrderByIDResponse {
	response := &GetOrderByIDResponse{
		OrderID:         result.OrderID,
		CustomerID:      result.CustomerID,
		OrderStatus:     result.OrderStatus,
		Items:           make([]OrderItemResponse, 0, len(result.Items)),
		TotalAmount:     MoneyResponse(result.TotalAmount),
		ShippingAddress: AddressResponse(result.ShippingAddress),
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.UpdatedAt,
	}

	for _, item := range result.Items {
		response.Items = append(response.Items, OrderItemResponse{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   MoneyResponse(item.UnitPrice),
			TotalPrice:  MoneyResponse(item.TotalPrice),
		})
	}

	return response
}

func toListOrdersByCustomerResponse(result *gatewayservice.ListOrdersByCustomerResult) *ListOrdersByCustomerResponse {
	response := &ListOrdersByCustomerResponse{
		Orders:   make([]GetOrderByIDResponse, 0, len(result.Orders)),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}

	for _, order := range result.Orders {
		order := order
		response.Orders = append(response.Orders, *toGetOrderByIDResponse(&order))
	}

	return response
}
