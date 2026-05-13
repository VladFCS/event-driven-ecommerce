package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func (h *HTTPHandler) GetPaymentByID(c *gin.Context) {
	var req GetPaymentByIDURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeBindError(c, err, req, "invalid request path parameters")
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

func (h *HTTPHandler) CancelPayment(c *gin.Context) {
	var uriReq CancelPaymentURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		writeBindError(c, err, uriReq, "invalid request path parameters")
		return
	}

	var bodyReq CancelPaymentRequest
	if err := c.ShouldBindJSON(&bodyReq); err != nil {
		writeBindError(c, err, bodyReq, "invalid request body")
		return
	}

	resp, err := h.gatewayService.CancelPayment(c.Request.Context(), &gatewayservice.CancelPaymentInput{
		PaymentID: uriReq.PaymentID,
		Reason:    bodyReq.Reason,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toCancelPaymentResponse(resp))
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

func toCancelPaymentResponse(result *gatewayservice.CancelPaymentResult) *CancelPaymentResponse {
	return &CancelPaymentResponse{
		PaymentID:  result.PaymentID,
		OrderID:    result.OrderID,
		CustomerID: result.CustomerID,
		Status:     result.Status,
	}
}
