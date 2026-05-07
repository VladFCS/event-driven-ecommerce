package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

type CheckoutService interface {
	Checkout(ctx context.Context, in *gatewayservice.CheckoutInput) (*gatewayservice.CheckoutResult, error)
	GetOrderByID(ctx context.Context, in *gatewayservice.GetOrderByIDInput) (*gatewayservice.GetOrderByIDResult, error)
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
	OrderID         string              `json:"order_id"`
	CustomerID      string              `json:"customer_id"`
	OrderStatus     string              `json:"order_status"`
	Items           []OrderItemResponse `json:"items"`
	TotalAmount     MoneyResponse       `json:"total_amount"`
	ShippingAddress AddressResponse     `json:"shipping_address"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
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

type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	SKU         string        `json:"sku"`
	ProductName string        `json:"product_name"`
	Quantity    int32         `json:"quantity"`
	UnitPrice   MoneyResponse `json:"unit_price"`
	TotalPrice  MoneyResponse `json:"total_price"`
}

type MoneyResponse struct {
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

type AddressResponse struct {
	Country    string `json:"country"`
	City       string `json:"city"`
	Street     string `json:"street"`
	PostalCode string `json:"postal_code"`
	House      string `json:"house"`
	Apartment  string `json:"apartment"`
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

	resp, err := h.checkoutService.Checkout(c.Request.Context(), &gatewayservice.CheckoutInput{
		CustomerID:      req.CustomerID,
		Items:           mapCheckoutItems(req.Items),
		ShippingAddress: mapAddressRequest(req.ShippingAddress),
		IdempotencyKey:  req.IdempotencyKey,
		Payment: gatewayservice.PaymentDetails{
			Method:        req.Payment.Method,
			MethodDetails: req.Payment.MethodDetails,
		},
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, &CheckoutResponse{
		OrderID:       resp.OrderID,
		PaymentID:     resp.PaymentID,
		OrderStatus:   resp.OrderStatus,
		PaymentStatus: resp.PaymentStatus,
	})
}

func (h *HTTPHandler) GetOrderByID(c *gin.Context) {
	var req GetOrderByIDRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.checkoutService.GetOrderByID(c.Request.Context(), &gatewayservice.GetOrderByIDInput{
		OrderID: req.OrderID,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toGetOrderByIDResponse(resp))
}

func mapCheckoutItems(items []CheckoutItemRequest) []gatewayservice.CheckoutItem {
	mapped := make([]gatewayservice.CheckoutItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, gatewayservice.CheckoutItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice: gatewayservice.Money{
				Currency:    item.UnitPrice.Currency,
				AmountCents: item.UnitPrice.AmountCents,
			},
		})
	}

	return mapped
}

func mapAddressRequest(address AddressRequest) gatewayservice.Address {
	return gatewayservice.Address{
		Country:    address.Country,
		City:       address.City,
		Street:     address.Street,
		PostalCode: address.PostalCode,
		House:      address.House,
		Apartment:  address.Apartment,
	}
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
