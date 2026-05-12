package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func (h *HTTPHandler) Checkout(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err, req, "invalid request body")
		return
	}

	resp, err := h.gatewayService.Checkout(c.Request.Context(), &gatewayservice.CheckoutInput{
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
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, &CheckoutResponse{
		OrderID:       resp.OrderID,
		PaymentID:     resp.PaymentID,
		OrderStatus:   resp.OrderStatus,
		PaymentStatus: resp.PaymentStatus,
	})
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
