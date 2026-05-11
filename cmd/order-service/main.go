package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := NewLogger("order-service")
	grpcPort := getenv("GRPC_PORT", "50054")

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	now := time.Now()
	repository := repository.NewMemoryRepository([]domain.Order{
		{
			ID:         "ord-100",
			CustomerID: "cust-100",
			Items: []domain.OrderItem{
				{
					ProductID:   "p-100",
					SKU:         "kbd-100",
					ProductName: "Mechanical Keyboard",
					Quantity:    1,
					UnitPrice: domain.Money{
						Currency:    orderv1.Currency_CURRENCY_USD,
						AmountCents: 12999,
					},
					TotalPrice: domain.Money{
						Currency:    orderv1.Currency_CURRENCY_USD,
						AmountCents: 12999,
					},
				},
			},
			TotalAmount: domain.Money{
				Currency:    orderv1.Currency_CURRENCY_USD,
				AmountCents: 12999,
			},
			Status: orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
			ShippingAddress: domain.Address{
				Country:    "US",
				City:       "New York",
				Street:     "5th Avenue",
				PostalCode: "10001",
				House:      "1A",
				Apartment:  "10",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:         "ord-200",
			CustomerID: "cust-200",
			Items: []domain.OrderItem{
				{
					ProductID:   "p-200",
					SKU:         "mse-200",
					ProductName: "Wireless Mouse",
					Quantity:    2,
					UnitPrice: domain.Money{
						Currency:    orderv1.Currency_CURRENCY_EUR,
						AmountCents: 5999,
					},
					TotalPrice: domain.Money{
						Currency:    orderv1.Currency_CURRENCY_EUR,
						AmountCents: 11998,
					},
				},
			},
			TotalAmount: domain.Money{
				Currency:    orderv1.Currency_CURRENCY_EUR,
				AmountCents: 11998,
			},
			Status: orderv1.OrderStatus_ORDER_STATUS_CANCELLED,
			ShippingAddress: domain.Address{
				Country:    "DE",
				City:       "Berlin",
				Street:     "Unter den Linden",
				PostalCode: "10117",
				House:      "7",
				Apartment:  "",
			},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now,
		},
	})

	service := service.NewOrderService(repository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	server := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(server, grpcHandler)

	reflection.Register(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("order-service started", slog.String("grpc_port", grpcPort))
		if serveErr := server.Serve(lis); serveErr != nil {
			log.Error("grpc server stopped with error", slog.Any("error", serveErr))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down order-service")
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
