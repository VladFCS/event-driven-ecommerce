package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
	"github.com/vladfc/event-driven-ecommerce-app/internal/gateway/handler"
	gatewayservice "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := newLogger("gateway-service")
	httpPort := getenv("HTTP_PORT", "8080")
	orderServiceAddr := getenv("ORDER_SERVICE_ADDR", "localhost:50054")
	inventoryServiceAddr := getenv("INVENTORY_SERVICE_ADDR", "localhost:50052")
	paymentServiceAddr := getenv("PAYMENT_SERVICE_ADDR", "localhost:50053")

	orderConn, err := grpc.Dial(orderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to order-service", slog.Any("error", err))
		os.Exit(1)
	}
	defer orderConn.Close()

	inventoryConn, err := grpc.Dial(inventoryServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to inventory-service", slog.Any("error", err))
		os.Exit(1)
	}
	defer inventoryConn.Close()

	paymentConn, err := grpc.Dial(paymentServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to payment-service", slog.Any("error", err))
		os.Exit(1)
	}
	defer paymentConn.Close()

	orderSvcClient := orderclient.NewClient(orderConn)
	inventorySvcClient := inventoryclient.NewClient(inventoryConn)
	paymentSvcClient := paymentclient.NewClient(paymentConn)
	gatewaySvc := gatewayservice.NewGatewayService(
		orderSvcClient,
		gatewayservice.WithInventoryClient(inventorySvcClient),
		gatewayservice.WithPaymentClient(paymentSvcClient),
	)
	httpHandler := handler.NewHTTPHandler(gatewaySvc)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	httpHandler.Register(router)

	server := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("gateway-service started", slog.String("http_port", httpPort), slog.String("order_service_addr", orderServiceAddr))
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("http server stopped with error", slog.Any("error", serveErr))
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down gateway-service")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown http server", slog.Any("error", err))
	}
}

func newLogger(serviceName string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})).With(
		slog.String("service", serviceName),
	)
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}
