package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := NewLogger("payment-service")
	grpcPort := getenv("GRPC_PORT", "50053")

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	now := time.Now()
	repository := repository.NewMemoryRepository([]domain.Payment{
		{
			ID:             "pay-100",
			OrderID:        "ord-100",
			CustomerID:     "cust-100",
			Amount:         domain.Money{Currency: paymentv1.Currency_CURRENCY_USD, AmountCents: 12999},
			PaymentMethod:  paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD,
			IdempotencyKey: "idem-pay-100",
			Status:         paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             "pay-200",
			OrderID:        "ord-200",
			CustomerID:     "cust-200",
			Amount:         domain.Money{Currency: paymentv1.Currency_CURRENCY_EUR, AmountCents: 5999},
			PaymentMethod:  paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH,
			IdempotencyKey: "idem-pay-200",
			Status:         paymentv1.PaymentStatus_PAYMENT_STATUS_CAPTURED,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	})

	service := service.NewPaymentService(repository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	server := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(server, grpcHandler)

	reflection.Register(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("payment-service started", slog.String("grpc_port", grpcPort))
		if serveErr := server.Serve(lis); serveErr != nil {
			log.Error("grpc server stopped with error", slog.Any("error", serveErr))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down payment-service")
	server.GracefulStop()
}

func NewLogger(serviceName string) *slog.Logger {
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
