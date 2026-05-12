package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
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
