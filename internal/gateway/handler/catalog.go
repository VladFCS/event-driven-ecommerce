package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

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
