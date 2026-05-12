package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *HTTPHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "gateway-service",
	})
}

func (h *HTTPHandler) Readyz(c *gin.Context) {
	if h.gatewayService == nil {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:              "not_ready",
			Service:             "gateway-service",
			MissingDependencies: []string{"gateway_service"},
		})
		return
	}

	readiness := h.gatewayService.ReadinessStatus()
	if !readiness.Ready {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:              "not_ready",
			Service:             "gateway-service",
			MissingDependencies: readiness.MissingDependencies,
		})
		return
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ready",
		Service: "gateway-service",
	})
}
