package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewLogger(serviceName string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})).With(
		slog.String("service", serviceName),
	)
}

func main() {
	log := NewLogger("inventory-service")
	grpcPort := getenv("GRPC_PORT", "50052")

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	repository := repository.NewMemoryRepository([]domain.Stock{
		{
			ProductID:         "p-100",
			AvailableQuantity: 20,
			ReservedQuantity:  0,
			TotalQuantity:     20,
		},
		{
			ProductID:         "p-200",
			AvailableQuantity: 35,
			ReservedQuantity:  0,
			TotalQuantity:     35,
		},
	})

	service := service.NewInventoryService(repository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(handler.RequestIDUnaryServerInterceptor()),
	)
	inventoryv1.RegisterInventoryServiceServer(server, grpcHandler)

	reflection.Register(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("inventory-service started", slog.String("grpc_port", grpcPort))
		if serveErr := server.Serve(lis); serveErr != nil {
			log.Error("grpc server stopped with error", slog.Any("error", serveErr))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down inventory-service")
	server.GracefulStop()
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
