package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func (h *HTTPHandler) GetPaymentByID(c *gin.Context) {
	var req GetPaymentByIDURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.gatewayService.GetPaymentByID(c.Request.Context(), &gatewayservice.GetPaymentByIDInput{
		PaymentID: req.PaymentID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGetPaymentByIDResponse(resp))
}

func toGetPaymentByIDResponse(result *gatewayservice.GetPaymentByIDResult) *GetPaymentByIDResponse {
	return &GetPaymentByIDResponse{
		PaymentID:  result.PaymentID,
		OrderID:    result.OrderID,
		CustomerID: result.CustomerID,
		Status:     result.Status,
		Amount: MoneyResponse{
			Currency:    result.Amount.Currency,
			AmountCents: result.Amount.AmountCents,
		},
		PaymentMethod: result.PaymentMethod,
	}
}
