package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
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

	productRepository, closeRepository, err := newCatalogRepository(log)
	if err != nil {
		log.Error("failed to configure catalog repository", slog.Any("error", err))
		os.Exit(1)
	}
	defer closeRepository()

	service := service.NewCatalogService(productRepository)
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

func newCatalogRepository(logger *slog.Logger) (repository.ProductRepository, func(), error) {
	databaseURL := strings.TrimSpace(getenv("CATALOG_DATABASE_URL", ""))
	if databaseURL == "" {
		return nil, nil, errors.New("CATALOG_DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Info("configured postgres catalog repository")

	return repository.NewPostgresRepository(pool), pool.Close, nil
}
