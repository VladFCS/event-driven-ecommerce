package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
)

func (h *HTTPHandler) GetStockByProductID(c *gin.Context) {
	var req GetStockByProductIDURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeBindError(c, err, req, "invalid request path parameters")
		return
	}

	resp, err := h.gatewayService.GetStockByProductID(c.Request.Context(), &gatewayservice.GetStockByProductIDInput{
		ProductID: req.ProductID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGetStockByProductIDResponse(resp))
}

func toGetStockByProductIDResponse(result *gatewayservice.GetStockByProductIDResult) *GetStockByProductIDResponse {
	return &GetStockByProductIDResponse{
		ProductID: result.ProductID,
		Available: result.Available,
		Reserved:  result.Reserved,
	}
}
