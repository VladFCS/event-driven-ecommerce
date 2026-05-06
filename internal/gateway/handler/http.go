package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CheckoutService interface {
	Checkout(c *gin.Context, req *CheckoutRequest) (*CheckoutResponse, error)
	GetOrderByID(c *gin.Context, req *GetOrderByIDRequest) (*GetOrderByIDResponse, error)
}

type HTTPHandler struct {
	checkoutService CheckoutService
}

func NewHTTPHandler(checkoutService CheckoutService) *HTTPHandler {
	return &HTTPHandler{checkoutService: checkoutService}
}

type CheckoutRequest struct {
	CustomerID      string                `json:"customer_id" binding:"required"`
	Items           []CheckoutItemRequest `json:"items" binding:"required,min=1"`
	ShippingAddress AddressRequest        `json:"shipping_address" binding:"required"`
	Payment         PaymentRequest        `json:"payment" binding:"required"`
	IdempotencyKey  string                `json:"idempotency_key"`
}

type GetOrderByIDRequest struct {
	OrderID string `uri:"order_id" binding:"required"`
}

type GetOrderByIDResponse struct {
	OrderID         string                `json:"order_id"`
	CustomerID      string                `json:"customer_id"`
	OrderStatus     string                `json:"order_status"`
	PaymentStatus   string                `json:"payment_status"`
	Items           []CheckoutItemRequest `json:"items"`
	ShippingAddress AddressRequest        `json:"shipping_address"`
	Payment         PaymentRequest        `json:"payment"`
}

type CheckoutItemRequest struct {
	ProductID   string       `json:"product_id" binding:"required"`
	SKU         string       `json:"sku"`
	ProductName string       `json:"product_name"`
	Quantity    int32        `json:"quantity" binding:"required,gt=0"`
	UnitPrice   MoneyRequest `json:"unit_price" binding:"required"`
}

type MoneyRequest struct {
	Currency    string `json:"currency" binding:"required"`
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
}

type AddressRequest struct {
	Country    string `json:"country" binding:"required"`
	City       string `json:"city" binding:"required"`
	Street     string `json:"street" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	House      string `json:"house" binding:"required"`
	Apartment  string `json:"apartment"`
}

type PaymentRequest struct {
	Method        string `json:"method" binding:"required"`
	MethodDetails string `json:"method_details"`
}

type CheckoutResponse struct {
	OrderID       string `json:"order_id"`
	PaymentID     string `json:"payment_id"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
}

func (h *HTTPHandler) Register(r *gin.Engine) {
	r.POST("/checkout", h.Checkout)
	r.GET("/orders/:order_id", h.GetOrderByID)
}

func (h *HTTPHandler) Checkout(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.checkoutService.Checkout(c, &req)
	if err != nil {
		// later: map domain/downstream errors properly
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *HTTPHandler) GetOrderByID(c *gin.Context) {
	var req GetOrderByIDRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.checkoutService.GetOrderByID(c, &req)
	if err != nil {
		// later: map domain/downstream errors properly
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
