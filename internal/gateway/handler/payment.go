package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

const (
	defaultPaymentListPage     = 1
	defaultPaymentListPageSize = 20
	maxPaymentListPageSize     = 100
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

func (h *HTTPHandler) GetPaymentByOrderID(c *gin.Context) {
	var req GetPaymentByOrderIDURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeBindError(c, err, req, "invalid request path parameters")
		return
	}

	resp, err := h.gatewayService.GetPaymentByOrderID(c.Request.Context(), &gatewayservice.GetPaymentByOrderIDInput{
		OrderID: req.OrderID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGetPaymentByOrderIDResponse(resp))
}

func (h *HTTPHandler) ListPaymentsByCustomer(c *gin.Context) {
	var uriReq ListPaymentsByCustomerURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		writeBindError(c, err, uriReq, "invalid request path parameters")
		return
	}

	var queryReq ListPaymentsByCustomerQueryRequest
	if err := c.ShouldBindQuery(&queryReq); err != nil {
		writeBindError(c, err, queryReq, "invalid query parameters")
		return
	}

	page := queryReq.Page
	if page <= 0 {
		page = defaultPaymentListPage
	}

	pageSize := queryReq.PageSize
	if pageSize <= 0 {
		pageSize = defaultPaymentListPageSize
	}
	if pageSize > maxPaymentListPageSize {
		pageSize = maxPaymentListPageSize
	}

	resp, err := h.gatewayService.ListPaymentsByCustomer(c.Request.Context(), &gatewayservice.ListPaymentsByCustomerInput{
		CustomerID: uriReq.CustomerID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListPaymentsByCustomerResponse(resp))
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

func toGetPaymentByOrderIDResponse(result *gatewayservice.GetPaymentByOrderIDResult) *GetPaymentByOrderIDResponse {
	return &GetPaymentByOrderIDResponse{
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

func toListPaymentsByCustomerResponse(result *gatewayservice.ListPaymentsByCustomerResult) *ListPaymentsByCustomerResponse {
	response := &ListPaymentsByCustomerResponse{
		Payments: make([]GetPaymentByIDResponse, 0, len(result.Payments)),
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}

	for _, payment := range result.Payments {
		response.Payments = append(response.Payments, GetPaymentByIDResponse{
			PaymentID:  payment.PaymentID,
			OrderID:    payment.OrderID,
			CustomerID: payment.CustomerID,
			Status:     payment.Status,
			Amount: MoneyResponse{
				Currency:    payment.Amount.Currency,
				AmountCents: payment.Amount.AmountCents,
			},
			PaymentMethod: payment.PaymentMethod,
		})
	}

	return response
}
