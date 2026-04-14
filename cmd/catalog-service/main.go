package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := NewLogger("catalog-service")
	grpcPort := getenv("GRPC_PORT", "50051")

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	repository := repository.NewMemoryRepository([]domain.Product{
		{
			ID:          "p-100",
			Name:        "Mechanical Keyboard",
			Description: "Hot-swappable mechanical keyboard",
			PriceCents:  12999,
			Currency:    catalogv1.Currency_CURRENCY_USD,
		},
		{
			ID:          "p-200",
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse",
			PriceCents:  5999,
			Currency:    catalogv1.Currency_CURRENCY_USD,
		},
	})

	service := service.NewCatalogService(repository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	server := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(server, grpcHandler)

	reflection.Register(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("catalog-service started", slog.String("grpc_port", grpcPort))
		if serveErr := server.Serve(lis); serveErr != nil {
			log.Error("grpc server stopped with error", slog.Any("error", serveErr))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down catalog-service")
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
