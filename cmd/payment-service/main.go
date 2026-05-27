package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/events"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/service"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/grpcmiddleware"
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

	paymentRepository, closeRepository, err := newPaymentRepository(log)
	if err != nil {
		log.Error("failed to configure payment repository", slog.Any("error", err))
		os.Exit(1)
	}
	defer closeRepository()

	publisher, closePublisher, err := newPaymentEventPublisher(log)
	if err != nil {
		log.Error("failed to configure payment event publisher", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePublisher(); err != nil {
			log.Error("failed to close payment event publisher", slog.Any("error", err))
		}
	}()

	service := service.NewPaymentService(paymentRepository)
	grpcHandler := handler.NewGRPCHandler(service, publisher, log)

	deduplicator, closeDeduplicator, err := newPaymentEventDeduplicator(log)
	if err != nil {
		log.Error("failed to configure payment event deduplicator", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeDeduplicator(); err != nil {
			log.Error("failed to close payment event deduplicator", slog.Any("error", err))
		}
	}()

	consumer, closeConsumer, err := newInventoryReservedConsumer(service, deduplicator, publisher, log)
	if err != nil {
		log.Error("failed to configure inventory.reserved consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeConsumer(); err != nil {
			log.Error("failed to close inventory.reserved consumer", slog.Any("error", err))
		}
	}()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.RequestIDUnaryServerInterceptor()),
	)
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

	go func() {
		log.Info("starting inventory.reserved consumer")
		if runErr := consumer.Run(ctx); runErr != nil && ctx.Err() == nil {
			log.Error("inventory.reserved consumer stopped with error", slog.Any("error", runErr))
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

func newPaymentRepository(logger *slog.Logger) (repository.PaymentRepository, func(), error) {
	databaseURL := strings.TrimSpace(getenv("PAYMENT_DATABASE_URL", ""))
	if databaseURL == "" {
		return nil, nil, errors.New("PAYMENT_DATABASE_URL is required")
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

	logger.Info("configured postgres payment repository")

	return repository.NewPostgresRepository(pool), pool.Close, nil
}

func newPaymentEventPublisher(logger *slog.Logger) (events.Publisher, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	createdTopic := strings.TrimSpace(getenv("KAFKA_PAYMENT_CREATED_TOPIC", "payment.created"))
	failedTopic := strings.TrimSpace(getenv("KAFKA_PAYMENT_CREATION_FAILED_TOPIC", "payment.creation_failed"))
	capturedTopic := strings.TrimSpace(getenv("KAFKA_PAYMENT_CAPTURED_TOPIC", "payment.captured"))
	paymentFailedTopic := strings.TrimSpace(getenv("KAFKA_PAYMENT_FAILED_TOPIC", "payment.failed"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if createdTopic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CREATED_TOPIC is required")
	}
	if failedTopic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CREATION_FAILED_TOPIC is required")
	}
	if capturedTopic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CAPTURED_TOPIC is required")
	}
	if paymentFailedTopic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_FAILED_TOPIC is required")
	}

	publisher := events.NewKafkaPublisher(
		brokers,
		createdTopic,
		failedTopic,
		capturedTopic,
		paymentFailedTopic,
		logger,
	)

	return publisher, publisher.Close, nil
}

func newPaymentEventDeduplicator(logger *slog.Logger) (events.EventDeduplicator, func() error, error) {
	addr := strings.TrimSpace(getenv("REDIS_ADDR", "localhost:6379"))
	password := getenv("REDIS_PASSWORD", "")
	rawDB := strings.TrimSpace(getenv("REDIS_DB", "0"))
	rawTTL := strings.TrimSpace(getenv("REDIS_EVENT_DEDUP_TTL", "24h"))

	if addr == "" {
		return nil, nil, errors.New("REDIS_ADDR is required")
	}

	db, err := strconv.Atoi(rawDB)
	if err != nil {
		return nil, nil, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	ttl, err := time.ParseDuration(rawTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse REDIS_EVENT_DEDUP_TTL: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.Info(
		"configured redis payment event deduplicator",
		slog.String("addr", addr),
		slog.Int("db", db),
		slog.String("ttl", ttl.String()),
	)

	return events.NewRedisEventDeduplicator(client, ttl), client.Close, nil
}

func newInventoryReservedConsumer(
	service *service.PaymentService,
	deduplicator events.EventDeduplicator,
	publisher events.Publisher,
	logger *slog.Logger,
) (*events.InventoryReservedConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_INVENTORY_RESERVED_TOPIC", "inventory.reserved"))
	groupID := strings.TrimSpace(getenv("KAFKA_PAYMENT_CONSUMER_GROUP", "payment-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_RESERVED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CONSUMER_GROUP is required")
	}

	consumer := events.NewInventoryReservedConsumer(brokers, topic, groupID, service, deduplicator, publisher, logger)
	return consumer, consumer.Close, nil
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
