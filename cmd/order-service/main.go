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
	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/events"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/service"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/grpcmiddleware"
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

	orderRepository, closeRepository, err := newOrderRepository(log)
	if err != nil {
		log.Error("failed to configure order repository", slog.Any("error", err))
		os.Exit(1)
	}
	defer closeRepository()

	publisher, closePublisher, err := newOrderEventPublisher(log)
	if err != nil {
		log.Error("failed to configure order event publisher", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePublisher(); err != nil {
			log.Error("failed to close order event publisher", slog.Any("error", err))
		}
	}()

	service := service.NewOrderService(orderRepository, publisher)
	grpcHandler := handler.NewGRPCHandler(service, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.RequestIDUnaryServerInterceptor()),
	)
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

func newOrderRepository(logger *slog.Logger) (repository.OrderRepository, func(), error) {
	databaseURL := strings.TrimSpace(getenv("ORDER_DATABASE_URL", ""))
	if databaseURL == "" {
		return nil, nil, errors.New("ORDER_DATABASE_URL is required")
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

	logger.Info("configured postgres order repository")

	return repository.NewPostgresRepository(pool), pool.Close, nil
}

func newOrderEventPublisher(logger *slog.Logger) (events.Publisher, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_ORDER_CREATED_TOPIC", "orders.created"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}

	if topic == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CREATED_TOPIC is required")
	}

	publisher := events.NewKafkaPublisher(brokers, topic, logger)
	logger.Info(
		"configured kafka order event publisher",
		slog.Any("brokers", brokers),
		slog.String("topic", topic),
	)
	return publisher, publisher.Close, nil
}

func parseKafkaBrokers(raw string) []string {
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}
