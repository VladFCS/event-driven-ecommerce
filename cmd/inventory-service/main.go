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
	inventoryv1 "github.com/vladfc/event-driven-ecommerce-app/gen/inventory/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/events"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/service"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/grpcmiddleware"
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

	pool, closePool, err := newInventoryDatabasePool(log)
	if err != nil {
		log.Error("failed to configure inventory repository", slog.Any("error", err))
		os.Exit(1)
	}
	defer closePool()

	inventoryRepository := repository.NewPostgresRepository(pool)

	service := service.NewInventoryService(inventoryRepository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	publisher, closePublisher, err := newInventoryEventPublisher(log)
	if err != nil {
		log.Error("failed to configure inventory event publisher", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePublisher(); err != nil {
			log.Error("failed to close inventory event publisher", slog.Any("error", err))
		}
	}()

	deduplicator, closeDeduplicator, err := newInventoryEventDeduplicator(log)
	if err != nil {
		log.Error("failed to configure inventory event deduplicator", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeDeduplicator(); err != nil {
			log.Error("failed to close inventory event deduplicator", slog.Any("error", err))
		}
	}()

	consumer, closeConsumer, err := newOrderCreatedConsumer(service, deduplicator, publisher, log)
	if err != nil {
		log.Error("failed to configure order.created consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeConsumer(); err != nil {
			log.Error("failed to close order.created consumer", slog.Any("error", err))
		}
	}()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.RequestIDUnaryServerInterceptor()),
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

	go func() {
		log.Info("starting order.created consumer")
		if runErr := consumer.Run(ctx); runErr != nil && ctx.Err() == nil {
			log.Error("order.created consumer stopped with error", slog.Any("error", runErr))
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

func newInventoryDatabasePool(logger *slog.Logger) (*pgxpool.Pool, func(), error) {
	databaseURL := strings.TrimSpace(getenv("INVENTORY_DATABASE_URL", ""))
	if databaseURL == "" {
		return nil, nil, errors.New("INVENTORY_DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("connected to inventory database")

	return pool, pool.Close, nil
}

func newInventoryEventPublisher(logger *slog.Logger) (events.Publisher, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	reservedTopic := strings.TrimSpace(getenv("KAFKA_INVENTORY_RESERVED_TOPIC", "inventory.reserved"))
	failedTopic := strings.TrimSpace(getenv("KAFKA_INVENTORY_RESERVATION_FAILED_TOPIC", "inventory.reservation_failed"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if reservedTopic == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_RESERVED_TOPIC is required")
	}
	if failedTopic == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_RESERVATION_FAILED_TOPIC is required")
	}

	publisher := events.NewKafkaPublisher(brokers, reservedTopic, failedTopic, logger)
	return publisher, publisher.Close, nil
}

func newOrderCreatedConsumer(
	service *service.InventoryService,
	deduplicator events.EventDeduplicator,
	publisher events.Publisher,
	logger *slog.Logger,
) (*events.OrderCreatedConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_ORDER_CREATED_TOPIC", "orders.created"))
	groupID := strings.TrimSpace(getenv("KAFKA_INVENTORY_CONSUMER_GROUP", "inventory-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CREATED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_CONSUMER_GROUP is required")
	}

	consumer := events.NewOrderCreatedConsumer(brokers, topic, groupID, service, deduplicator, publisher, logger)
	return consumer, consumer.Close, nil
}

func newInventoryEventDeduplicator(logger *slog.Logger) (events.EventDeduplicator, func() error, error) {
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
		"configured redis inventory event deduplicator",
		slog.String("addr", addr),
		slog.Int("db", db),
		slog.String("ttl", ttl.String()),
	)

	return events.NewRedisEventDeduplicator(client, ttl), client.Close, nil
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
