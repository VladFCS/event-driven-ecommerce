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

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
	mongoURI := strings.TrimSpace(getenv("CATALOG_MONGO_URI", "mongodb://localhost:27017"))
	if mongoURI == "" {
		return nil, nil, errors.New("CATALOG_MONGO_URI is required")
	}

	databaseName := strings.TrimSpace(getenv("CATALOG_MONGO_DATABASE", "ecommerce"))
	if databaseName == "" {
		return nil, nil, errors.New("CATALOG_MONGO_DATABASE is required")
	}

	collectionName := strings.TrimSpace(getenv("CATALOG_MONGO_COLLECTION", "catalog_products"))
	if collectionName == "" {
		return nil, nil, errors.New("CATALOG_MONGO_COLLECTION is required")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("ping mongo: %w", err)
	}

	collection := client.Database(databaseName).Collection(collectionName)
	closeRepository := func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := client.Disconnect(shutdownCtx); err != nil {
			logger.Error("failed to disconnect mongo client", slog.Any("error", err))
		}
	}

	logger.Info(
		"configured mongo catalog repository",
		slog.String("database", databaseName),
		slog.String("collection", collectionName),
	)

	return repository.NewMongoRepository(collection), closeRepository, nil
}
